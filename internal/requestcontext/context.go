package requestcontext

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey         contextKey = "user_id"
	organizationIDKey contextKey = "organization_id"
)

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func WithOrganizationID(ctx context.Context, organizationID uuid.UUID) context.Context {
	return context.WithValue(ctx, organizationIDKey, organizationID)
}

func OrganizationID(ctx context.Context) (uuid.UUID, bool) {
	organizationID, ok := ctx.Value(organizationIDKey).(uuid.UUID)
	return organizationID, ok
}
