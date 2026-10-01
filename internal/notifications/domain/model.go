package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	KindTicketOpened        = "ticket_opened"
	KindTicketClientComment = "ticket_client_comment"
	KindTaskAssigned        = "task_assigned"
	KindTaskCommented       = "task_commented"
	KindTicketStaffComment  = "ticket_staff_comment"
	KindTicketStatusChanged = "ticket_status_changed"
	KindTicketConverted     = "ticket_converted"

	EntityTicket        = "ticket"
	EntityTask          = "task"
	EntityComment       = "comment"
	EntityTicketComment = "ticket_comment"
)

type Notification struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RecipientID      uuid.UUID
	ActorID          uuid.UUID
	ActorEmail       string
	ActorDisplayName string
	Kind             string
	EntityType       string
	EntityID         uuid.UUID
	Summary          string
	ReadAt           *time.Time
	CreatedAt        time.Time
}
