package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Name  string `json:"name" validate:"required,min=4,max=100"`
	Notes string `json:"notes" validate:"max=2000"`
}

type ProjectResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ClientID       uuid.UUID `json:"client_id"`
	Name           string    `json:"name"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
