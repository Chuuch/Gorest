package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chuuch/gorest/internal/api"
)

type Limiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
	now    func() time.Time
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{
		hits:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
		now:    time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)
	recent := make([]time.Time, 0, len(l.hits[key]))

	for _, hit := range l.hits[key] {
		if hit.After(cutoff) {
			recent = append(recent, hit)
		}
	}

	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}

	l.hits[key] = append(recent, now)
	return true
}

func (l *Limiter) Login(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "login:" + clientIP(r) + ":" + peekEmail(r)
		if !l.Allow(key) {
			api.WriteError(
				w,
				http.StatusTooManyRequests,
				"rate_limited",
				"too many login attempts",
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) Forgot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "forgot:" + clientIP(r) + ":" + peekEmail(r)
		if !l.Allow(key) {
			api.WriteError(
				w,
				http.StatusTooManyRequests,
				"rate_limited",
				"too many reset attempts",
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) Refresh(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "refresh:" + clientIP(r)
		if !l.Allow(key) {
			api.WriteError(
				w,
				http.StatusTooManyRequests,
				"rate_limited",
				"too many refresh attempts",
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func peekEmail(r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Email string `json:"email"`
	}

	_ = json.Unmarshal(body, &payload)

	return strings.ToLower(strings.TrimSpace(payload.Email))
}

func clientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ip, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(ip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
