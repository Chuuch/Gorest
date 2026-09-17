package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProjectID      uuid.UUID
	Title          string
	Notes          string
	Status         Status
	CompletedAt		 *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
