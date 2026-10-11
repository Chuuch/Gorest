package server

import (
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/platform/middleware"
)

func (a *wiredApp) registerEstimates(mux *http.ServeMux) {
	intakeLimiter := middleware.NewLimiter(30, 15*time.Minute)

	mux.Handle("GET /api/v1/estimate-catalog", a.staff(a.estimateHandler.Catalog))
	mux.Handle("POST /api/v1/estimates/preview", a.staff(a.estimateHandler.Preview))
	mux.Handle("POST /api/v1/estimates", a.staffIdempotent(a.estimateHandler.Create))
	mux.Handle("GET /api/v1/estimates", a.staff(a.estimateHandler.List))
	mux.Handle("GET /api/v1/estimates/{id}", a.staff(a.estimateHandler.Get))
	mux.Handle("POST /api/v1/estimates/{id}/create-project", a.staffIdempotent(a.estimateHandler.CreateProject))

	mux.Handle("POST /api/v1/estimate-intakes", a.staffIdempotent(a.estimateHandler.CreateIntake))
	mux.Handle("GET /api/v1/estimate-intakes", a.staff(a.estimateHandler.ListIntakes))
	mux.Handle("GET /api/v1/estimate-intakes/{id}", a.staff(a.estimateHandler.GetIntake))

	mux.Handle(
		"GET /api/v1/public/estimate-intakes/{token}",
		intakeLimiter.IntakeGet(http.HandlerFunc(a.estimateHandler.GetPublicIntake)),
	)
	mux.Handle(
		"POST /api/v1/public/estimate-intakes/{token}/submit",
		intakeLimiter.IntakeSubmit(http.HandlerFunc(a.estimateHandler.SubmitIntake)),
	)
}
