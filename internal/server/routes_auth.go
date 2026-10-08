package server

import (
	"net/http"

	"github.com/chuuch/gorest/internal/platform/middleware"
)

func (a *wiredApp) registerAuth(
	mux *http.ServeMux,
	loginLimiter *middleware.Limiter,
	refreshLimiter *middleware.Limiter,
) {
	mux.HandleFunc("POST /api/v1/auth/register", a.authHandler.Register)
	mux.Handle("POST /api/v1/auth/login", loginLimiter.Login(http.HandlerFunc(a.authHandler.Login)))
	mux.Handle("POST /api/v1/auth/refresh", refreshLimiter.Refresh(http.HandlerFunc(a.authHandler.Refresh)))
	mux.HandleFunc("POST /api/v1/auth/logout", a.authHandler.Logout)
	mux.HandleFunc("POST /api/v1/auth/accept-invite", a.authHandler.AcceptInvite)
	mux.Handle(
		"POST /api/v1/auth/forgot-password",
		loginLimiter.Forgot(http.HandlerFunc(a.authHandler.ForgotPassword)),
	)
	mux.HandleFunc("POST /api/v1/auth/reset-password", a.authHandler.ResetPassword)
	mux.Handle("POST /api/v1/auth-change-password", a.staff(a.authHandler.ChangePassword))
	mux.Handle("PATCH /api/v1/auth/display-name", a.staff(a.authHandler.UpdateDisplayName))
	mux.Handle("GET /api/v1/auth/me", a.staff(a.authHandler.Me))

	mux.Handle("GET /api/v1/users/{id}", a.staff(a.userHandler.GetByID))
	mux.Handle("PUT /api/v1/users/{id}", a.staff(a.userHandler.Update))
	mux.Handle("DELETE /api/v1/users/{id}", a.staff(a.userHandler.Delete))

	mux.Handle("GET /api/v1/members", a.staff(a.orgHandler.ListMembers))
	mux.Handle("POST /api/v1/members", a.staffIdempotent(a.orgHandler.CreateMember))
	mux.Handle("PATCH /api/v1/members/{id}", a.staff(a.orgHandler.UpdateMember))
	mux.Handle("DELETE /api/v1/members/{id}", a.staff(a.orgHandler.DeleteMember))
	mux.Handle("PATCH /api/v1/organization", a.staff(a.orgHandler.Update))
}
