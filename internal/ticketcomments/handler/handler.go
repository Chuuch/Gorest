package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	ticketcommentdomain "github.com/chuuch/gorest/internal/ticketcomments/domain"
	"github.com/chuuch/gorest/internal/ticketcomments/usecase"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
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

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeList(w, r, organizationID, ticketID, uuid.Nil)
}

func (h *Handler) ListPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeList(w, r, organizationID, ticketID, clientID)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeCreate(w, r, organizationID, ticketID, userID, uuid.Nil)
}

func (h *Handler) CreatePortal(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeCreate(w, r, organizationID, ticketID, userID, clientID)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, actorRole, ok := h.actor(w, r)
	if !ok {
		return
	}

	h.writeUpdate(w, r, organizationID, userID, uuid.Nil, actorRole)
}

func (h *Handler) UpdatePortal(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	actorRole, ok := h.role(w, r)
	if !ok {
		return
	}

	h.writeUpdate(w, r, organizationID, userID, clientID, actorRole)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, actorRole, ok := h.actor(w, r)
	if !ok {
		return
	}

	h.writeDelete(w, r, organizationID, userID, uuid.Nil, actorRole)
}

func (h *Handler) DeletePortal(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	actorRole, ok := h.role(w, r)
	if !ok {
		return
	}

	h.writeDelete(w, r, organizationID, userID, clientID, actorRole)
}

func (h *Handler) writeList(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, ticketID, portalClientID uuid.UUID,
) {
	comments, err := h.service.List(r.Context(), organizationID, ticketID, portalClientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]ticketcommentdomain.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, toResponse(comment))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) writeCreate(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
) {
	var req ticketcommentdomain.CreateCommentRequest

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

	comment, err := h.service.Create(
		r.Context(),
		organizationID,
		ticketID,
		userID,
		portalClientID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(comment))
}

func (h *Handler) writeUpdate(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, userID, portalClientID uuid.UUID,
	actorRole orgdomain.Role,
) {
	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}

	var req ticketcommentdomain.UpdateCommentRequest

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

	comment, err := h.service.Update(
		r.Context(),
		organizationID,
		commentID,
		userID,
		portalClientID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(comment))
}

func (h *Handler) writeDelete(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, userID, portalClientID uuid.UUID,
	actorRole orgdomain.Role,
) {
	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		commentID,
		userID,
		portalClientID,
		actorRole,
	); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) commentID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	commentID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_comment_id",
			"invalid comment id",
		)
		return uuid.Nil, false
	}
	return commentID, true
}

func (h *Handler) staffSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}
	return organizationID, userID, true
}

func (h *Handler) portalSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	organizationID, userID, ok := h.staffSession(w, r)
	if !ok {
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

func (h *Handler) role(
	w http.ResponseWriter,
	r *http.Request,
) (orgdomain.Role, bool) {
	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return "", false
	}
	return orgdomain.Role(role), true
}

func (h *Handler) actor(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, orgdomain.Role, bool) {
	organizationID, userID, ok := h.staffSession(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}

	actorRole, ok := h.role(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}

	return organizationID, userID, actorRole, true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ticketcommentdomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")

	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"ticket_not_found",
			"ticket not found",
		)

	case errors.Is(err, ticketcommentdomain.ErrCommentNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"comment_not_found",
			"comment not found",
		)

	default:
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
		)
	}
}

func toResponse(comment *ticketcommentdomain.Comment) ticketcommentdomain.CommentResponse {
	return ticketcommentdomain.CommentResponse{
		ID:             comment.ID,
		OrganizationID: comment.OrganizationID,
		TicketID:       comment.TicketID,
		UserID:         comment.UserID,
		Body:           comment.Body,
		CreatedAt:      comment.CreatedAt,
		UpdatedAt:      comment.UpdatedAt,
	}
}
