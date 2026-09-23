package server

import (
	"net/http"
	"time"

	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	"github.com/chuuch/gorest/internal/auth/security"
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	clientuserhandler "github.com/chuuch/gorest/internal/clientusers/handler"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	"github.com/chuuch/gorest/internal/middleware"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	ticketcommenthandler "github.com/chuuch/gorest/internal/ticketcomments/handler"
	ticketfilehandler "github.com/chuuch/gorest/internal/ticketfiles/handler"
	tickethandler "github.com/chuuch/gorest/internal/tickets/handler"
	timeentryhandler "github.com/chuuch/gorest/internal/timeentries/handler"
	userhandler "github.com/chuuch/gorest/internal/user/handler"
)

func staff(tokenManager security.TokenManager, h http.HandlerFunc) http.Handler {
	return middleware.Auth(tokenManager)(middleware.Staff(h))
}

func portal(tokenManager security.TokenManager, h http.HandlerFunc) http.Handler {
	return middleware.Auth(tokenManager)(middleware.ClientPortal(h))
}

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
	clientUserHandler *clientuserhandler.Handler,
	ticketHandler *tickethandler.Handler,
	ticketFileHandler *ticketfilehandler.Handler,
	ticketCommentHandler *ticketcommenthandler.Handler,
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
		clientUserHandler,
		ticketHandler,
		ticketFileHandler,
		ticketCommentHandler,
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
	clientUserHandler *clientuserhandler.Handler,
	ticketHandler *tickethandler.Handler,
	ticketFileHandler *ticketfilehandler.Handler,
	ticketCommentHandler *ticketcommenthandler.Handler,
	tokenManager security.TokenManager,
) {
	loginLimiter := middleware.NewLimiter(10, 15*time.Minute)
	refreshLimiter := middleware.NewLimiter(30, 15*time.Minute)
	// HEALTH
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.Handle("GET /metrics", middleware.MetricsHandler())

	// USERS
	mux.Handle("GET /api/v1/auth/me", staff(tokenManager, authHandler.Me))
	mux.Handle("GET /api/v1/users/{id}", staff(tokenManager, userHandler.GetByID))
	mux.Handle("PUT /api/v1/users/{id}", staff(tokenManager, userHandler.Update))
	mux.Handle("DELETE /api/v1/users/{id}", staff(tokenManager, userHandler.Delete))

	// MEMBERS
	mux.Handle("GET /api/v1/members", staff(tokenManager, orgHandler.ListMembers))
	mux.Handle("POST /api/v1/members", staff(tokenManager, orgHandler.CreateMember))
	mux.Handle("PATCH /api/v1/members/{id}", staff(tokenManager, orgHandler.UpdateMember))
	mux.Handle("DELETE /api/v1/members/{id}", staff(tokenManager, orgHandler.DeleteMember))

	// CLIENTS
	mux.Handle("GET /api/v1/clients", staff(tokenManager, clientHandler.List))
	mux.Handle("POST /api/v1/clients", staff(tokenManager, clientHandler.Create))
	mux.Handle("PATCH /api/v1/clients/{id}", staff(tokenManager, clientHandler.Update))
	mux.Handle("DELETE /api/v1/clients/{id}", staff(tokenManager, clientHandler.Delete))

	// PROJECTS
	mux.Handle("GET /api/v1/clients/{id}/projects", staff(tokenManager, projectHandler.List))
	mux.Handle("POST /api/v1/clients/{id}/projects", staff(tokenManager, projectHandler.Create))
	mux.Handle("PATCH /api/v1/projects/{id}", staff(tokenManager, projectHandler.Update))
	mux.Handle("DELETE /api/v1/projects/{id}", staff(tokenManager, projectHandler.Delete))

	// CLIENT USERS
	mux.Handle("GET /api/v1/clients/{id}/users", staff(tokenManager, clientUserHandler.List))
	mux.Handle("POST /api/v1/clients/{id}/users", staff(tokenManager, clientUserHandler.Create))
	mux.Handle("DELETE /api/v1/clients/{id}/users/{id}", staff(tokenManager, clientUserHandler.Delete))

	// TASKS
	mux.Handle("GET /api/v1/projects/{id}/tasks", staff(tokenManager, taskHandler.List))
	mux.Handle("POST /api/v1/projects/{id}/tasks", staff(tokenManager, taskHandler.Create))
	mux.Handle("PATCH /api/v1/tasks/{id}", staff(tokenManager, taskHandler.Update))
	mux.Handle("POST /api/v1/tickets/{id}/convert", staff(tokenManager, taskHandler.Convert))

	// TIME ENTRIES
	mux.Handle("GET /api/v1/tasks/{id}/time-entries", staff(tokenManager, timeEntryHandler.List))
	mux.Handle("POST /api/v1/tasks/{id}/time-entries", staff(tokenManager, timeEntryHandler.Create))
	mux.Handle("PATCH /api/v1/time-entries/{id}", staff(tokenManager, timeEntryHandler.Update))
	mux.Handle("DELETE /api/v1/time-entries/{id}", staff(tokenManager, timeEntryHandler.Delete))

	// FILES
	mux.Handle("GET /api/v1/projects/{id}/files", staff(tokenManager, fileHandler.List))
	mux.Handle("POST /api/v1/projects/{id}/files", staff(tokenManager, fileHandler.Create))
	mux.Handle("DELETE /api/v1/files/{id}", staff(tokenManager, fileHandler.Delete))

	// COMMENTS
	mux.Handle("GET /api/v1/tasks/{id}/comments", staff(tokenManager, commentHandler.List))
	mux.Handle("POST /api/v1/tasks/{id}/comments", staff(tokenManager, commentHandler.Create))
	mux.Handle("PATCH /api/v1/comments/{id}", staff(tokenManager, commentHandler.Update))
	mux.Handle("DELETE /api/v1/comments/{id}", staff(tokenManager, commentHandler.Delete))

	// TICKETS
	mux.Handle("GET /api/v1/clients/{id}/tickets", staff(tokenManager, ticketHandler.List))
	mux.Handle("PATCH /api/v1/tickets/{id}", staff(tokenManager, ticketHandler.Update))
	mux.Handle("GET /api/v1/tickets/{id}/files", staff(tokenManager, ticketFileHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/files", staff(tokenManager, ticketFileHandler.Create))
	mux.Handle("GET /api/v1/tickets/{id}/comments", staff(tokenManager, ticketCommentHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/comments", staff(tokenManager, ticketCommentHandler.Create))
	mux.Handle("GET /api/v1/client-auth/tickets", portal(tokenManager, ticketHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets", portal(tokenManager, ticketHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/files", portal(tokenManager, ticketFileHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets/{id}/files", portal(tokenManager, ticketFileHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/comments", portal(tokenManager, ticketCommentHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets/{id}/comments", portal(tokenManager, ticketCommentHandler.CreatePortal))

	// CLIENT USERS AUTH
	mux.Handle("POST /api/v1/client-auth/login", loginLimiter.Login(http.HandlerFunc(clientUserHandler.Login)))
	mux.Handle("POST /api/v1/client-auth/refresh", refreshLimiter.Refresh(http.HandlerFunc(clientUserHandler.Refresh)))
	mux.HandleFunc("POST /api/v1/client-auth/logout", clientUserHandler.Logout)
	mux.Handle("GET /api/v1/client-auth/me", portal(tokenManager, clientUserHandler.Me))

	// AUTH
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.Handle("POST /api/v1/auth/login", loginLimiter.Login(http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /api/v1/auth/refresh", refreshLimiter.Refresh(http.HandlerFunc(authHandler.Refresh)))
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
