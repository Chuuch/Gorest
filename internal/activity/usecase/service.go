package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/activity/repository"
	"github.com/google/uuid"
)

type Service interface {
	Record(ctx context.Context, event domain.Event) error
	List(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
	) ([]*domain.Event, error)
}

type service struct {
	events repository.EventRepository
}

func NewService(events repository.EventRepository) Service {
	return &service{events: events}
}

func (s *service) Record(
	ctx context.Context,
	event domain.Event,
) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	if err := s.events.Create(ctx, &event); err != nil {
		return fmt.Errorf("create activity event: %w", err)
	}

	return nil
}

func (s *service) List(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
) ([]*domain.Event, error) {
	events, err := s.events.ListByOrganizationID(ctx, organizationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}

	return events, nil
}
