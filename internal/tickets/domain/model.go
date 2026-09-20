package domain

import (
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindBug      Kind = "bug"
	KindFeature  Kind = "feature"
	KindQuestion Kind = "question"
	KindOther    Kind = "other"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
)

type Ticket struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ClientID       uuid.UUID
	UserID         uuid.UUID
	Kind           Kind
	Status         Status
	Title          string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
