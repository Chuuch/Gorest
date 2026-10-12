package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusNotStarted Status = "not_started"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Phase struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProjectID      uuid.UUID
	Name           string
	SortOrder      int
	Status         Status
	StartsOn       *time.Time
	DueOn          *time.Time
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
