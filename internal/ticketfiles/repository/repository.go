package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/ticketfiles/domain"
	"github.com/google/uuid"
)

type TicketFileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.File, error)
	ListByTicketID(ctx context.Context, organizationID, ticketID uuid.UUID) ([]*domain.File, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
}
