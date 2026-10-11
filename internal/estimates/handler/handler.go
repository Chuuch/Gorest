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

func (h *Handler) CreateIntake(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, role, ok := h.session(w, r)
	if !ok {
		return
	}

	var req domain.CreateIntakeRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}
	out, err := h.service.CreateIntake(r.Context(), organizationID, userID, role, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) GetIntake(w http.ResponseWriter, r *http.Request) {
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}
	id, ok := h.intakeID(w, r)
	if !ok {
		return
	}
	intake, err := h.service.GetIntake(r.Context(), organizationID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, toIntakeResponse(intake))
}

func (h *Handler) ListIntakes(w http.ResponseWriter, r *http.Request) {
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}
	limit, cursor, ok := pagination.LimitAndCursor(w, r, api.WriteError)
	if !ok {
		return
	}
	items, next, err := h.service.ListIntakes(r.Context(), organizationID, limit, cursor, r.URL.Query().Get("status"))
	if err != nil {
		h.handleError(w, err)
		return
	}

	out := make([]domain.IntakeResponse, 0, len(items))
	for _, intake := range items {
		out = append(out, toIntakeResponse(intake))
	}
	api.WriteJSON(w, http.StatusOK, pagination.Page[domain.IntakeResponse]{
		Items: out, NextCursor: next,
	})
}

func (h *Handler) GetPublicIntake(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetPublicIntake(r.Context(), r.PathValue("token"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) SubmitIntake(w http.ResponseWriter, r *http.Request) {
	var req domain.SubmitIntakeRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}
	if err := h.service.SubmitIntake(r.Context(), r.PathValue("token"), req); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) intakeID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_intake_id", "invalid intake id")
		return uuid.Nil, false
	}
	return id, true
}

func toIntakeResponse(intake *domain.EstimateIntake) domain.IntakeResponse {
	out := domain.IntakeResponse{
		ID:              intake.ID,
		OrganizationID:  intake.OrganizationID,
		CreatedByUserID: intake.CreatedByUserID,
		Category:        intake.Category,
		Mode:            intake.Mode,
		Status:          intake.Status,
		ExpiresAt:       intake.ExpiresAt,
		RespondentEmail: intake.RespondentEmail,
		RespondentName:  intake.RespondentName,
		CompanyName:     intake.CompanyName,
		ProjectName:     intake.ProjectName,
		ClientID:        intake.ClientID,
		Scope:           intake.Scope,
		SubmittedAt:     intake.SubmittedAt,
		CreatedAt:       intake.CreatedAt,
	}
	if intake.PreferredDeadline != nil {
		d := intake.PreferredDeadline.Format("2006-01-02")
		out.PreferredDeadline = &d
	}
	return out
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
	case errors.Is(err, domain.ErrIntakeNotFound):
		api.WriteError(w, http.StatusNotFound, "intake_not_found", "intake not found")
	case errors.Is(err, domain.ErrIntakeExpired):
		api.WriteError(w, http.StatusGone, "intake_expired", "intake expired")
	case errors.Is(err, domain.ErrIntakeSubmitted):
		api.WriteError(w, http.StatusConflict, "intake_already_submitted", "intake already submitted")
	case errors.Is(err, domain.ErrIntakeConsumed):
		api.WriteError(w, http.StatusConflict, "intake_already_consumed", "intake already consumed")
	case errors.Is(err, domain.ErrIntakeNotPending):
		api.WriteError(w, http.StatusConflict, "intake_not_pending", "intake not pending")
	case errors.Is(err, domain.ErrIntakeNotSubmitted):
		api.WriteError(w, http.StatusConflict, "intake_not_submitted", "intake is not submitted")
	case errors.Is(err, domain.ErrInvalidIntakeScope):
		api.WriteError(w, http.StatusBadRequest, "innvalid_intake_scope", err.Error())
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
