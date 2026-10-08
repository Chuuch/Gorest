package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/notifications"
	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/chuuch/gorest/internal/notifications/repository"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	"github.com/chuuch/gorest/internal/platform/events"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/google/uuid"
)

type Service interface {
	notifications.Publisher
	List(
		ctx context.Context,
		organizationID, recipientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
	) ([]*domain.Notification, *string, error)
	MarkRead(
		ctx context.Context,
		organizationID, recipientID, id uuid.UUID,
	) error
}

type service struct {
	notifications repository.NotificationRepository
	memberships   orgrepository.MembershipRepository
	hub           *events.Hub
}

func NewService(
	notes repository.NotificationRepository,
	memberships orgrepository.MembershipRepository,
) Service {
	return &service{
		notifications: notes,
		memberships:   memberships,
	}
}

func EnableRealtime(s Service, hub *events.Hub) Service {
	inner, ok := s.(*service)
	if !ok || hub == nil {
		return s
	}
	inner.hub = hub
	return inner
}

func (s *service) Publish(ctx context.Context, msg notifications.Message) error {
	actorID, ok := requestcontext.UserID(ctx)
	if !ok {
		return nil
	}

	recipientIDs, err := s.recipients(ctx, msg, actorID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, recipientID := range recipientIDs {
		if recipientID == actorID {
			continue
		}

		item := &domain.Notification{
			ID:             uuid.New(),
			OrganizationID: msg.OrganizationID,
			RecipientID:    recipientID,
			ActorID:        actorID,
			Kind:           msg.Kind,
			EntityType:     msg.EntityType,
			EntityID:       msg.EntityID,
			Summary:        msg.Summary,
			CreatedAt:      now,
		}
		if err := s.notifications.Create(ctx, item); err != nil {
			return fmt.Errorf("create notification: %w", err)
		}
		if s.hub != nil {
			s.hub.PublishNotification(recipientID)
		}
	}

	return nil
}

func (s *service) recipients(
	ctx context.Context,
	msg notifications.Message,
	actorID uuid.UUID,
) ([]uuid.UUID, error) {
	if !msg.Staff {
		if msg.RecipientUserID == uuid.Nil || msg.RecipientUserID == actorID {
			return nil, nil
		}
		return []uuid.UUID{msg.RecipientUserID}, nil
	}

	memberships, err := s.memberships.ListByOrganizationID(ctx, msg.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("list notification staff: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(memberships))
	for _, membership := range memberships {
		if membership.UserID == actorID {
			continue
		}
		ids = append(ids, membership.UserID)
	}

	return ids, nil
}

func (s *service) List(
	ctx context.Context,
	organizationID, recipientID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
) ([]*domain.Notification, *string, error) {
	items, err := s.notifications.ListByRecipientID(ctx, organizationID, recipientID, limit+1, cursor)
	if err != nil {
		return nil, nil, fmt.Errorf("list notifications: %w", err)
	}

	page, next := pagination.NextCursor(items, limit, func(item *domain.Notification) pagination.Cursor {
		return pagination.Cursor{
			CreatedAt: item.CreatedAt,
			ID:        item.ID,
		}
	})

	return page, next, nil
}

func (s *service) MarkRead(
	ctx context.Context,
	organizationID, recipientID, id uuid.UUID,
) error {
	return s.notifications.MarkRead(ctx, id, organizationID, recipientID)
}
