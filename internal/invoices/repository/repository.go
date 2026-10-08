package repository

import (
	"context"
	"time"

	"github.com/chuuch/gorest/internal/invoices/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/google/uuid"
)

type InvoiceRepository interface {
	Create(ctx context.Context, invoice *domain.Invoice) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Invoice, error)
	ListByClientID(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
		statuses []string,
	) ([]*domain.Invoice, error)
	Update(ctx context.Context, invoice *domain.Invoice) error
	ReplaceLines(ctx context.Context, invoice *domain.Invoice) error
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
	NextNumber(ctx context.Context, organizationID uuid.UUID, year int) (string, error)
	SnapshotLines(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		from, to time.Time,
	) ([]domain.SnapshotLine, error)
}
