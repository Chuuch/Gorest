package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"strconv"

	clientdomain "github.com/chuuch/gorest/internal/clients/domain"
	"github.com/chuuch/gorest/internal/clients/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/api"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/chuuch/gorest/internal/platform/validation"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	limit := 50
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_limit",
				"invalid limit",
			)
			return
		}
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	var cursor *pagination.Cursor
	if rawCursor := r.URL.Query().Get("cursor"); rawCursor != "" {
		decoded, err := pagination.Decode(rawCursor)
		if err != nil {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_cursor",
				"invalid cursor",
			)
			return
		}
		cursor = &decoded
	}

	clients, nextCursor, err := h.service.List(
		r.Context(),
		organizationID,
		limit,
		cursor,
		r.URL.Query().Get("q"),
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]clientdomain.ClientResponse, 0, len(clients))
	for _, client := range clients {
		responses = append(responses, toResponse(client))
	}

	api.WriteJSON(w, http.StatusOK, pagination.Page[clientdomain.ClientResponse]{
		Items:      responses,
		NextCursor: nextCursor,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	var req clientdomain.CreateClientRequest

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

	client, err := h.service.Create(
		r.Context(),
		organizationID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(client))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	var req clientdomain.UpdateClientRequest

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

	client, err := h.service.Update(
		r.Context(),
		organizationID,
		clientID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(client))
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		clientID,
		actorRole,
	); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) session(w http.ResponseWriter, r *http.Request) (uuid.UUID, orgdomain.Role, bool) {
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, clientdomain.ErrForbidden):
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)

	case errors.Is(err, clientdomain.ErrClientNameExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"client_name_already_exists",
			"client name already exists",
		)

	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"client_not_found",
			"client not found",
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

func toResponse(client *clientdomain.Client) clientdomain.ClientResponse {
	return clientdomain.ClientResponse{
		ID:             client.ID,
		OrganizationID: client.OrganizationID,
		Name:           client.Name,
		Notes:          client.Notes,
		CreatedAt:      client.CreatedAt,
		UpdatedAt:      client.UpdatedAt,
	}
}
