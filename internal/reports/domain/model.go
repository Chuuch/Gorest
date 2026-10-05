package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClientRow struct {
	ClientID   uuid.UUID
	ClientName string
	Minutes    int
}

type ProjectRow struct {
	ProjectID   uuid.UUID
	ProjectName string
	ClientID    uuid.UUID
	ClientName  string
	Minutes     int
}

type MemberRow struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Minutes     int
}

type TimeReport struct {
	From         time.Time
	To           time.Time
	TotalMinutes int
	ByClient     []ClientRow
	ByProject    []ProjectRow
	ByMember     []MemberRow
}
