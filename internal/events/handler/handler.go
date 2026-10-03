package handler

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/events"
	"github.com/chuuch/gorest/internal/requestcontext"
)

type Handler struct {
	hub *events.Hub
}

func NewHandler(hub *events.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) Staff(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	ch, cancel := h.hub.SubscribeStaff(organizationID, userID)
	h.stream(w, r, ch, cancel)
}

func (h *Handler) Portal(w http.ResponseWriter, r *http.Request) {
	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	ch, cancel := h.hub.SubscribePortal(userID)
	h.stream(w, r, ch, cancel)
}

func (h *Handler) stream(
	w http.ResponseWriter,
	r *http.Request,
	ch <-chan events.Event,
	cancel func(),
) {
	defer cancel()

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, open := <-ch:
			if !open {
				return
			}
			payload, err := json.Marshal(event)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
