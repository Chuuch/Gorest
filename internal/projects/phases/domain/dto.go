package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreatePhaseRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=100"`
	SortOrder *int    `json:"sort_order" validate:"omitempty,min=0"`
	StartsOn  *string `json:"starts_on" validate:"omitempty,datetime=2006-01-02"`
	DueOn     *string `json:"due_on" validate:"omitempty,datetime=2006-01-02"`
}

type UpdatePhaseRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=100"`
	SortOrder int     `json:"sort_order" validate:"min=0"`
	Status    string  `json:"status" validate:"required,oneof=not_started in_progress done"`
	StartsOn  *string `json:"starts_on" validate:"omitempty,datetime=2006-01-02"`
	DueOn     *string `json:"due_on" validate:"omitempty,datetime=2006-01-02"`
	Version   int     `json:"version" validate:"required,min=1"`
}

type PhaseResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ProjectID      uuid.UUID `json:"project_id"`
	Name           string    `json:"name"`
	SortOrder      int       `json:"sort_order"`
	Status         Status    `json:"status"`
	StartsOn       *string   `json:"starts_on,omitempty"`
	DueOn          *string   `json:"due_on,omitempty"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
