package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	"github.com/chuuch/gorest/internal/tickets/usecase"
	"github.com/chuuch/gorest/internal/validation"
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

	h.writeList(w, r, organizationID, clientID)
}

func (h *Handler) ListPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	h.writeList(w, r, organizationID, clientID)
}

func (h *Handler) CreatePortal(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	var req ticketdomain.CreateTicketRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	ticket, err := h.service.Create(
		r.Context(),
		organizationID,
		clientID,
		userID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(ticket))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	var req ticketdomain.UpdateTicketRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	ticket, err := h.service.Update(
		r.Context(),
		organizationID,
		ticketID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(ticket))
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), organizationID, ticketID, actorRole); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeList(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, clientID uuid.UUID,
) {
	tickets, err := h.service.List(r.Context(), organizationID, clientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]ticketdomain.TicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		responses = append(responses, toResponse(ticket))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) clientID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
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

func (h *Handler) ticketID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_ticket_id",
			"invalid ticket id",
		)
		return uuid.Nil, false
	}
	return ticketID, true
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ticketdomain.ErrForbidden):
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)

	case errors.Is(err, ticketdomain.ErrTicketVersionMismatch):
		api.WriteError(
			w,
			http.StatusConflict,
			"ticket_version_mismatch",
			"ticket was updated by someone else",
		)

	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"client_not_found",
			"client not found",
		)

	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"ticket_not_found",
			"ticket not found",
		)

	default:
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal error",
		)
	}
}

func toResponse(ticket *ticketdomain.Ticket) ticketdomain.TicketResponse {
	return ticketdomain.TicketResponse{
		ID:             ticket.ID,
		OrganizationID: ticket.OrganizationID,
		ClientID:       ticket.ClientID,
		UserID:         ticket.UserID,
		Kind:           ticket.Kind,
		Status:         ticket.Status,
		Title:          ticket.Title,
		Body:           ticket.Body,
		Version:        ticket.Version,
		CreatedAt:      ticket.CreatedAt,
		UpdatedAt:      ticket.UpdatedAt,
	}
}
