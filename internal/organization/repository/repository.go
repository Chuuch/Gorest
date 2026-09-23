package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/organization/domain"
	"github.com/google/uuid"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
}

type MembershipRepository interface {
	Create(ctx context.Context, membership *domain.Membership) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Membership, error)
	ListByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]*domain.Membership, error)
	CountOwners(ctx context.Context, organizationID uuid.UUID) (int, error)
	UpdateRole(ctx context.Context, organizationID, userID uuid.UUID, role domain.Role) error
	Delete(ctx context.Context, organizationID, userID uuid.UUID) error
}
