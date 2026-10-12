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
	ClientID        *uuid.UUID
	ProjectID       *uuid.UUID
	Summary         string
	RecipientUserID uuid.UUID
	ActorUserID     uuid.UUID
	IncludeActor    bool
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
	clientID, projectID *uuid.UUID,
) {
	if pub == nil {
		return
	}

	err := pub.Publish(ctx, Message{
		OrganizationID: organizationID,
		Kind:           kind,
		EntityType:     entityType,
		EntityID:       entityID,
		ClientID:       clientID,
		ProjectID:      projectID,
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
	clientID, projectID *uuid.UUID,
) {
	if pub == nil || recipientID == uuid.Nil {
		return
	}

	err := pub.Publish(ctx, Message{
		OrganizationID:  organizationID,
		Kind:            kind,
		EntityType:      entityType,
		EntityID:        entityID,
		ClientID:        clientID,
		ProjectID:       projectID,
		Summary:         activity.Summary(summary),
		RecipientUserID: recipientID,
	})
	if err != nil {
		slog.Error("publish notification", "error", err)
	}
}

// NotifyStaffAs notifies every staff member (including actorID) using actorID as the
// notification actor. Use for public/system events where context has no user.
func NotifyStaffAs(
	ctx context.Context,
	pub Publisher,
	organizationID, actorID uuid.UUID,
	kind, entityType, summary string,
	entityID uuid.UUID,
	clientID, projectID *uuid.UUID,
) {
	if pub == nil || actorID == uuid.Nil {
		return
	}

	err := pub.Publish(ctx, Message{
		OrganizationID: organizationID,
		Kind:           kind,
		EntityType:     entityType,
		EntityID:       entityID,
		ClientID:       clientID,
		ProjectID:      projectID,
		Summary:        activity.Summary(summary),
		ActorUserID:    actorID,
		IncludeActor:   true,
		Staff:          true,
	})
	if err != nil {
		slog.Error("publish notification", "error", err)
	}
}
