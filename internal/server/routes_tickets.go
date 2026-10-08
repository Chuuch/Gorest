package server

import "net/http"

func (a *wiredApp) registerTickets(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/clients/{id}/tickets", a.staff(a.ticketHandler.List))
	mux.Handle("PATCH /api/v1/tickets/{id}", a.staff(a.ticketHandler.Update))
	mux.Handle("DELETE /api/v1/tickets/{id}", a.staff(a.ticketHandler.Delete))

	mux.Handle("GET /api/v1/tickets/{id}/files", a.staff(a.ticketFileHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/files", a.staffIdempotent(a.ticketFileHandler.Create))
	mux.Handle("DELETE /api/v1/ticket-files/{id}", a.staff(a.ticketFileHandler.Delete))
	mux.Handle("DELETE /api/v1/client-auth/ticket-files/{id}", a.staff(a.ticketFileHandler.DeletePortal))

	mux.Handle("GET /api/v1/tickets/{id}/comments", a.staff(a.ticketCommentHandler.List))
	mux.Handle("POST /api/v1/tickets/{id}/comments", a.staff(a.ticketCommentHandler.Create))
	mux.Handle("PATCH /api/v1/ticket-comments/{id}", a.staff(a.ticketCommentHandler.Update))
	mux.Handle("DELETE /api/v1/ticket-comments/{id}", a.staff(a.ticketCommentHandler.Delete))
}
