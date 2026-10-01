package handler

import (
	"net/http"
	"time"

	"github.com/chuuch/gorest/internal/api"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	reportsdomain "github.com/chuuch/gorest/internal/reports/domain"
	usecase "github.com/chuuch/gorest/internal/reports/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Time(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, actorRole, ok := h.actor(w, r)
	if !ok {
		return
	}

	from, to, ok := h.rangeBounds(w, r)
	if !ok {
		return
	}

	report, err := h.service.Time(
		r.Context(),
		organizationID,
		userID,
		actorRole,
		from,
		to,
	)
	if err != nil {
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_server",
			"internal server error",
		)
		return
	}

	api.WriteJSON(w, http.StatusOK, toResponse(report))
}

func (h *Handler) rangeBounds(
	w http.ResponseWriter,
	r *http.Request,
) (time.Time, time.Time, bool) {
	from, fromErr := time.Parse(time.RFC3339Nano, r.URL.Query().Get("from"))
	to, toErr := time.Parse(time.RFC3339Nano, r.URL.Query().Get("to"))
	if fromErr != nil || toErr != nil || !to.After(from) {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_time_range",
			"from and to must be RFC3339 and to must be after from",
		)
		return time.Time{}, time.Time{}, false
	}

	return from, to, true
}

func (h *Handler) actor(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	return organizationID, userID, orgdomain.Role(role), true
}

func toResponse(report *reportsdomain.TimeReport) reportsdomain.TimeReportResponse {
	clients := make([]reportsdomain.ClientRowResponse, 0, len(report.ByClient))
	for _, row := range report.ByClient {
		clients = append(clients, reportsdomain.ClientRowResponse{
			ClientID:   row.ClientID,
			ClientName: row.ClientName,
			Minutes:    row.Minutes,
		})
	}

	projects := make([]reportsdomain.ProjectRowResponse, 0, len(report.ByProject))
	for _, row := range report.ByProject {
		projects = append(projects, reportsdomain.ProjectRowResponse{
			ProjectID:   row.ProjectID,
			ProjectName: row.ProjectName,
			ClientID:    row.ClientID,
			ClientName:  row.ClientName,
			Minutes:     row.Minutes,
		})
	}

	members := make([]reportsdomain.MemberRowResponse, 0, len(report.ByMember))
	for _, row := range report.ByMember {
		members = append(members, reportsdomain.MemberRowResponse{
			UserID:      row.UserID,
			Email:       row.Email,
			DisplayName: row.DisplayName,
			Minutes:     row.Minutes,
		})
	}

	return reportsdomain.TimeReportResponse{
		From:         report.From,
		To:           report.To,
		TotalMinutes: report.TotalMinutes,
		ByClient:     clients,
		ByProject:    projects,
		ByMember:     members,
	}
}
