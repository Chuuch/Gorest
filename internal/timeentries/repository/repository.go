package repository

import (
	"context"
	"time"

	"github.com/chuuch/gorest/internal/timeentries/domain"
	"github.com/google/uuid"
)

type TimeEntryRepository interface {
	Create(ctx context.Context, entry *domain.TimeEntry) error
	GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.TimeEntry, error)
	ListByTaskID(ctx context.Context, organizationID, taskID uuid.UUID) ([]*domain.TimeEntry, error)
	ListByRange(ctx context.Context, organizationID uuid.UUID, userID *uuid.UUID, from, to time.Time) ([]*domain.TimeEntry, error)
	Update(ctx context.Context, entry *domain.TimeEntry) error
	Delete(ctx context.Context, id, organizationID uuid.UUID) error
}
