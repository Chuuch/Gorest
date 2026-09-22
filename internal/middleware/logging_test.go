package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chuuch/gorest/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequestLog_WritesJSONFields(t *testing.T) {
	var buffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.RequestLog(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &entry))
	require.Equal(t, "http_request", entry["msg"])
	require.Equal(t, "GET", entry["method"])
	require.Equal(t, "GET /api/v1/health", entry["route"])
	require.Equal(t, float64(http.StatusOK), entry["status"])
	require.Contains(t, entry, "duration_ms")
}

func TestRequestLog_SkipsMetrics(t *testing.T) {
	var buffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequestLog(mux)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, buffer.String())
}
