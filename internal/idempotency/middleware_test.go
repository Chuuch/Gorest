package idempotency_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/idempotency"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_SkipsWithoutKey(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.NewMiddleware(store)
	calls := 0

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		api.WriteJSON(w, http.StatusCreated, map[string]string{"id": "1"})
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader(`{"name":"A"}`))
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, 1, calls)
}

func TestMiddleware_ReplaysSameKeyAndBody(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.NewMiddleware(store)
	calls := 0
	orgID := uuid.New()

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		api.WriteJSON(w, http.StatusCreated, map[string]any{"n": calls})
	}))

	body := `{"name":"Northwind"}`
	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader(body))
		req.Header.Set(idempotency.HeaderKey, "key-1")
		req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), orgID))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	first := do()
	second := do()

	require.Equal(t, http.StatusCreated, first.Code)
	require.Equal(t, http.StatusCreated, second.Code)
	require.Equal(t, first.Body.String(), second.Body.String())
	require.Equal(t, "true", second.Header().Get("Idempotent-Replay"))
	require.Equal(t, 1, calls)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &payload))
	require.Equal(t, float64(1), payload["n"])
}

func TestMiddleware_ConflictOnDifferentBody(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.NewMiddleware(store)
	orgID := uuid.New()

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteJSON(w, http.StatusCreated, map[string]string{"ok": "1"})
	}))

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader(`{"name":"A"}`))
	req1.Header.Set(idempotency.HeaderKey, "key-2")
	req1 = req1.WithContext(requestcontext.WithOrganizationID(req1.Context(), orgID))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader(`{"name":"B"}`))
	req2.Header.Set(idempotency.HeaderKey, "key-2")
	req2 = req2.WithContext(requestcontext.WithOrganizationID(req2.Context(), orgID))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusConflict, rec2.Code)
	body, err := io.ReadAll(rec2.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "idempotency_key_reuse")
}

func TestMiddleware_PreservesBodyForHandler(t *testing.T) {
	store := idempotency.NewMemoryStore()
	mw := idempotency.NewMiddleware(store)
	orgID := uuid.New()
	var got string

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		got = string(raw)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", strings.NewReader(`{"name":"Zed"}`))
	req.Header.Set(idempotency.HeaderKey, "key-3")
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), orgID))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, `{"name":"Zed"}`, got)
}
