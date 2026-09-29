package domain

import (
	"bytes"
	"encoding/json/v2"
	"time"

	"github.com/google/uuid"
)

type CreateTaskRequest struct {
	Title      string     `json:"title" validate:"required,min=4,max=100"`
	Notes      string     `json:"notes" validate:"max=2000"`
	Status     string     `json:"status" validate:"required,oneof=todo in_progress done"`
	AssigneeID *uuid.UUID `json:"assignee_id"`
}

type OptionalAssignee struct {
	Set   bool
	Value *uuid.UUID
}

func (o *OptionalAssignee) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}

	var id uuid.UUID
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}

	o.Value = &id
	return nil
}

type UpdateTaskRequest struct {
	Title      string           `json:"title" validate:"omitempty,min=4,max=100"`
	Notes      *string          `json:"notes" validate:"omitempty,max=2000"`
	Status     string           `json:"status" validate:"required,oneof=todo in_progress done"`
	AssigneeID OptionalAssignee `json:"assignee_id"`
	Version    int              `json:"version" validate:"required,min=1"`
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
	CreatedBy      *uuid.UUID `json:"created_by"`
	AssigneeID     *uuid.UUID `json:"assignee_id"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
