package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/activity/repository"
	"github.com/chuuch/gorest/internal/events"
	"github.com/chuuch/gorest/internal/pagination"
	"github.com/google/uuid"
)

type Service interface {
	Record(ctx context.Context, event domain.Event) error
	List(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
	) ([]*domain.Event, *string, error)
}

type service struct {
	events repository.EventRepository
	hub    *events.Hub
}

func NewService(events repository.EventRepository) Service {
	return &service{events: events}
}

func EnableRealtime(s Service, hub *events.Hub) Service {
	inner, ok := s.(*service)
	if !ok || hub == nil {
		return s
	}
	inner.hub = hub
	return inner
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

	if s.hub != nil {
		s.hub.PublishActivity(event.OrganizationID)
	}

	return nil
}

func (s *service) List(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
) ([]*domain.Event, *string, error) {
	events, err := s.events.ListByOrganizationID(ctx, organizationID, limit+1, cursor)
	if err != nil {
		return nil, nil, fmt.Errorf("list activity events: %w", err)
	}

	page, next := pagination.NextCursor(events, limit, func(event *domain.Event) pagination.Cursor {
		return pagination.Cursor{
			CreatedAt: event.CreatedAt,
			ID:        event.ID,
		}
	})
	return page, next, nil
}
