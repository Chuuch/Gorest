package auth

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/user"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeAuthResponse(w, http.StatusCreated, response)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeAuthResponse(w, http.StatusOK, response)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeAuthResponse(w, http.StatusOK, response)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeAuthResponse(
	w http.ResponseWriter,
	status int,
	response *AuthResponse,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.MarshalWrite(w, response)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrEmailAlreadyExists):
		http.Error(w, "email already exists", http.StatusConflict)

	case errors.Is(err, ErrInvalidCredentials):
		http.Error(w, "invalid credentials", http.StatusUnauthorized)

	case errors.Is(err, ErrInvalidToken):
		http.Error(w, "invalid token", http.StatusUnauthorized)

	case errors.Is(err, ErrTokenExpired):
		http.Error(w, "token expired", http.StatusUnauthorized)

	case errors.Is(err, ErrTokenRevoked):
		http.Error(w, "token revoked", http.StatusUnauthorized)

	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
