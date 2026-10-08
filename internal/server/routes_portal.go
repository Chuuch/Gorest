package server

import (
	"net/http"

	"github.com/chuuch/gorest/internal/platform/middleware"
)

func (a *wiredApp) registerPortal(
	mux *http.ServeMux,
	loginLimiter *middleware.Limiter,
	refreshLimiter *middleware.Limiter,
) {
	mux.Handle("POST /api/v1/client-auth/login", loginLimiter.Login(http.HandlerFunc(a.clientUserHandler.Login)))
	mux.Handle(
		"POST /api/v1/client-auth/refresh",
		refreshLimiter.Refresh(http.HandlerFunc(a.clientUserHandler.Refresh)),
	)
	mux.HandleFunc("POST /api/v1/client-auth/logout", a.clientUserHandler.Logout)
	mux.Handle("GET /api/v1/client-auth/me", a.portal(a.clientUserHandler.Me))
	mux.Handle("POST /api/v1/client-auth/change-password", a.portal(a.clientUserHandler.ChangePassword))

	mux.Handle("GET /api/v1/client-auth/tickets", a.portal(a.ticketHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets", a.portalIdempotent(a.ticketHandler.CreatePortal))
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/files", a.portal(a.ticketFileHandler.ListPortal))
	mux.Handle(
		"POST /api/v1/client-auth/tickets/{id}/files",
		a.portalIdempotent(a.ticketFileHandler.CreatePortal),
	)
	mux.Handle("GET /api/v1/client-auth/tickets/{id}/comments", a.portal(a.ticketCommentHandler.ListPortal))
	mux.Handle("POST /api/v1/client-auth/tickets/{id}/comments", a.portal(a.ticketCommentHandler.CreatePortal))
	mux.Handle("PATCH /api/v1/client-auth/ticket-comments/{id}", a.portal(a.ticketCommentHandler.UpdatePortal))
	mux.Handle("DELETE /api/v1/client-auth/ticket-comments/{id}", a.portal(a.ticketCommentHandler.DeletePortal))

	mux.Handle("GET /api/v1/client-auth/notifications", a.portal(a.notificationHandler.List))
	mux.Handle("PATCH /api/v1/client-auth/notifications/{id}/read", a.portal(a.notificationHandler.MarkRead))
	mux.Handle("GET /api/v1/client-auth/nav/counts", a.portal(a.navHandler.Counts))
	mux.Handle("GET /api/v1/client-auth/invoices", a.portal(a.invoiceHandler.ListPortal))
	mux.Handle("GET /api/v1/client-auth/invoices/{id}", a.portal(a.invoiceHandler.GetPortal))
	mux.Handle("GET /api/v1/client-auth/invoices/{id}/pdf", a.portal(a.invoiceHandler.PDFPortal))
	mux.Handle("GET /api/v1/client-auth/events", a.portal(a.eventsHandler.Portal))
}
