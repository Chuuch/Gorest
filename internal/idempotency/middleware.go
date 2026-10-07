package idempotency

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
)

const (
	HeaderKey  = "Idempotency-Key"
	maxKeyLen  = 255
	defaultTTL = 24 * time.Hour
)

type Middleware struct {
	store Store
	ttl   time.Duration
	now   func() time.Time
}

func NewMiddleware(store Store) *Middleware {
	return &Middleware{
		store: store,
		ttl:   defaultTTL,
		now:   time.Now,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get(HeaderKey))
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !validKey(key) {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_idempotency_key",
				"idempotency key must be 1-255 printable characters",
			)
			return
		}

		organizationID, ok := requestcontext.OrganizationID(r.Context())
		if !ok {
			api.WriteError(
				w,
				http.StatusUnauthorized,
				"unauthorized",
				"organization context required",
			)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_body",
				"could not read request body",
			)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))

		path := r.URL.Path
		hash := RequestHash(r.Method, path, body)
		expiresAt := m.now().UTC().Add(m.ttl)

		existing, err := m.store.TryClaim(
			r.Context(),
			organizationID,
			key,
			r.Method,
			path,
			hash,
			expiresAt,
		)
		if errors.Is(err, ErrKeyMismatch) {
			api.WriteError(
				w,
				http.StatusConflict,
				"idempotency_key_reuse",
				"idempotency key already used with a different request",
			)
			return
		}
		if errors.Is(err, ErrKeyInProgress) {
			api.WriteError(
				w,
				http.StatusConflict,
				"idempotency_key_in_progress",
				"a request with this idempotency key is already in progress",
			)
			return
		}
		if err != nil {
			api.WriteError(
				w,
				http.StatusInternalServerError,
				"internal_error",
				"could not process idempotency key",
			)
			return
		}

		if existing != nil {
			replay(w, existing)
			return
		}

		cw := &captureWriter{ResponseWriter: w, status: http.StatusOK}
		completed := false
		defer func() {
			if !completed {
				_ = m.store.Release(r.Context(), organizationID, key)
			}
		}()

		next.ServeHTTP(cw, r)

		if err := m.store.Complete(
			r.Context(),
			organizationID,
			key,
			cw.status,
			cw.body.Bytes(),
		); err != nil {
			_ = m.store.Release(r.Context(), organizationID, key)
			return
		}
		completed = true
	})
}

func (m *Middleware) Handler(next http.HandlerFunc) http.Handler {
	return m.Wrap(next)
}

func replay(w http.ResponseWriter, rec *Record) {
	if ct := w.Header().Get("Content-Type"); ct == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Header().Set("Idempotent-Replay", "true")
	status := http.StatusOK
	if rec.StatusCode != nil {
		status = *rec.StatusCode
	}
	w.WriteHeader(status)
	if len(rec.ResponseBody) > 0 {
		_, _ = w.Write(rec.ResponseBody)
	}
}

func validKey(key string) bool {
	if key == "" || utf8.RuneCountInString(key) > maxKeyLen {
		return false
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

type captureWriter struct {
	http.ResponseWriter
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func (w *captureWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	_, _ = w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
