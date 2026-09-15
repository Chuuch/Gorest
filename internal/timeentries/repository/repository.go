package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/timeentries/domain"
	"github.com/google/uuid"
)

type TimeEntryRepository interface {
	Create(ctx context.Context, entry *domain.TimeEntry) error
	ListByTaskID(ctx context.Context, organizationID, taskID uuid.UUID) ([]*domain.TimeEntry, error)
}
