package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/auth/domain"
	"github.com/chuuch/gorest/internal/auth/usecase"
	"github.com/chuuch/gorest/internal/invites"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/chuuch/gorest/internal/validation"
)

const refreshTokenCookieName = "refresh_token"

type Handler struct {
	service         usecase.Service
	invites         invites.MailTokens
	refreshTokenTTL time.Duration
	cookieSecure    bool
}

func NewHandler(
	service usecase.Service,
	refreshTokenTTL time.Duration,
	cookieSecure bool,
	invites invites.MailTokens,
) *Handler {
	return &Handler{
		service:         service,
		invites:         invites,
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

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req domain.AcceptInviteRequest

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

	if err := h.invites.Accept(r.Context(), req.Token, req.Password); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ForgotPasswordRequest

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

	if err := h.invites.RequestReset(r.Context(), req.Email); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ResetPasswordRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid body request",
		)
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	if err := h.invites.Reset(r.Context(), req.Token, req.Password); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ChangePassword(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	var req domain.ChangePasswordRequest

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

	result, err := h.service.ChangePassword(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
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
			ID:        result.User.ID,
			Email:     result.User.Email,
			CreatedAt: result.User.CreatedAt,
			UpdatedAt: result.User.UpdatedAt,
		},
		Organization: orgdomain.OrganizationResponse{
			ID:        result.Organization.ID,
			Name:      result.Organization.Name,
			CreatedAt: result.Organization.CreatedAt,
			UpdatedAt: result.Organization.UpdatedAt,
		},
		Role: string(result.Role),
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

	case errors.Is(err, domain.ErrNoOrganization):
		api.WriteError(
			w,
			http.StatusForbidden,
			"no_organization",
			"user has no organization",
		)

	case errors.Is(err, invites.ErrInviteNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"invite_not_found",
			"invite not found",
		)

	case errors.Is(err, invites.ErrInviteExpired):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invite_expired",
			"invite expired",
		)

	case errors.Is(err, invites.ErrInviteUsed):
		api.WriteError(
			w,
			http.StatusConflict,
			"invite_used",
			"invite already used",
		)

	case errors.Is(err, invites.ErrResetNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"reset_not_found",
			"reset not found",
		)

	case errors.Is(err, invites.ErrResetExpired):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"reset_expired",
			"reset expired",
		)

	case errors.Is(err, invites.ErrResetUsed):
		api.WriteError(
			w,
			http.StatusConflict,
			"reset_used",
			"reset already used",
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
