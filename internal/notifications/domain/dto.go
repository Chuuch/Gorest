package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationResponse struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	RecipientID      uuid.UUID  `json:"recipient_id"`
	ActorID          uuid.UUID  `json:"actor_id"`
	ActorEmail       string     `json:"actor_email"`
	ActorDisplayName string     `json:"actor_display_name"`
	Kind             string     `json:"kind"`
	EntityType       string     `json:"entity_type"`
	EntityID         uuid.UUID  `json:"entity_id"`
	Summary          string     `json:"summary"`
	ReadAt           *time.Time `json:"read_at"`
	CreatedAt        time.Time  `json:"created_at"`
}
