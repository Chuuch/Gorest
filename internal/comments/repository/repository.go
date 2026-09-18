package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/comments/domain"
	"github.com/google/uuid"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Comment, error)
	Update(ctx context.Context, comment *domain.Comment) error
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
	ListByTaskID(ctx context.Context, organizationID, taskID uuid.UUID) ([]*domain.Comment, error)
}
