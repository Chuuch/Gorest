package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/chuuch/gorest/internal/projects/usecase"
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

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	projects, err := h.service.List(r.Context(), organizationID, clientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]projectdomain.ProjectResponse, 0, len(projects))
	for _, project := range projects {
		responses = append(responses, toResponse(project))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	var req projectdomain.CreateProjectRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_reques",
			"invalid request",
		)
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	project, err := h.service.Create(
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

	api.WriteJSON(w, http.StatusCreated, toResponse(project))
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

func (h *Handler) session(
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projectdomain.ErrForbidden):
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)

	case errors.Is(err, projectdomain.ErrProjectNameExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"project_name_already_exists",
			"project name already exists",
		)

	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"client_not_found",
			"client not found",
		)

	case errors.Is(err, projectdomain.ErrProjectNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"project_not_found",
			"project not found",
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

func toResponse(project *projectdomain.Project) projectdomain.ProjectResponse {
	return projectdomain.ProjectResponse {
		ID: project.ID,
		OrganizationID: project.OrganizationID,
		ClientID: project.ClientID,
		Name: project.Name,
		Notes: project.Notes,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}
