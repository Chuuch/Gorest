package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	"github.com/chuuch/gorest/internal/tasks/usecase"
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
	organizationID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	tasks, err := h.service.List(r.Context(), organizationID, projectID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]taskdomain.TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		responses = append(responses, toResponse(task))
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

	var req taskdomain.CreateTaskRequest

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

	task, err := h.service.Create(
		r.Context(),
		organizationID,
		projectID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(task))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	var req taskdomain.UpdateTaskRequest

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

	task, err := h.service.Update(r.Context(), organizationID, taskID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(task))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), organizationID, taskID, actorRole); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	var req taskdomain.ConvertTicketRequest

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

	task, err := h.service.Convert(
		r.Context(),
		organizationID,
		ticketID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(task))
}

func (h *Handler) projectID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_project_id",
			"invalid project id",
		)
		return uuid.Nil, false
	}
	return projectID, true
}

func (h *Handler) taskID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	taskID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_task_id",
			"invalid task id",
		)
		return uuid.Nil, false
	}
	return taskID, true
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
	case errors.Is(err, taskdomain.ErrForbidden):
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbiden",
			"forbidden",
		)

	case errors.Is(err, taskdomain.ErrTaskTitleExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"task_title_already_exists",
			"task title already exists",
		)

	case errors.Is(err, taskdomain.ErrTaskVersionMismatch):
		api.WriteError(
			w,
			http.StatusConflict,
			"task_version_mismatch",
			"task was updated by someone else",
		)

	case errors.Is(err, taskdomain.ErrTicketAlreadyConverted):
		api.WriteError(
			w,
			http.StatusConflict,
			"ticket_already_converted",
			"ticket already converted",
		)

	case errors.Is(err, projectdomain.ErrProjectNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"project_not_found",
			"project not found",
		)

	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"ticket_not_found",
			"ticket not found",
		)

	case errors.Is(err, taskdomain.ErrTaskNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"task_not_found",
			"task not found",
		)

	default:
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_server",
			"internal server error",
		)
	}
}

func toResponse(task *taskdomain.Task) taskdomain.TaskResponse {
	return taskdomain.TaskResponse{
		ID:             task.ID,
		OrganizationID: task.OrganizationID,
		ProjectID:      task.ProjectID,
		TicketID:       task.TicketID,
		Title:          task.Title,
		Notes:          task.Notes,
		Status:         task.Status,
		CompletedAt:    task.CompletedAt,
		Version:        task.Version,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}
}
