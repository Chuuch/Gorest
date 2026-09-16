package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateTimeEntryRequest struct {
	Minutes int    `json:"minutes" validate:"required,min=1,max=1440"`
	Notes   string `json:"notes" validate:"max=2000"`
}

type TimeEntryResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	TaskID         uuid.UUID `json:"task_id"`
	UserID         uuid.UUID `json:"user_id"`
	Minutes        int       `json:"minutes"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
