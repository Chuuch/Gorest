package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/projects/phases/domain"
	"github.com/google/uuid"
)

type PhaseRepository interface {
	Create(ctx context.Context, phase *domain.Phase) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Phase, error)
	Update(ctx context.Context, phase *domain.Phase) error
	ListByProjectID(ctx context.Context, organizationID, projectID uuid.UUID) ([]*domain.Phase, error)
	NextSortOrder(ctx context.Context, organizationID, projectID uuid.UUID) (int, error)
}
