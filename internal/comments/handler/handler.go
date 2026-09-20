package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	commentdomain "github.com/chuuch/gorest/internal/comments/domain"
	"github.com/chuuch/gorest/internal/comments/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
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
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	comments, err := h.service.List(r.Context(), organizationID, taskID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]commentdomain.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, toResponse(comment))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	taskID, ok := h.taskID(w, r)
	if !ok {
		return
	}

	var req commentdomain.CreateCommentRequest

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
		taskID,
		userID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(comment))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}

	var req commentdomain.UpdateCommentRequest

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
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(comment))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	commentID, ok := h.commentID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		commentID,
		userID,
		actorRole,
	); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) session(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	return organizationID, userID, orgdomain.Role(role), true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commentdomain.ErrForbidden):
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)

	case errors.Is(err, taskdomain.ErrTaskNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"task_not_found",
			"task not found",
		)

	case errors.Is(err, commentdomain.ErrCommentNotFound):
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
			"internal_server",
			"internal server error",
		)
	}
}

func toResponse(comment *commentdomain.Comment) commentdomain.CommentResponse {
	return commentdomain.CommentResponse{
		ID:             comment.ID,
		OrganizationID: comment.OrganizationID,
		TaskID:         comment.TaskID,
		UserID:         comment.UserID,
		Body:           comment.Body,
		CreatedAt:      comment.CreatedAt,
		UpdatedAt:      comment.UpdatedAt,
	}
}
