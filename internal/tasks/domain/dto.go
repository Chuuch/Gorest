package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateTaskRequest struct {
	Title string `json:"title" validate:"required,min=2,max=100"`
	Notes string `json:"notes" validate:"max=2000"`
	Status string `json:"status" validate:"required,oneof=todo in_progress done"`
}

type TaskResponse struct {
	ID uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ProjectID uuid.UUID `json:"project_id"`
	Title string `json:"title"`
	Notes string `json:"notes"`
	Status Status `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
