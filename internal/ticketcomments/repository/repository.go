package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/ticketcomments/domain"
	"github.com/google/uuid"
)

type TicketCommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	ListByTicketID(ctx context.Context, organizationID, ticketID uuid.UUID) ([]*domain.Comment, error)
}
