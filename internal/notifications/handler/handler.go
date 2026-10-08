package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/chuuch/gorest/internal/notifications/usecase"
	"github.com/chuuch/gorest/internal/platform/api"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, ok := h.session(w, r)
	if !ok {
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

	items, nextCursor, err := h.service.List(r.Context(), organizationID, userID, limit, cursor)
	if err != nil {
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_server",
			"internal server error",
		)
		return
	}

	responses := make([]domain.NotificationResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toResponse(item))
	}

	api.WriteJSON(w, http.StatusOK, pagination.Page[domain.NotificationResponse]{
		Items:      responses,
		NextCursor: nextCursor,
	})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, ok := h.session(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_notification_id",
			"invalid notification id",
		)
		return
	}

	if err := h.service.MarkRead(r.Context(), organizationID, userID, id); err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			api.WriteError(
				w,
				http.StatusNotFound,
				"notification_not_found",
				"notification not found",
			)
			return
		}

		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_server",
			"internal server error",
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) session(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}

	return organizationID, userID, true
}

func toResponse(item *domain.Notification) domain.NotificationResponse {
	return domain.NotificationResponse{
		ID:               item.ID,
		OrganizationID:   item.OrganizationID,
		RecipientID:      item.RecipientID,
		ActorID:          item.ActorID,
		ActorEmail:       item.ActorEmail,
		ActorDisplayName: item.ActorDisplayName,
		Kind:             item.Kind,
		EntityType:       item.EntityType,
		EntityID:         item.EntityID,
		Summary:          item.Summary,
		ReadAt:           item.ReadAt,
		CreatedAt:        item.CreatedAt,
	}
}
