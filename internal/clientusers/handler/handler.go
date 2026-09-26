package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	"github.com/chuuch/gorest/internal/clientusers/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/chuuch/gorest/internal/validation"
	"github.com/google/uuid"
)

const refreshTokenCookieName = "refresh_token"

type Handler struct {
	service      usecase.Service
	refreshTTL   time.Duration
	cookieSecure bool
}

func NewHandler(
	service usecase.Service,
	refreshTTL time.Duration,
	cookieSecure bool,
) *Handler {
	return &Handler{
		service:      service,
		refreshTTL:   refreshTTL,
		cookieSecure: cookieSecure,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	members, err := h.service.List(r.Context(), organizationID, clientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]clientuserdomain.ClientUserResponse, 0, len(members))
	for _, member := range members {
		responses = append(responses, toMemberResponse(member))
	}
	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	var req clientuserdomain.CreateClientUserRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	member, err := h.service.Create(r.Context(), organizationID, clientID, actorRole, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toMemberResponse(*member))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	clientID, ok := h.clientID(w, r)
	if !ok {
		return
	}

	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		clientID,
		userID,
		actorRole,
	); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req clientuserdomain.LoginRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
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
			api.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid token")
			return
		}
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
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
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
		h.handleError(w, err)
		return
	}

	h.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	result, err := h.service.Me(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeAuthResponse(w, http.StatusOK, result)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req authdomain.ChangePasswordRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
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

func (h *Handler) clientID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	clientID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_client_id",
			"invalid client id",
		)
		return uuid.Nil, false
	}
	return clientID, true
}

func (h *Handler) userID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	userID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_user_id",
			"invlaid user id",
		)
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) staffSession(
	w http.ResponseWriter, r *http.Request,
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
			"unaurhotized",
		)
		return uuid.Nil, "", false
	}
	return organizationID, orgdomain.Role(role), true
}

func (h *Handler) writeAuthResponse(
	w http.ResponseWriter,
	status int,
	result *usecase.AuthResult,
) {
	api.WriteJSON(w, status, clientuserdomain.AuthResponse{
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
		Client: clientdomain.ClientResponse{
			ID:             result.Client.ID,
			OrganizationID: result.Client.OrganizationID,
			Name:           result.Client.Name,
			Notes:          result.Client.Notes,
			CreatedAt:      result.Client.CreatedAt,
			UpdatedAt:      result.Client.UpdatedAt,
		},
		Role: result.Role,
	})
}

func (h *Handler) setRefreshTokenCookie(
	w http.ResponseWriter,
	refreshToken string,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/client-auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(h.refreshTTL),
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/client-auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, clientuserdomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")

	case errors.Is(err, clientuserdomain.ErrUserIsStaff):
		api.WriteError(w, http.StatusConflict, "user_is_staff", "user is staff")

	case errors.Is(err, clientuserdomain.ErrClientUserAlreadyExists):
		api.WriteError(w, http.StatusConflict, "client_user_already_exists", "client user already exists")

	case errors.Is(err, clientuserdomain.ErrClientUserNotFound):
		api.WriteError(w, http.StatusNotFound, "client_user_not_found", "client user not found")

	case errors.Is(err, clientdomain.ErrClientNotFound):
		api.WriteError(w, http.StatusNotFound, "client_not_found", "client not found")

	case errors.Is(err, authdomain.ErrInvalidCredentials):
		api.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")

	case errors.Is(err, authdomain.ErrInvalidToken):
		api.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid token")

	case errors.Is(err, authdomain.ErrTokenExpired):
		api.WriteError(w, http.StatusUnauthorized, "token_expired", "token expired")

	case errors.Is(err, authdomain.ErrTokenRevoked):
		api.WriteError(w, http.StatusUnauthorized, "token_revoked", "token revoked")

	case errors.Is(err, userdomain.ErrEmailAlreadyExists):
		api.WriteError(w, http.StatusConflict, "email_already_exists", "email already exists")

	default:
		api.WriteError(w, http.StatusInternalServerError, "internal_server", "internal server error")
	}
}

func toMemberResponse(member usecase.Member) clientuserdomain.ClientUserResponse {
	return clientuserdomain.ClientUserResponse{
		UserID:         member.UserID,
		Email:          member.Email,
		OrganizationID: member.OrganizationID,
		ClientID:       member.ClientID,
		CreatedAt:      member.CreatedAt,
	}
}
