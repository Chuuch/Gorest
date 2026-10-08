package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/tickets/domain"
	"github.com/google/uuid"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket *domain.Ticket) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Ticket, error)
	Update(ctx context.Context, ticket *domain.Ticket) error
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
	ListByClientID(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
	) ([]*domain.Ticket, error)
	ListByOrganizationID(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
	) ([]*domain.Ticket, error)
}
