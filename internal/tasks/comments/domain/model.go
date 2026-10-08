package domain

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	TaskID         uuid.UUID
	UserID         uuid.UUID
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
