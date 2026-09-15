package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/projects/domain"
	"github.com/google/uuid"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Project, error)
	ListByClientID(ctx context.Context, organizationID, clientID uuid.UUID) ([]*domain.Project, error)
}
