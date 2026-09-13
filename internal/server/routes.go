package server

import (
	"net/http"

	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/chuuch/gorest/internal/middleware"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	userhandler "github.com/chuuch/gorest/internal/user/handler"
)

func newRouter(
	userHandler *userhandler.Handler,
	authHandler *authhandler.Handler,
	orghandler *orghandler.Handler,
	tokenManager security.TokenManager,
) *http.ServeMux {
	mux := http.NewServeMux()

	registerRoutes(mux, userHandler, authHandler, orghandler, tokenManager)

	return mux
}

func registerRoutes(
	mux *http.ServeMux,
	userHandler *userhandler.Handler,
	authHandler *authhandler.Handler,
	orghandler *orghandler.Handler,
	tokenManager security.TokenManager,
) {
	mux.HandleFunc(
		"GET /api/v1/health",
		healthHandler,
	)

	mux.Handle(
		"GET /api/v1/auth/me",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(authHandler.Me),
		),
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

	mux.Handle(
		"GET /api/v1/members",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(orghandler.ListMembers),
		),
	)

	mux.Handle(
		"POST /api/v1/members",
		middleware.Auth(tokenManager)(
			http.HandlerFunc(orghandler.CreateMember),
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
