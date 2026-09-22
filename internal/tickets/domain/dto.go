package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateTicketRequest struct {
	Kind  string `json:"kind" validte:"required,oneof=bug feature question other"`
	Title string `json:"title" validate:"required,min=4,max=2000"`
	Body  string `json:"body" validate:"required,min=1,max=2000"`
}

type UpdateTicketRequest struct {
	Status  string `json:"status" validate:"required,oneof=open in_progress resolved closed"`
	Version int    `json:"version" validate:"required,min=1"`
}

type TicketResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ClientID       uuid.UUID `json:"client_id"`
	UserID         uuid.UUID `json:"user_id"`
	Kind           Kind      `json:"kind"`
	Status         Status    `json:"status"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
