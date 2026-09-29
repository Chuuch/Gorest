package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/clientusers/domain"
	"github.com/google/uuid"
)

type ClientUserRepository interface {
	Create(ctx context.Context, clientuser *domain.ClientUser) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.ClientUser, error)
	ListByClientID(ctx context.Context, organizationID, clientID uuid.UUID) ([]*domain.ClientUser, error)
	Delete(ctx context.Context, organizationID, clientID, userID uuid.UUID) error
}
