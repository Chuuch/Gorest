package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/files/domain"
	"github.com/google/uuid"
)

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.File, error)
	ListByProjectID(ctx context.Context, organizationID, projectID uuid.UUID) ([]*domain.File, error)
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
}
