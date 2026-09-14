package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
