package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClientRowResponse struct {
	ClientID   uuid.UUID `json:"client_id"`
	ClientName string    `json:"client_name"`
	Minutes    int       `json:"minutes"`
}

type ProjectRowResponse struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ClientID    uuid.UUID `json:"client_id"`
	ClientName  string    `json:"client_name"`
	Minutes     int       `json:"minutes"`
}

type MemberRowResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Minutes     int       `json:"minutes"`
}

type TimeReportResponse struct {
	From         time.Time            `json:"from"`
	To           time.Time            `json:"to"`
	TotalMinutes int                  `json:"total_minutes"`
	ByClient     []ClientRowResponse  `json:"by_client"`
	ByProject    []ProjectRowResponse `json:"by_project"`
	ByMember     []MemberRowResponse  `json:"by_member"`
}
