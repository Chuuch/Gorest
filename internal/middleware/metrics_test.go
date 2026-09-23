package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chuuch/gorest/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestMetrics_IncrementsRequestCounter(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", middleware.MetricsHandler())

	handler := middleware.Metrics(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	handler.ServeHTTP(metricsRec, metricsReq)

	require.Equal(t, http.StatusOK, metricsRec.Code)
	body := metricsRec.Body.String()
	require.Contains(t, body, "http_requests_total")
	require.Contains(t, body, "http_request_duration_seconds")
	require.Contains(t, body, `route="GET /api/v1/health"`)
	require.Contains(t, body, `status="200"`)
}

func TestMetricsHandler_IsPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	middleware.MetricsHandler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "go_goroutines")
}
