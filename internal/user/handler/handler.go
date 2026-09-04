package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/chuuch/gorest/internal/user/domain"
	"github.com/chuuch/gorest/internal/user/usecase"
	"github.com/chuuch/gorest/internal/validation"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var dto domain.CreateUserRequest

	if err := json.UnmarshalRead(r.Body, &dto); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	if err := validation.Struct(dto); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	user, err := h.service.Create(r.Context(), dto)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := domain.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	api.WriteJSON(w, http.StatusCreated, response)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_user_id",
			"invalid user id",
		)
		return
	}

	authenticatedUserID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return
	}

	if authenticatedUserID != id {
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := domain.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	api.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_user_id",
			"invalid user id",
		)
		return
	}

	authenticatedUserID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return
	}

	if authenticatedUserID != id {
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)
		return
	}

	var dto domain.UpdateUserRequest

	if err := json.UnmarshalRead(r.Body, &dto); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	if err := validation.Struct(dto); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	user, err := h.service.Update(r.Context(), id, dto)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := domain.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	api.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_user_id",
			"invalid user id",
		)
		return
	}

	authenticatedUserID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return
	}

	if authenticatedUserID != id {
		api.WriteError(
			w,
			http.StatusForbidden,
			"forbidden",
			"forbidden",
		)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"user_not_found",
			"user not found",
		)

	case errors.Is(err, domain.ErrEmailAlreadyExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"email_already_exists",
			"email already exists",
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
