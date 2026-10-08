package activity

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	"github.com/google/uuid"
)

type Recorder interface {
	Record(ctx context.Context, event domain.Event) error
}

type Nop struct{}

func (Nop) Record(context.Context, domain.Event) error {
	return nil
}

func Summary(value string) string {
	trimmed := strings.TrimSpace(value)
	if utf8.RuneCountInString(trimmed) <= 80 {
		return trimmed
	}

	runes := []rune(trimmed)
	return string(runes[:80])
}

func Capture(
	ctx context.Context,
	rec Recorder,
	organizationID uuid.UUID,
	action, entityType, summary string,
	entityID uuid.UUID,
) {
	if rec == nil {
		return
	}

	actorID, ok := requestcontext.UserID(ctx)
	if !ok {
		return
	}

	err := rec.Record(ctx, domain.Event{
		OrganizationID: organizationID,
		ActorID:        actorID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		Summary:        Summary(summary),
	})
	if err != nil {
		slog.Error("record activity", "error", err)
	}
}
