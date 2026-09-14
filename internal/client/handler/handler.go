package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	"github.com/chuuch/gorest/internal/client/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
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
	organizationID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	clients, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]clientdomain.ClientResponse, 0, len(clients))
	for _, client := range clients {
		responses = append(responses, toResponse(client))
	}

	api.WriteJSON(w, http.StatusOK, responses)
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
