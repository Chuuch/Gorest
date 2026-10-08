package handler

import (
	"net/http"

	"github.com/chuuch/gorest/internal/nav/usecase"
	"github.com/chuuch/gorest/internal/platform/api"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, portal, ok := h.session(w, r)
	if !ok {
		return
	}

	counts, err := h.service.Counts(r.Context(), organizationID, userID, portal)
	if err != nil {
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
		)
		return
	}

	api.WriteJSON(w, http.StatusOK, counts)
}

func (h *Handler) session(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, bool, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, false, false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return uuid.Nil, uuid.Nil, false, false
	}

	role, _ := requestcontext.Role(r.Context())
	portal := role == "client"
	return organizationID, userID, portal, true
}
