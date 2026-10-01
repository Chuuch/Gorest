package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	ListByRecipientID(
		ctx context.Context,
		organizationID, recipientID uuid.UUID,
		limit int,
	) ([]*domain.Notification, error)
	MarkRead(
		ctx context.Context,
		id, organizationID, recipientID uuid.UUID,
	) error
}
