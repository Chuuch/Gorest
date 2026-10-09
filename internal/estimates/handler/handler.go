package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	clientdomain "github.com/chuuch/gorest/internal/clients/domain"
	"github.com/chuuch/gorest/internal/estimates/domain"
	"github.com/chuuch/gorest/internal/estimates/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/api"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/chuuch/gorest/internal/platform/validation"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, h.service.Catalog())
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	var req domain.PreviewRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	out, err := h.service.Preview(req.Input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, role, ok := h.session(w, r)
	if !ok {
		return
	}
	var req domain.CreateEstimateRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	run, err := h.service.Create(r.Context(), organizationID, userID, role, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, toResponse(run))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}
	id, ok := h.estimateID(w, r)
	if !ok {
		return
	}
	run, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, toResponse(run))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}
	limit, cursor, ok := pagination.LimitAndCursor(w, r, api.WriteError)
	if !ok {
		return
	}
	var clientID *uuid.UUID
	if raw := r.URL.Query().Get("client_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, "invalid_client_id", "invalid client id")
			return
		}
		clientID = &parsed
	}
	runs, next, err := h.service.List(r.Context(), organizationID, limit, cursor, clientID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	items := make([]domain.EstimateResponse, 0, len(runs))
	for _, run := range runs {
		items = append(items, toResponse(run))
	}
	api.WriteJSON(w, http.StatusOK, pagination.Page[domain.EstimateResponse]{
		Items: items, NextCursor: next,
	})
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, role, ok := h.session(w, r)
	if !ok {
		return
	}
	estimateID, ok := h.estimateID(w, r)
	if !ok {
		return
	}
	var req domain.CreateProjectFromEstimateRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}
	project, err := h.service.CreateProject(r.Context(), organizationID, estimateID, userID, role, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	// Reuse projects toResponse if exported; else minimal map:
	api.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":              project.ID,
		"organization_id": project.OrganizationID,
		"client_id":       project.ClientID,
		"name":            project.Name,
		"notes":           project.Notes,
		"estimate_run_id": project.EstimateRunID,
		"estimated_hours": project.EstimatedHours,
		"created_at":      project.CreatedAt,
		"updated_at":      project.UpdatedAt,
	})
}

func (h *Handler) estimateID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_estimate_id", "invalid estimate id")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, "", false
	}
	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, "", false
	}
	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, "", false
	}
	return organizationID, userID, orgdomain.Role(role), true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, domain.ErrEstimateNotFound):
		api.WriteError(w, http.StatusNotFound, "estimate_not_found", "estimate not found")
	case errors.Is(err, domain.ErrInvalidInput):
		api.WriteError(w, http.StatusBadRequest, "invalid_estimate_input", err.Error())
	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(w, http.StatusNotFound, "client_not_found", "client not found")
	case errors.Is(err, projectdomain.ErrProjectNameExists):
		api.WriteError(w, http.StatusConflict, "project_name_already_exists", "project name already exists")
	case errors.Is(err, projectdomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	default:
		api.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func toResponse(run *domain.EstimateRun) domain.EstimateResponse {
	return domain.EstimateResponse{
		ID:                    run.ID,
		OrganizationID:        run.OrganizationID,
		CreatedByUserID:       run.CreatedByUserID,
		ClientID:              run.ClientID,
		Category:              run.Category,
		Mode:                  run.Mode,
		CatalogVersionID:      run.CatalogVersionID,
		Currency:              run.Currency,
		Input:                 run.Input,
		Result:                run.Result,
		EstimatedHours:        run.EstimatedHours,
		EstimatedTimelineDays: run.EstimatedTimelineDays,
		HoursPerMonth:         run.HoursPerMonth,
		MinimumPriceCents:     run.MinimumPriceCents,
		RecommendedPriceCents: run.RecommendedPriceCents,
		RiskLevel:             run.RiskLevel,
		CreatedAt:             run.CreatedAt,
	}
}
