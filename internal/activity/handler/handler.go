package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/activity/usecase"
	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/pagination"
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
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
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

	var cursor *pagination.Cursor
	if rawCursor := r.URL.Query().Get("cursor"); rawCursor != "" {
		decoded, err := pagination.Decode(rawCursor)
		if err != nil {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_cursor",
				"invalid cursor",
			)
			return
		}
		cursor = &decoded
	}

	events, nextCursor, err := h.service.List(r.Context(), organizationID, limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			api.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_cursor",
				"invalid_cursor",
			)
			return
		}
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

	api.WriteJSON(w, http.StatusOK, pagination.Page[domain.EventResponse]{
		Items:      responses,
		NextCursor: nextCursor,
	})
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
