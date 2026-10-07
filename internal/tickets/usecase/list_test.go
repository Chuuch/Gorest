package usecase_test

import (
	"context"
	"testing"
	"time"

	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	ticketusecase "github.com/chuuch/gorest/internal/tickets/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTicketListTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
		CREATE TABLE clients (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			legal_name TEXT NOT NULL DEFAULT '',
			vat_id TEXT NOT NULL DEFAULT '',
			address_line1 TEXT NOT NULL DEFAULT '',
			address_line2 TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			postal_code TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE tickets (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func setupTicketService(db *pgxpool.Pool) ticketusecase.Service {
	return ticketusecase.NewService(
		ticketpostgres.NewRepository(db),
		clientpostgres.NewRepository(db),
	)
}

func seedListUser(t *testing.T, db *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		userID,
		email,
		"hash",
		now,
		now,
	)
	require.NoError(t, err)

	return userID
}

func seedListOrganization(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
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

func seedListClient(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID uuid.UUID,
	name string,
) uuid.UUID {
	t.Helper()

	clientID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO clients (id, organization_id, name, notes, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		clientID,
		organizationID,
		name,
		"",
		now,
		now,
	)
	require.NoError(t, err)

	return clientID
}

func seedListTicket(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID, userID uuid.UUID,
	title, body string,
) uuid.UUID {
	t.Helper()

	ticketID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO tickets (
				id, organization_id, client_id, user_id, kind, status, title, body, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
		ticketID,
		organizationID,
		clientID,
		userID,
		"bug",
		"open",
		title,
		body,
		now,
		now,
	)
	require.NoError(t, err)

	return ticketID
}

func TestTicketService_List_Search(t *testing.T) {
	db, cleanup := setupTicketListTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	organizationID := seedListOrganization(t, db, "Acme")
	clientID := seedListClient(t, db, organizationID, "Northwind")
	userID := seedListUser(t, db, "pat@example.com")

	seedListTicket(
		t, db, organizationID, clientID, userID,
		"Login button broken",
		"Clicking Sign in does nothing on mobile.",
	)
	seedListTicket(
		t, db, organizationID, clientID, userID,
		"Invoice question",
		"Need help with billing export",
	)

	byTitle, err := service.List(context.Background(), organizationID, clientID, "login")
	require.NoError(t, err)
	require.Len(t, byTitle, 1)
	require.Equal(t, "Login button broken", byTitle[0].Title)

	byBody, err := service.List(context.Background(), organizationID, clientID, "billing")
	require.NoError(t, err)
	require.Len(t, byBody, 1)
	require.Equal(t, "Invoice question", byBody[0].Title)

	none, err := service.List(context.Background(), organizationID, clientID, "zzz")
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestTicketService_ListOrganization_Search(t *testing.T) {
	db, cleanup := setupTicketListTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	organizationID := seedListOrganization(t, db, "Acme")
	clientA := seedListClient(t, db, organizationID, "Northwind")
	clientB := seedListClient(t, db, organizationID, "Contoso")
	userID := seedListUser(t, db, "pat@example.com")

	seedListTicket(
		t, db, organizationID, clientA, userID,
		"Login button broken",
		"Clicking Sign in does nothing on mobile.",
	)
	seedListTicket(
		t, db, organizationID, clientB, userID,
		"Feature request",
		"Please add dark mode.",
	)

	matched, err := service.ListOrganization(context.Background(), organizationID, "login")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	require.Equal(t, clientA, matched[0].ClientID)

	none, err := service.ListOrganization(context.Background(), organizationID, "zzz")
	require.NoError(t, err)
	require.Empty(t, none)
}
