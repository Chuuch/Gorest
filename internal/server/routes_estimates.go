package server

import "net/http"

func (a *wiredApp) registerEstimates(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/estimate-catalog", a.staff(a.estimateHandler.Catalog))
	mux.Handle("POST /api/v1/estimates/preview", a.staff(a.estimateHandler.Preview))
	mux.Handle("POST /api/v1/estimates", a.staffIdempotent(a.estimateHandler.Create))
	mux.Handle("GET /api/v1/estimates", a.staff(a.estimateHandler.List))
	mux.Handle("GET /api/v1/estimates/{id}", a.staff(a.estimateHandler.Get))
	mux.Handle("POST /api/v1/estimates/{id}/create-project", a.staffIdempotent(a.estimateHandler.CreateProject))
}
