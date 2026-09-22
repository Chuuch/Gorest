package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateTaskRequest struct {
	Title  string `json:"title" validate:"required,min=4,max=100"`
	Notes  string `json:"notes" validate:"max=2000"`
	Status string `json:"status" validate:"required,oneof=todo in_progress done"`
}

type UpdateTaskRequest struct {
	Status  string `json:"status" validate:"required,oneof=todo in_progress done"`
	Version int    `json:"version" validate:"required,min=1"`
}

type ConvertTicketRequest struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
}

type TaskResponse struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	TicketID       *uuid.UUID `json:"ticket_id"`
	Title          string     `json:"title"`
	Notes          string     `json:"notes"`
	Status         Status     `json:"status"`
	CompletedAt    *time.Time `json:"completed_at"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
