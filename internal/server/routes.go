package server

import (
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/platform/middleware"
)

func newRouter(a *wiredApp) *http.ServeMux {
	mux := http.NewServeMux()
	loginLimiter := middleware.NewLimiter(10, 15*time.Minute)
	refreshLimiter := middleware.NewLimiter(30, 15*time.Minute)

	a.registerMeta(mux)
	a.registerAuth(mux, loginLimiter, refreshLimiter)
	a.registerClients(mux)
	a.registerTasks(mux)
	a.registerTickets(mux)
	a.registerPortal(mux, loginLimiter, refreshLimiter)
	a.registerEstimates(mux)

	return mux
}

func (a *wiredApp) staff(h http.HandlerFunc) http.Handler {
	return middleware.Auth(a.tokenManager)(middleware.Staff(h))
}

func (a *wiredApp) portal(h http.HandlerFunc) http.Handler {
	return middleware.Auth(a.tokenManager)(middleware.ClientPortal(h))
}

func (a *wiredApp) staffIdempotent(h http.HandlerFunc) http.Handler {
	return middleware.Auth(a.tokenManager)(middleware.Staff(a.idem.Handler(h)))
}

func (a *wiredApp) portalIdempotent(h http.HandlerFunc) http.Handler {
	return middleware.Auth(a.tokenManager)(middleware.ClientPortal(a.idem.Handler(h)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
