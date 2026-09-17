package domain

import (
	"time"

	"github.com/google/uuid"
)

type TimeEntry struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	TaskID         uuid.UUID
	UserID         uuid.UUID
	Minutes        int
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
