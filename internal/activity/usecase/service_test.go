package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/activity/domain"
	uc "github.com/chuuch/gorest/internal/activity/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	createFn func(context.Context, *domain.Event) error
	listFn   func(context.Context, uuid.UUID, int) ([]*domain.Event, error)
}

func (m *mockRepository) Create(ctx context.Context, event *domain.Event) error {
	return m.createFn(ctx, event)
}

func (m *mockRepository) ListByOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
) ([]*domain.Event, error) {
	return m.listFn(ctx, organizationID, limit)
}

func TestService_Record(t *testing.T) {
	t.Parallel()

	organizationID := uuid.New()
	actorID := uuid.New()
	entityID := uuid.New()

	repo := &mockRepository{
		createFn: func(_ context.Context, event *domain.Event) error {
			require.NotEqual(t, uuid.Nil, event.ID)
			require.Equal(t, organizationID, event.OrganizationID)
			require.Equal(t, actorID, event.ActorID)
			require.Equal(t, domain.ActionCreated, event.Action)
			require.Equal(t, domain.EntityTask, event.EntityType)
			require.Equal(t, entityID, event.EntityID)
			require.Equal(t, "Draw wireframes", event.Summary)
			require.False(t, event.CreatedAt.IsZero())
			return nil
		},
	}

	service := uc.NewService(repo)

	err := service.Record(context.Background(), domain.Event{
		OrganizationID: organizationID,
		ActorID:        actorID,
		Action:         domain.ActionCreated,
		EntityType:     domain.EntityTask,
		EntityID:       entityID,
		Summary:        "Draw wireframes",
	})
	require.NoError(t, err)
}

func TestService_List(t *testing.T) {
	t.Parallel()

	organizationID := uuid.New()
	expected := &domain.Event{
		ID:               uuid.New(),
		OrganizationID:   organizationID,
		ActorID:          uuid.New(),
		ActorEmail:       "ada@example.com",
		ActorDisplayName: "Ada",
		Action:           domain.ActionCreated,
		EntityType:       domain.EntityTask,
		EntityID:         uuid.New(),
		Summary:          "Draw wireframes",
		CreatedAt:        time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	repo := &mockRepository{
		listFn: func(_ context.Context, gotOrganizationID uuid.UUID, limit int) ([]*domain.Event, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, 50, limit)
			return []*domain.Event{expected}, nil
		},
	}

	service := uc.NewService(repo)

	events, err := service.List(context.Background(), organizationID, 50)
	require.NoError(t, err)
	require.Equal(t, []*domain.Event{expected}, events)
}
