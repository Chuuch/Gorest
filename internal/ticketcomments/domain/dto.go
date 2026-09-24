package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateCommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

type UpdateCommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	TicketID       uuid.UUID `json:"ticket_id"`
	UserID         uuid.UUID `json:"user_id"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
