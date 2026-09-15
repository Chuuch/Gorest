package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	timeentrydomain "github.com/chuuch/gorest/internal/timeentries/domain"
	usecase "github.com/chuuch/gorest/internal/timeentries/usecase"
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

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	entries, err := h.service.List(r.Context(), organizationID, taskID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]timeentrydomain.TimeEntryResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, toResponse(entry))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, ok := h.session(w, r)
	if !ok {
		return
	}

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	var req timeentrydomain.CreateTimeEntryRequest

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

	entry, err := h.service.Create(
		r.Context(),
		organizationID,
		taskID,
		userID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(entry))
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

func (h *Handler) session(
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrTaskNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"task_not_found",
			"task not founD",
		)

	case errors.Is(err, timeentrydomain.ErrTimeEntryNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"time_entry_not_found",
			"time entry not found",
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

func toResponse(entry *timeentrydomain.TimeEntry) timeentrydomain.TimeEntryResponse {
	return timeentrydomain.TimeEntryResponse{
		ID: entry.ID,
		OrganizationID: entry.OrganizationID,
		TaskID: entry.TaskID,
		UserID: entry.UserID,
		Minutes: entry.Minutes,
		Notes: entry.Notes,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}
}
