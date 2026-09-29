package handler

import (
	"net/http"
	"strconv"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/activity/usecase"
	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return
	}

	limit := 50
	raw := r.URL.Query().Get("limit")
	if raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_limit",
				"invalid limit",
			)
			return
		}
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	events, err := h.service.List(r.Context(), organizationID, limit)
	if err != nil {
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
		)
		return
	}

	responses := make([]domain.EventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, toResponse(event))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func toResponse(event *domain.Event) domain.EventResponse {
	return domain.EventResponse{
		ID:               event.ID,
		OrganizationID:   event.OrganizationID,
		ActorID:          event.ActorID,
		ActorEmail:       event.ActorEmail,
		ActorDisplayName: event.ActorDisplayName,
		Action:           event.Action,
		EntityType:       event.EntityType,
		EntityID:         event.EntityID,
		Summary:          event.Summary,
		CreatedAt:        event.CreatedAt,
	}
}
