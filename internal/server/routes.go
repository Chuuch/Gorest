package server

import "net/http"

func newRouter() *http.ServeMux {
	mux := http.NewServeMux()

	registerRoutes(mux)
	return mux
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
