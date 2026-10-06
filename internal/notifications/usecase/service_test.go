package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/notifications"
	"github.com/chuuch/gorest/internal/notifications/domain"
	notificationpostgres "github.com/chuuch/gorest/internal/notifications/postgres"
	notificationsusecase "github.com/chuuch/gorest/internal/notifications/usecase"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupNotificationTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			email TEXT NOT NULL,
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
		CREATE TABLE memberships (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE notifications (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			actor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id UUID NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			read_at TIMESTAMPTZ,
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

func seedUser(t *testing.T, db *pgxpool.Pool, email, displayName string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		userID,
		email,
		displayName,
		"hash",
		now,
		now,
	)
	require.NoError(t, err)

	return userID
}

func seedOrganization(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()

	organizationID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO organizations (id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		name,
		now,
		now,
	)
	require.NoError(t, err)

	return organizationID
}

func seedMembership(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, userID uuid.UUID,
	role string,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO memberships (id, organization_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		uuid.New(),
		organizationID,
		userID,
		role,
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func setupNotificationService(db *pgxpool.Pool) notificationsusecase.Service {
	return notificationsusecase.NewService(
		notificationpostgres.NewRepository(db),
		orgpostgres.NewMembershipRepository(db),
	)
}

func TestPublish_StaffSkipsActor(t *testing.T) {
	db, cleanup := setupNotificationTestDatabase(t)
	defer cleanup()

	service := setupNotificationService(db)
	adaID := seedUser(t, db, "ada@example.com", "Ada")
	benID := seedUser(t, db, "ben@example.com", "Ben")
	organizationID := seedOrganization(t, db, "Acme")
	seedMembership(t, db, organizationID, adaID, "owner")
	seedMembership(t, db, organizationID, benID, "member")

	ctx := requestcontext.WithUserID(context.Background(), adaID)
	ticketID := uuid.New()

	require.NoError(t, service.Publish(ctx, notifications.Message{
		OrganizationID: organizationID,
		Kind:           domain.KindTicketOpened,
		EntityType:     domain.EntityTicket,
		EntityID:       ticketID,
		Summary:        "Login broken",
		Staff:          true,
	}))

	items, next, err := service.List(ctx, organizationID, benID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, items, 1)
	require.Equal(t, benID, items[0].RecipientID)
	require.Equal(t, adaID, items[0].ActorID)
	require.Equal(t, "ada@example.com", items[0].ActorEmail)
	require.Equal(t, "Ada", items[0].ActorDisplayName)
	require.Equal(t, domain.KindTicketOpened, items[0].Kind)
	require.Equal(t, "Login broken", items[0].Summary)
	require.Nil(t, items[0].ReadAt)

	own, next, err := service.List(ctx, organizationID, adaID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Empty(t, own)
}

func TestPublish_UserSkipsActor(t *testing.T) {
	db, cleanup := setupNotificationTestDatabase(t)
	defer cleanup()

	service := setupNotificationService(db)
	patID := seedUser(t, db, "pat@example.com", "Pat")
	organizationID := seedOrganization(t, db, "Acme")
	ctx := requestcontext.WithUserID(context.Background(), patID)

	require.NoError(t, service.Publish(ctx, notifications.Message{
		OrganizationID:  organizationID,
		Kind:            domain.KindTicketStaffComment,
		EntityType:      domain.EntityTicketComment,
		EntityID:        uuid.New(),
		Summary:         "Looking now",
		RecipientUserID: patID,
	}))

	items, next, err := service.List(ctx, organizationID, patID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Empty(t, items)
}

func TestPublish_UserNotifiesRecipient(t *testing.T) {
	db, cleanup := setupNotificationTestDatabase(t)
	defer cleanup()

	service := setupNotificationService(db)
	adaID := seedUser(t, db, "ada@example.com", "Ada")
	patID := seedUser(t, db, "pat@example.com", "Pat")
	organizationID := seedOrganization(t, db, "Acme")
	ctx := requestcontext.WithUserID(context.Background(), adaID)
	ticketID := uuid.New()

	require.NoError(t, service.Publish(ctx, notifications.Message{
		OrganizationID:  organizationID,
		Kind:            domain.KindTicketStaffComment,
		EntityType:      domain.EntityTicketComment,
		EntityID:        ticketID,
		Summary:         "Looking now",
		RecipientUserID: patID,
	}))

	items, next, err := service.List(ctx, organizationID, patID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, items, 1)
	require.Equal(t, patID, items[0].RecipientID)
	require.Equal(t, adaID, items[0].ActorID)
	require.Equal(t, domain.KindTicketStaffComment, items[0].Kind)

	own, next, err := service.List(ctx, organizationID, adaID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Empty(t, own)
}

func TestMarkRead(t *testing.T) {
	db, cleanup := setupNotificationTestDatabase(t)
	defer cleanup()

	service := setupNotificationService(db)
	adaID := seedUser(t, db, "ada@example.com", "Ada")
	benID := seedUser(t, db, "ben@example.com", "Ben")
	organizationID := seedOrganization(t, db, "Acme")
	seedMembership(t, db, organizationID, adaID, "owner")
	seedMembership(t, db, organizationID, benID, "member")

	ctx := requestcontext.WithUserID(context.Background(), adaID)
	require.NoError(t, service.Publish(ctx, notifications.Message{
		OrganizationID: organizationID,
		Kind:           domain.KindTicketOpened,
		EntityType:     domain.EntityTicket,
		EntityID:       uuid.New(),
		Summary:        "Login broken",
		Staff:          true,
	}))

	items, next, err := service.List(ctx, organizationID, benID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, items, 1)
	require.Nil(t, items[0].ReadAt)

	require.NoError(t, service.MarkRead(ctx, organizationID, benID, items[0].ID))

	read, next, err := service.List(ctx, organizationID, benID, 50, nil)
	require.NoError(t, err)
	require.Nil(t, next)
	require.Len(t, read, 1)
	require.NotNil(t, read[0].ReadAt)
}

func TestMarkRead_NotFound(t *testing.T) {
	db, cleanup := setupNotificationTestDatabase(t)
	defer cleanup()

	service := setupNotificationService(db)
	userID := seedUser(t, db, "ada@example.com", "Ada")
	organizationID := seedOrganization(t, db, "Acme")

	err := service.MarkRead(context.Background(), organizationID, userID, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotificationNotFound)
}
