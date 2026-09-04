package auth

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/user"
	"github.com/chuuch/gorest/internal/validation"
)

const refreshTokenCookieName = "refresh_token"

type Handler struct {
	service         Service
	refreshTokenTTL time.Duration
}

func NewHandler(service Service, refreshTokenTTL time.Duration) *Handler {
	return &Handler{
		service:         service,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

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
		api.WriteError(
			w,
			http.StatusBadRequest,
			"validation_error",
			"request validation vailed",
		)
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

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
		api.WriteError(
			w,
			http.StatusBadRequest,
			"validation_error",
			"request validation failed",
		)
	}

	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusOK, result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			api.WriteError(
				w,
				http.StatusUnauthorized,
				"invalid_token",
				"invalid token",
			)
			return
		}

		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request",
		)
		return
	}

	result, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			h.clearRefreshTokenCookie(w)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request",
		)
		return
	}

	if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
		h.handleError(w, err)
		return
	}

	h.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeAuthResponse(
	w http.ResponseWriter,
	status int,
	result *AuthResult,
) {
	api.WriteJSON(w, status, AuthResponse{
		AccessToken: result.AccessToken,
	})
}

func (h *Handler) setRefreshTokenCookie(
	w http.ResponseWriter,
	refreshToken string,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(h.refreshTokenTTL),
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrEmailAlreadyExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"email_already_exists",
			"email already exists",
		)

	case errors.Is(err, ErrInvalidCredentials):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"invalid_credentials",
			"invalid credentials",
		)

	case errors.Is(err, ErrInvalidToken):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"invalid_token",
			"invalid token",
		)

	case errors.Is(err, ErrTokenExpired):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"token_expired",
			"token expired",
		)

	case errors.Is(err, ErrTokenRevoked):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"token_revoked",
			"token revoked",
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
