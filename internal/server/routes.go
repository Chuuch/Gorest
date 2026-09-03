package server

import (
	"net/http"

	"github.com/chuuch/gorest/internal/auth"
	"github.com/chuuch/gorest/internal/middleware"
	"github.com/chuuch/gorest/internal/user"
)

func newRouter(
	userHandler *user.Handler,
	authHandler *auth.Handler,
	tokenManager auth.TokenManager,
) *http.ServeMux {
	mux := http.NewServeMux()

	registerRoutes(mux, userHandler, authHandler, tokenManager)
	return mux
}

func registerRoutes(
	mux *http.ServeMux,
	userHandler *user.Handler,
	authHandler *auth.Handler,
	tokenManager auth.TokenManager,
) {
	mux.HandleFunc(
		"GET /api/v1/health",
		healthHandler,
	)

	mux.Handle(
		"GET /api/v1/users/{id}",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(userHandler.GetByID),
		),
	)

	mux.Handle(
		"PUT /api/v1/users/{id}",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(userHandler.Update),
		),
	)

	mux.Handle(
		"DELETE /api/v1/users/{id}",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(userHandler.Delete),
		),
	)

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		authHandler.Register,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/refresh",
		authHandler.Refresh,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/logout",
		authHandler.Logout,
	)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
