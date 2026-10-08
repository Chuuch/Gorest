package server

import (
	"net/http"

	"github.com/chuuch/gorest/internal/platform/middleware"
)

func (a *wiredApp) registerMeta(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.Handle("GET /metrics", middleware.MetricsHandler())

	mux.Handle("GET /api/v1/activity", a.staff(a.activityHandler.List))
	mux.Handle("GET /api/v1/reports/time", a.staff(a.reportHandler.Time))
	mux.Handle("GET /api/v1/notifications", a.staff(a.notificationHandler.List))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", a.staff(a.notificationHandler.MarkRead))
	mux.Handle("GET /api/v1/nav/counts", a.staff(a.navHandler.Counts))
	mux.Handle("GET /api/v1/events", a.staff(a.eventsHandler.Staff))
}
