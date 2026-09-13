package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/organization/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/chuuch/gorest/internal/validation"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.session(w, r)
	if !ok {
		return
	}

	members, err := h.service.ListMembers(r.Context(), organizationID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]orgdomain.MemberResponse, 0, len(members))
	for _, member := range members {
		responses = append(responses, orgdomain.MemberResponse{
			UserID:    member.UserID,
			Email:     member.Email,
			Role:      member.Role,
			CreatedAt: member.CreatedAt,
		})
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) CreateMember(w http.ResponseWriter, r *http.Request) {
	organizationID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	var req orgdomain.CreateMemberRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	member, err := h.service.CreateMember(r.Context(), organizationID, actorRole, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, orgdomain.MemberResponse{
		UserID:    member.UserID,
		Email:     member.Email,
		Role:      member.Role,
		CreatedAt: member.CreatedAt,
	})
}

func (h *Handler) session(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, "", false
	}

	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, "", false
	}

	return organizationID, orgdomain.Role(role), true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orgdomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")

	case errors.Is(err, orgdomain.ErrMemberAlreadyExists):
		api.WriteError(w, http.StatusConflict, "member_already_exists", "member already exists")

	case errors.Is(err, orgdomain.ErrCannotCreateOwner):
		api.WriteError(w, http.StatusBadRequest, "invalid_role", "cannot create owner")

	default:
		api.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
