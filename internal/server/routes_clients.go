package server

import "net/http"

func (a *wiredApp) registerClients(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/clients", a.staff(a.clientHandler.List))
	mux.Handle("POST /api/v1/clients", a.staffIdempotent(a.clientHandler.Create))
	mux.Handle("PATCH /api/v1/clients/{id}", a.staff(a.clientHandler.Update))
	mux.Handle("DELETE /api/v1/clients/{id}", a.staff(a.clientHandler.Delete))

	mux.Handle("GET /api/v1/clients/{id}/projects", a.staff(a.projectHandler.List))
	mux.Handle("POST /api/v1/clients/{id}/projects", a.staffIdempotent(a.projectHandler.Create))
	mux.Handle("PATCH /api/v1/projects/{id}", a.staff(a.projectHandler.Update))
	mux.Handle("DELETE /api/v1/projects/{id}", a.staff(a.projectHandler.Delete))

	mux.Handle("GET /api/v1/clients/{id}/users", a.staff(a.clientUserHandler.List))
	mux.Handle("POST /api/v1/clients/{id}/users", a.staffIdempotent(a.clientUserHandler.Create))
	mux.Handle("DELETE /api/v1/clients/{id}/users/{userId}", a.staff(a.clientUserHandler.Delete))

	mux.Handle("GET /api/v1/clients/{id}/invoices", a.staff(a.invoiceHandler.List))
	mux.Handle("GET /api/v1/invoices/{id}/pdf", a.staff(a.invoiceHandler.PDF))
	mux.Handle("POST /api/v1/clients/{id}/invoices", a.staffIdempotent(a.invoiceHandler.Create))
	mux.Handle("GET /api/v1/invoices/{id}", a.staff(a.invoiceHandler.Get))
	mux.Handle("PATCH /api/v1/invoices/{id}", a.staff(a.invoiceHandler.Update))
	mux.Handle("DELETE /api/v1/invoices/{id}", a.staff(a.invoiceHandler.Delete))
	mux.Handle("POST /api/v1/invoices/{id}/send", a.staffIdempotent(a.invoiceHandler.Send))
	mux.Handle("POST /api/v1/invoices/{id}/paid", a.staffIdempotent(a.invoiceHandler.MarkPaid))
}
