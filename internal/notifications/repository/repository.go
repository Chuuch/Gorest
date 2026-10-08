package repository

import (
	"context"

	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	ListByRecipientID(
		ctx context.Context,
		organizationID, recipientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
	) ([]*domain.Notification, error)
	MarkRead(
		ctx context.Context,
		id, organizationID, recipientID uuid.UUID,
	) error
}
