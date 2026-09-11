package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/auth/domain"
	"github.com/chuuch/gorest/internal/auth/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/chuuch/gorest/internal/validation"
)

const refreshTokenCookieName = "refresh_token"

type Handler struct {
	service         usecase.Service
	refreshTokenTTL time.Duration
	cookieSecure    bool
}

func NewHandler(
	service usecase.Service,
	refreshTokenTTL time.Duration,
	cookieSecure bool,
) *Handler {
	return &Handler{
		service:         service,
		refreshTokenTTL: refreshTokenTTL,
		cookieSecure:    cookieSecure,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest

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

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	h.writeAuthResponse(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest

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
	result *usecase.AuthResult,
) {
	api.WriteJSON(w, status, domain.AuthResponse{
		AccessToken: result.AccessToken,
		User: userdomain.UserResponse{
			ID: result.User.ID,
			Email: result.User.Email,
			CreatedAt: result.User.CreatedAt,
			UpdatedAt: result.User.UpdatedAt,
		},
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
		Secure:   h.cookieSecure,
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
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, userdomain.ErrEmailAlreadyExists):
		api.WriteError(
			w,
			http.StatusConflict,
			"email_already_exists",
			"email already exists",
		)

	case errors.Is(err, domain.ErrInvalidCredentials):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"invalid_credentials",
			"invalid credentials",
		)

	case errors.Is(err, domain.ErrInvalidToken):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"invalid_token",
			"invalid token",
		)

	case errors.Is(err, domain.ErrTokenExpired):
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"token_expired",
			"token expired",
		)

	case errors.Is(err, domain.ErrTokenRevoked):
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

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
		w,
		http.StatusUnauthorized,
		"unauthorized",
		"unauthorized",
		)
		return
	}
	result, err := h.service.Me(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeAuthResponse(w, http.StatusOK, result)
}
