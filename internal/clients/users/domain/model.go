package domain

import (
	"time"

	"github.com/google/uuid"
)

const RoleClient = "client"

type ClientUser struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ClientID       uuid.UUID
	UserID         uuid.UUID
	CreatedAt      time.Time
}
