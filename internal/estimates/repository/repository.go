package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/estimates/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/google/uuid"
)

type EstimateRepository interface {
	Create(ctx context.Context, run *domain.EstimateRun) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.EstimateRun, error)
	List(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		clientID *uuid.UUID,
	) ([]*domain.EstimateRun, error)
	CreateIntake(ctx context.Context, intake *domain.EstimateIntake) error
	GetIntakeByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.EstimateIntake, error)
	GetIntakeByTokenHash(ctx context.Context, tokenHash string) (*domain.EstimateIntake, error)
	ListIntakes(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		status string,
	) ([]*domain.EstimateIntake, error)
	UpdateIntake(ctx context.Context, intake *domain.EstimateIntake) error
}
