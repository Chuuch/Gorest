package handler

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	invoicedomain "github.com/chuuch/gorest/internal/invoices/domain"
	"github.com/chuuch/gorest/internal/invoices/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	invoices, err := h.service.List(r.Context(), organizationID, clientID, r.URL.Query().Get("q"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeList(w, invoices)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.Get(r.Context(), organizationID, invoiceID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(invoice))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	from, to, ok := h.rangeFromBody(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.Create(r.Context(), organizationID, clientID, actorRole, from, to)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(invoice))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	from, to, ok := h.rangeFromBody(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.Update(r.Context(), organizationID, invoiceID, actorRole, from, to)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(invoice))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), organizationID, invoiceID, actorRole); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.Send(r.Context(), organizationID, invoiceID, actorRole)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(invoice))
}

func (h *Handler) MarkPaid(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.MarkPaid(r.Context(), organizationID, invoiceID, actorRole)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(invoice))
}

func (h *Handler) PDF(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	data, filename, err := h.service.PDF(r.Context(), organizationID, invoiceID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, filename),
	)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) ListPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	invoices, err := h.service.ListPortal(r.Context(), organizationID, clientID, r.URL.Query().Get("q"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeList(w, invoices)
}

func (h *Handler) GetPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	invoice, err := h.service.GetPortal(r.Context(), organizationID, clientID, invoiceID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(invoice))
}

func (h *Handler) PDFPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	invoiceID, ok := h.invoiceID(w, r)
	if !ok {
		return
	}

	data, filename, err := h.service.PDFPortal(r.Context(), organizationID, clientID, invoiceID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) portalSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	clientID, ok := requestcontext.ClientID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return organizationID, userID, clientID, true
}

func (h *Handler) rangeFromBody(
	w http.ResponseWriter,
	r *http.Request,
) (time.Time, time.Time, bool) {
	var req invoicedomain.CreateInvoiceRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return time.Time{}, time.Time{}, false
	}

	from, fromErr := time.Parse(time.RFC3339Nano, req.From)
	to, toErr := time.Parse(time.RFC3339Nano, req.To)
	if fromErr != nil || toErr != nil || !to.After(from) {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_time_range",
			"from and to must be RFC3339 and to must be after from",
		)
		return time.Time{}, time.Time{}, false
	}

	return from, to, true
}

func (h *Handler) clientID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	clientID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_client_id",
			"invalid client id",
		)
		return uuid.Nil, false
	}
	return clientID, true
}

func (h *Handler) invoiceID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	invoiceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_invoice_id",
			"invalid invoice id",
		)
		return uuid.Nil, false
	}
	return invoiceID, true
}

func (h *Handler) staffSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, "", false
	}

	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, "", false
	}

	return organizationID, orgdomain.Role(role), true
}

func (h *Handler) writeList(w http.ResponseWriter, invoices []*invoicedomain.Invoice) {
	responses := make([]invoicedomain.InvoiceResponse, 0, len(invoices))
	for _, invoice := range invoices {
		responses = append(responses, toResponse(invoice))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invoicedomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(w, http.StatusNotFound, "client_not_found", "client not found")
	case errors.Is(err, invoicedomain.ErrInvoiceNotFound):
		api.WriteError(w, http.StatusNotFound, "invoice_not_found", "invoice not found")
	case errors.Is(err, invoicedomain.ErrNoLineItems):
		api.WriteError(w, http.StatusBadRequest, "no_line_items", "no time in this range")
	case errors.Is(err, invoicedomain.ErrNotDraft):
		api.WriteError(w, http.StatusConflict, "invoice_not_draft", "invoice is not a draft")
	case errors.Is(err, invoicedomain.ErrNotSent):
		api.WriteError(w, http.StatusConflict, "invoice_not_sent", "invoice is not sent")
	case errors.Is(err, invoicedomain.ErrNoClientUsers):
		api.WriteError(w, http.StatusConflict, "no_client_users", "no client users")
	case errors.Is(err, invoicedomain.ErrBillingProfileIncomplete):
		api.WriteError(
			w,
			http.StatusUnprocessableEntity,
			"billing_profile_incomplete",
			"complete organization and client billing details before sending",
		)
	default:
		api.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func toResponse(invoice *invoicedomain.Invoice) invoicedomain.InvoiceResponse {
	lines := make([]invoicedomain.LineItemResponse, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		lines = append(lines, invoicedomain.LineItemResponse{
			ID:          line.ID,
			ProjectName: line.ProjectName,
			TaskTitle:   line.TaskTitle,
			Minutes:     line.Minutes,
			AmountCents: line.AmountCents,
			Position:    line.Position,
		})
	}

	return invoicedomain.InvoiceResponse{
		ID:                       invoice.ID,
		OrganizationID:           invoice.OrganizationID,
		ClientID:                 invoice.ClientID,
		Number:                   invoice.Number,
		Status:                   invoice.Status,
		Currency:                 invoice.Currency,
		RateCents:                invoice.RateCents,
		OrganizationName:         invoice.OrganizationName,
		ClientName:               invoice.ClientName,
		SellerLegalName:          invoice.SellerLegalName,
		SellerRegistrationNumber: invoice.SellerRegistrationNumber,
		SellerVATID:              invoice.SellerVATID,
		SellerAddressLine1:       invoice.SellerAddressLine1,
		SellerAddressLine2:       invoice.SellerAddressLine2,
		SellerCity:               invoice.SellerCity,
		SellerPostalCode:         invoice.SellerPostalCode,
		SellerCountry:            invoice.SellerCountry,
		BuyerLegalName:           invoice.BuyerLegalName,
		BuyerVATID:               invoice.BuyerVATID,
		BuyerAddressLine1:        invoice.BuyerAddressLine1,
		BuyerAddressLine2:        invoice.BuyerAddressLine2,
		BuyerCity:                invoice.BuyerCity,
		BuyerPostalCode:          invoice.BuyerPostalCode,
		BuyerCountry:             invoice.BuyerCountry,
		VATRegime:                invoice.VATRegime,
		VATRateBPS:               invoice.VATRateBPS,
		SubtotalCents:            invoice.SubtotalCents,
		VATCents:                 invoice.VATCents,
		BankIBAN:                 invoice.BankIBAN,
		BankBIC:                  invoice.BankBIC,
		BankName:                 invoice.BankName,
		PeriodFrom:               invoice.PeriodFrom,
		PeriodTo:                 invoice.PeriodTo,
		IssuedAt:                 invoice.IssuedAt,
		DueAt:                    invoice.DueAt,
		SentAt:                   invoice.SentAt,
		PaidAt:                   invoice.PaidAt,
		TotalMinutes:             invoice.TotalMinutes,
		TotalCents:               invoice.TotalCents,
		CreatedAt:                invoice.CreatedAt,
		UpdatedAt:                invoice.UpdatedAt,
		Lines:                    lines,
	}
}
