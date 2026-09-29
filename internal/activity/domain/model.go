package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActionCreated   = "created"
	ActionUpdated   = "updated"
	ActionDeleted   = "deleted"
	ActionConverted = "converted"

	EntityTask          = "task"
	EntityTicket        = "ticket"
	EntityFile          = "file"
	EntityTicketFile    = "ticket_file"
	EntityComment       = "comment"
	EntityTicketComment = "ticket_comment"
)

type Event struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	ActorID          uuid.UUID
	ActorEmail       string
	ActorDisplayName string
	Action           string
	EntityType       string
	EntityID         uuid.UUID
	Summary          string
	CreatedAt        time.Time
}
