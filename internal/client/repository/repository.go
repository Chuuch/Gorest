package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/client/domain"
	"github.com/google/uuid"
)

type ClientRepository interface {
	Create(ctx context.Context, client *domain.Client) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Client, error)
	ListByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]*domain.Client, error)
}
