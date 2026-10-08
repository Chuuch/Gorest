package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/google/uuid"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	ListByOrganizationID(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
	) ([]*domain.Event, error)
}
