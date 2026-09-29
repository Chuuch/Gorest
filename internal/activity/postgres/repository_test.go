package postgres_test

import (
	"context"
	"testing"
	"time"

	activitydomain "github.com/chuuch/gorest/internal/activity/domain"
	activitypostgres "github.com/chuuch/gorest/internal/activity/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gorest_test"),
		tcpostgres.WithUsername("gorest"),
		tcpostgres.WithPassword("gorest"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	require.NoError(t, db.Ping(ctx))

	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE organizations (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE activity_events (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			action TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id UUID NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func seedUserAndOrg(t *testing.T, db *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()

	userID := uuid.New()
	organizationID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		userID,
		"ada@example.com",
		"Ada",
		"hash",
		now,
		now,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		context.Background(),
		`
			INSERT INTO organizations (id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		"Acme",
		now,
		now,
	)
	require.NoError(t, err)

	return userID, organizationID
}

func TestActivityRepository_CreateAndList(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	userID, organizationID := seedUserAndOrg(t, db)
	repo := activitypostgres.NewRepository(db)
	ctx := context.Background()
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	event := &activitydomain.Event{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ActorID:        userID,
		Action:         activitydomain.ActionCreated,
		EntityType:     activitydomain.EntityTask,
		EntityID:       uuid.New(),
		Summary:        "Draw wireframes",
		CreatedAt:      createdAt,
	}

	require.NoError(t, repo.Create(ctx, event))

	listed, err := repo.ListByOrganizationID(ctx, organizationID, 50)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, event.ID, listed[0].ID)
	require.Equal(t, "ada@example.com", listed[0].ActorEmail)
	require.Equal(t, "Ada", listed[0].ActorDisplayName)
	require.Equal(t, "Draw wireframes", listed[0].Summary)
}

func TestActivityRepository_ListEmpty(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	_, organizationID := seedUserAndOrg(t, db)
	repo := activitypostgres.NewRepository(db)

	listed, err := repo.ListByOrganizationID(context.Background(), organizationID, 50)
	require.NoError(t, err)
	require.Empty(t, listed)
}
