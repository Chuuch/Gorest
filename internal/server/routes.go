package server

import (
	"net/http"
	"time"

	activityhandler "github.com/chuuch/gorest/internal/activity/handler"
	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	"github.com/chuuch/gorest/internal/auth/security"
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	clientuserhandler "github.com/chuuch/gorest/internal/clientusers/handler"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	invoicehandler "github.com/chuuch/gorest/internal/invoices/handler"
	"github.com/chuuch/gorest/internal/middleware"
	notificationhandler "github.com/chuuch/gorest/internal/notifications/handler"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	reporthandler "github.com/chuuch/gorest/internal/reports/handler"
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
	activityHandler *activityhandler.Handler,
	reportHandler *reporthandler.Handler,
	notificationHandler *notificationhandler.Handler,
	invoiceHandler *invoicehandler.Handler,
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
		activityHandler,
		reportHandler,
		notificationHandler,
		invoiceHandler,
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
	activityHandler *activityhandler.Handler,
	reportHandler *reporthandler.Handler,
	notificationHandler *notificationhandler.Handler,
	invoiceHandler *invoicehandler.Handler,
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
	mux.Handle("PATCH /api/v1/organization", staff(tokenManager, orgHandler.Update))

	// ACTIVITY
	mux.Handle("GET /api/v1/activity", staff(tokenManager, activityHandler.List))
	mux.Handle("GET /api/v1/reports/time", staff(tokenManager, reportHandler.Time))
	mux.Handle("GET /api/v1/notifications", staff(tokenManager, notificationHandler.List))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", staff(tokenManager, notificationHandler.MarkRead))

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
	mux.Handle("DELETE /api/v1/clients/{id}/users/{userId}", staff(tokenManager, clientUserHandler.Delete))

	// INVOICES
	mux.Handle("GET /api/v1/clients/{id}/invoices", staff(tokenManager, invoiceHandler.List))
	mux.Handle("GET /api/v1/invoices/{id}/pdf", staff(tokenManager, invoiceHandler.PDF))
	mux.Handle("POST /api/v1/clients/{id}/invoices", staff(tokenManager, invoiceHandler.Create))
	mux.Handle("GET /api/v1/invoices/{id}", staff(tokenManager, invoiceHandler.Get))
	mux.Handle("PATCH /api/v1/invoices/{id}", staff(tokenManager, invoiceHandler.Update))
	mux.Handle("DELETE /api/v1/invoices/{id}", staff(tokenManager, invoiceHandler.Delete))
	mux.Handle("POST /api/v1/invoices/{id}/send", staff(tokenManager, invoiceHandler.Send))
	mux.Handle("POST /api/v1/invoices/{id}/paid", staff(tokenManager, invoiceHandler.MarkPaid))

	// TASKS
	mux.Handle("GET /api/v1/projects/{id}/tasks", staff(tokenManager, taskHandler.List))
	mux.Handle("GET /api/v1/inbox/tasks", staff(tokenManager, taskHandler.Inbox))
	mux.Handle("POST /api/v1/projects/{id}/tasks", staff(tokenManager, taskHandler.Create))
	mux.Handle("PATCH /api/v1/tasks/{id}", staff(tokenManager, taskHandler.Update))
	mux.Handle("POST /api/v1/tickets/{id}/convert", staff(tokenManager, taskHandler.Convert))
	mux.Handle("DELETE /api/v1/tasks/{id}", staff(tokenManager, taskHandler.Delete))

	// TIME ENTRIES
	mux.Handle("GET /api/v1/time-entries", staff(tokenManager, timeEntryHandler.ListRange))
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
	mux.Handle("DELETE /api/v1/tickets/{id}", staff(tokenManager, ticketHandler.Delete))
	mux.Handle("GET /api/v1/tickets/{id}/files", staff(tokenManager, ticketFileHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/files", staff(tokenManager, ticketFileHandler.Create))
	mux.Handle("DELETE /api/v1/ticket-files/{id}", staff(tokenManager, ticketFileHandler.Delete))
	mux.Handle("DELETE /api/v1/client-auth/ticket-files/{id}", staff(tokenManager, ticketFileHandler.DeletePortal))
	mux.Handle("GET /api/v1/tickets/{id}/comments", staff(tokenManager, ticketCommentHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/comments", staff(tokenManager, ticketCommentHandler.Create))
	mux.Handle("GET /api/v1/client-auth/tickets", portal(tokenManager, ticketHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets", portal(tokenManager, ticketHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/files", portal(tokenManager, ticketFileHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets/{id}/files", portal(tokenManager, ticketFileHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/comments", portal(tokenManager, ticketCommentHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets/{id}/comments", portal(tokenManager, ticketCommentHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/notifications", portal(tokenManager, notificationHandler.List))
	mux.Handle("PATCH /api/v1/client-auth/notifications/{id}/read", portal(tokenManager, notificationHandler.MarkRead))
	mux.Handle("PATCH /api/v1/ticket-comments/{id}", staff(tokenManager, ticketCommentHandler.Update))
	mux.Handle("DELETE /api/v1/ticket-comments/{id}", staff(tokenManager, ticketCommentHandler.Delete))
	mux.Handle("PATCH /api/v1/client-auth/ticket-comments/{id}", portal(tokenManager, ticketCommentHandler.UpdatePortal))
	mux.Handle("DELETE /api/v1/client-auth/ticket-comments/{id}", portal(tokenManager, ticketCommentHandler.DeletePortal))

	// CLIENT USERS AUTH
	mux.Handle("POST /api/v1/client-auth/login", loginLimiter.Login(http.HandlerFunc(clientUserHandler.Login)))
	mux.Handle("POST /api/v1/client-auth/refresh", refreshLimiter.Refresh(http.HandlerFunc(clientUserHandler.Refresh)))
	mux.HandleFunc("POST /api/v1/client-auth/logout", clientUserHandler.Logout)
	mux.Handle("GET /api/v1/client-auth/me", portal(tokenManager, clientUserHandler.Me))
	mux.Handle("POST /api/v1/client-auth/change-password", portal(tokenManager, clientUserHandler.ChangePassword))
	mux.Handle("GET /api/v1/client-auth/invoices", portal(tokenManager, invoiceHandler.ListPortal))
	mux.Handle("GET /api/v1/client-auth/invoices/{id}", portal(tokenManager, invoiceHandler.GetPortal))
	mux.Handle("GET /api/v1/client-auth/invoices/{id}/pdf", portal(tokenManager, invoiceHandler.PDFPortal))

	// AUTH
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.Handle("POST /api/v1/auth/login", loginLimiter.Login(http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /api/v1/auth/refresh", refreshLimiter.Refresh(http.HandlerFunc(authHandler.Refresh)))
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/v1/auth/accept-invite", authHandler.AcceptInvite)
	mux.Handle("POST /api/v1/auth/forgot-password", loginLimiter.Forgot(http.HandlerFunc(authHandler.ForgotPassword)))
	mux.HandleFunc("POST /api/v1/auth/reset-password", authHandler.ResetPassword)
	mux.Handle("POST /api/v1/auth-change-password", staff(tokenManager, authHandler.ChangePassword))
	mux.Handle("PATCH /api/v1/auth/display-name", staff(tokenManager, authHandler.UpdateDisplayName))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
