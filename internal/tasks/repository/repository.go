package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/tasks/domain"
	"github.com/google/uuid"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	ListByProjectID(ctx context.Context, organizationID, projectID uuid.UUID) ([]*domain.Task, error)
}
