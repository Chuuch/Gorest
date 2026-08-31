package server

import (
	"net/http"

	"github.com/chuuch/gorest/internal/user"
)

func newRouter(userHandler *user.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	registerRoutes(mux, userHandler)
	return mux
}

func registerRoutes(
	mux *http.ServeMux,
	userHandler *user.Handler,
) {
	mux.HandleFunc(
		"GET /api/v1/health",
		healthHandler,
	)

	mux.HandleFunc(
		"POST /api/v1/users",
		userHandler.Create,
	)

	mux.HandleFunc(
		"GET /api/v1/users/{id}",
		userHandler.GetByID,
	)

	mux.HandleFunc(
		"PUT /api/v1/users/{id}",
		userHandler.Update,
	)

	mux.HandleFunc(
		"DELETE /api/v1/users/{id}",
		userHandler.Delete,
	)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
