package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	ActorID          uuid.UUID `json:"actor_id"`
	ActorEmail       string    `json:"acotr_email"`
	ActorDisplayName string    `json:"display_name"`
	Action           string    `json:"action"`
	EntityType       string    `json:"entity_type"`
	EntityID         uuid.UUID `json:"entity_id"`
	Summary          string    `json:"summary"`
	CreatedAt        time.Time `json:"created_at"`
}
