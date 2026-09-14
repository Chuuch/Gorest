package domain

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ClientID       uuid.UUID
	Name           string
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
