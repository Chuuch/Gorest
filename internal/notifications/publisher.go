package notifications

import (
	"context"
	"log/slog"

	"github.com/chuuch/gorest/internal/activity"
	"github.com/google/uuid"
)

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

type Message struct {
	OrganizationID  uuid.UUID
	Kind            string
	EntityType      string
	EntityID        uuid.UUID
	Summary         string
	RecipientUserID uuid.UUID
	Staff           bool
}

type Nop struct{}

func (Nop) Publish(context.Context, Message) error {
	return nil
}

func NotifyStaff(
	ctx context.Context,
	pub Publisher,
	organizationID uuid.UUID,
	kind, entityType, summary string,
	entityID uuid.UUID,
) {
	if pub == nil {
		return
	}

	err := pub.Publish(ctx, Message{
		OrganizationID: organizationID,
		Kind:           kind,
		EntityType:     entityType,
		EntityID:       entityID,
		Summary:        activity.Summary(summary),
		Staff:          true,
	})
	if err != nil {
		slog.Error("publish notification", "error", err)
	}
}

func NotifyUser(
	ctx context.Context,
	pub Publisher,
	organizationID, recipientID uuid.UUID,
	kind, entityType, summary string,
	entityID uuid.UUID,
) {
	if pub == nil || recipientID == uuid.Nil {
		return
	}

	err := pub.Publish(ctx, Message{
		OrganizationID:  organizationID,
		Kind:            kind,
		EntityType:      entityType,
		EntityID:        entityID,
		Summary:         activity.Summary(summary),
		RecipientUserID: recipientID,
	})
	if err != nil {
		slog.Error("publish notification", "error", err)
	}
}
