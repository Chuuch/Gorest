package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/api"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/chuuch/gorest/internal/platform/validation"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	phasedomain "github.com/chuuch/gorest/internal/projects/phases/domain"
	"github.com/chuuch/gorest/internal/projects/phases/usecase"
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

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	phases, err := h.service.List(r.Context(), organizationID, projectID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]phasedomain.PhaseResponse, 0, len(phases))
	for _, phase := range phases {
		responses = append(responses, toResponse(phase))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) ListPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	phases, err := h.service.ListPortal(r.Context(), organizationID, clientID, projectID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]phasedomain.PhaseResponse, 0, len(phases))
	for _, phase := range phases {
		responses = append(responses, toResponse(phase))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	var req phasedomain.CreatePhaseRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	phase, err := h.service.Create(r.Context(), organizationID, projectID, actorRole, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(phase))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	phaseID, ok := h.phaseID(w, r)
	if !ok {
		return
	}

	var req phasedomain.UpdatePhaseRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	phase, err := h.service.Update(r.Context(), organizationID, phaseID, actorRole, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, toResponse(phase))
}

func (h *Handler) projectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	phaseID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_phase_id", "invalid phase id")
		return uuid.Nil, false
	}
	return phaseID, true
}

func (h *Handler) phaseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	phaseID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_phase_id", "invalid phase id")
		return uuid.Nil, false
	}
	return phaseID, true
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) (uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, "", false
	}
	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, "", false
	}
	return organizationID, orgdomain.Role(role), true
}

func (h *Handler) portalSession(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	clientID, ok := requestcontext.ClientID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	return organizationID, clientID, true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, phasedomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, phasedomain.ErrInvalidDates):
		api.WriteError(w, http.StatusBadRequest, "invalid_phase_dates", "invalid phase dates")
	case errors.Is(err, phasedomain.ErrPhaseNameExists):
		api.WriteError(w, http.StatusConflict, "phase_name_already_exists", "phase name already exists")
	case errors.Is(err, phasedomain.ErrPhaseSortExists):
		api.WriteError(w, http.StatusConflict, "phase_sort_already_exists", "phase sort order already exists")
	case errors.Is(err, phasedomain.ErrPhaseVersionMismatch):
		api.WriteError(w, http.StatusConflict, "phase_version_mismatch", "phase version mismatch")
	case errors.Is(err, projectdomain.ErrProjectNotFound):
		api.WriteError(w, http.StatusNotFound, "project_not_found", "project not found")
	case errors.Is(err, phasedomain.ErrPhaseNotFound):
		api.WriteError(w, http.StatusNotFound, "phase_not_found", "phase not found")
	default:
		api.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func toResponse(phase *phasedomain.Phase) phasedomain.PhaseResponse {
	out := phasedomain.PhaseResponse{
		ID:             phase.ID,
		OrganizationID: phase.OrganizationID,
		ProjectID:      phase.ProjectID,
		Name:           phase.Name,
		SortOrder:      phase.SortOrder,
		Status:         phase.Status,
		Version:        phase.Version,
		CreatedAt:      phase.CreatedAt,
		UpdatedAt:      phase.UpdatedAt,
	}
	if phase.StartsOn != nil {
		d := phase.StartsOn.Format("2006-01-02")
		out.StartsOn = &d
	}
	if phase.DueOn != nil {
		d := phase.DueOn.Format("2006-01-02")
		out.DueOn = &d
	}

	return out
}
