package server

import (
	"net/http"

	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	"github.com/chuuch/gorest/internal/auth/security"
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	"github.com/chuuch/gorest/internal/middleware"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	timeentryhandler "github.com/chuuch/gorest/internal/timeentries/handler"
	userhandler "github.com/chuuch/gorest/internal/user/handler"
)

func newRouter(
	userHandler *userhandler.Handler,
	authHandler *authhandler.Handler,
	orgHandler *orghandler.Handler,
	clientHandler *clienthandler.Handler,
	projectHandler *projecthandler.Handler,
	taskHandler *taskhandler.Handler,
	timeEntryHandler *timeentryhandler.Handler,
	fileHandler *filehandler.Handler,
	commentHandler *commenthandler.Handler,
	tokenManager security.TokenManager,
) *http.ServeMux {
	mux := http.NewServeMux()

	registerRoutes(
		mux,
		userHandler,
		authHandler,
		orgHandler,
		clientHandler,
		projectHandler,
		taskHandler,
		timeEntryHandler,
		fileHandler,
		commentHandler,
		tokenManager,
	)

	return mux
}

func registerRoutes(
	mux *http.ServeMux,
	userHandler *userhandler.Handler,
	authHandler *authhandler.Handler,
	orgHandler *orghandler.Handler,
	clientHandler *clienthandler.Handler,
	projectHandler *projecthandler.Handler,
	taskHandler *taskhandler.Handler,
	timeEntryHandler *timeentryhandler.Handler,
	fileHandler *filehandler.Handler,
	commentHandler *commenthandler.Handler,
	tokenManager security.TokenManager,
) {
	// HEALTH
	mux.HandleFunc("GET /api/v1/health", healthHandler)

	// USERS
	mux.Handle("GET /api/v1/auth/me", middleware.Auth(tokenManager)(http.HandlerFunc(authHandler.Me)))
	mux.Handle("GET /api/v1/users/{id}", middleware.Auth(tokenManager)(http.HandlerFunc(userHandler.GetByID)))
	mux.Handle("PUT /api/v1/users/{id}", middleware.Auth(tokenManager)(http.HandlerFunc(userHandler.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", middleware.Auth(tokenManager)(http.HandlerFunc(userHandler.Delete)))

	// MEMBERS
	mux.Handle("GET /api/v1/members", middleware.Auth(tokenManager)(http.HandlerFunc(orgHandler.ListMembers)))
	mux.Handle("POST /api/v1/members", middleware.Auth(tokenManager)(http.HandlerFunc(orgHandler.CreateMember)))

	// CLIENTS
	mux.Handle("GET /api/v1/clients", middleware.Auth(tokenManager)(http.HandlerFunc(clientHandler.List)))
	mux.Handle("POST /api/v1/clients", middleware.Auth(tokenManager)(http.HandlerFunc(clientHandler.Create)))

	// PROJECTS
	mux.Handle("GET /api/v1/clients/{id}/projects", middleware.Auth(tokenManager)(http.HandlerFunc(projectHandler.List)))
	mux.Handle("POST /api/v1/clients/{id}/projects", middleware.Auth(tokenManager)(http.HandlerFunc(projectHandler.Create)))

	// TASKS
	mux.Handle("GET /api/v1/projects/{id}/tasks", middleware.Auth(tokenManager)(http.HandlerFunc(taskHandler.List)))
	mux.Handle("POST /api/v1/projects/{id}/tasks", middleware.Auth(tokenManager)(http.HandlerFunc(taskHandler.Create)))

	// TIME ENTRIES
	mux.Handle("GET /api/v1/tasks/{id}/tasks", middleware.Auth(tokenManager)(http.HandlerFunc(timeEntryHandler.List)))
	mux.Handle("POST /api/v1/tasks/{id}/tasks", middleware.Auth(tokenManager)(http.HandlerFunc(timeEntryHandler.Create)))

	// FILES
	mux.Handle("GET /api/v1/projects/{id}/files", middleware.Auth(tokenManager)(http.HandlerFunc(fileHandler.List)))
	mux.Handle("POST /api/v1/projects/{id}/files", middleware.Auth(tokenManager)(http.HandlerFunc(fileHandler.Create)))

	// COMMENTS
	mux.Handle("GET /api/v1/tasks/{id}/comments", middleware.Auth(tokenManager)(http.HandlerFunc(commentHandler.List)))
	mux.Handle("POST /api/v1/tasks/{id}/comments", middleware.Auth(tokenManager)(http.HandlerFunc(commentHandler.Create)))
	mux.Handle("PATCH /api/v1/comments/{id}", middleware.Auth(tokenManager)(http.HandlerFunc(commentHandler.Update)))
	mux.Handle("DELETE /api/v1/comments/{id}", middleware.Auth(tokenManager)(http.HandlerFunc(commentHandler.Delete)))

	// AUTH
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
