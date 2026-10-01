package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/notifications"
	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/chuuch/gorest/internal/notifications/repository"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
)

type Service interface {
	notifications.Publisher
	List(
		ctx context.Context,
		organizationID, recipientID uuid.UUID,
		limit int,
	) ([]*domain.Notification, error)
	MarkRead(
		ctx context.Context,
		organizationID, recipientID, id uuid.UUID,
	) error
}

type service struct {
	notifications repository.NotificationRepository
	memberships   orgrepository.MembershipRepository
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
) ([]*domain.Notification, error) {
	items, err := s.notifications.ListByRecipientID(ctx, organizationID, recipientID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (s *service) MarkRead(
	ctx context.Context,
	organizationID, recipientID, id uuid.UUID,
) error {
	return s.notifications.MarkRead(ctx, id, organizationID, recipientID)
}
