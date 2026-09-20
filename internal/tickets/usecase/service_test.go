package usecase_test

import (
	"context"
	"testing"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	ticketusecase "github.com/chuuch/gorest/internal/tickets/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTicketTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT tickets_kind_check CHECK (kind IN ('bug', 'feature', 'question', 'other')),
			CONSTRAINT tickets_status_check CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
			CONSTRAINT tickets_title_check CHECK (char_length(title) >= 4 AND char_length(title) <= 100),
			CONSTRAINT tickets_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func seedUser(t *testing.T, db *pgxpool.Pool, email string) uuid.UUID {
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

func seedClient(
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
			INSERT INTO clients (
				id,
				organization_id,
				name,
				notes,
				created_at,
				updated_at
			)
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

func setupTicketService(db *pgxpool.Pool) ticketusecase.Service {
	return ticketusecase.NewService(
		ticketpostgres.NewRepository(db),
		clientpostgres.NewRepository(db),
	)
}

func TestTicketService_CreateAndList(t *testing.T) {
	db, cleanup := setupTicketTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	foreignClientID := seedClient(t, db, otherOrganizationID, "Northwind")

	ticket, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		userID,
		ticketdomain.CreateTicketRequest{
			Kind:  "bug",
			Title: "Login button broken",
			Body:  "Clicking Sign in does nothing on mobile.",
		},
	)
	require.NoError(t, err)
	require.Equal(t, ticketdomain.KindBug, ticket.Kind)
	require.Equal(t, ticketdomain.StatusOpen, ticket.Status)
	require.Equal(t, clientID, ticket.ClientID)
	require.Equal(t, userID, ticket.UserID)

	_, err = service.Create(
		context.Background(),
		organizationID,
		clientID,
		userID,
		ticketdomain.CreateTicketRequest{
			Kind:  "feature",
			Title: "Export invoices",
			Body:  "We need a CSV export from the billing page.",
		},
	)
	require.NoError(t, err)

	own, err := service.List(context.Background(), organizationID, clientID)
	require.NoError(t, err)
	require.Len(t, own, 2)
	require.Equal(t, "Export invoices", own[0].Title)

	other, err := service.List(context.Background(), organizationID, otherClientID)
	require.NoError(t, err)
	require.Empty(t, other)

	_, err = service.List(context.Background(), otherOrganizationID, clientID)
	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)

	foreign, err := service.List(context.Background(), otherOrganizationID, foreignClientID)
	require.NoError(t, err)
	require.Empty(t, foreign)
}

func TestTicketService_Create_ClientNotFound(t *testing.T) {
	db, cleanup := setupTicketTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		ticketdomain.CreateTicketRequest{
			Kind:  "question",
			Title: "Where are invoices",
			Body:  "Can we see last month somewhere?",
		},
	)
	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)
}

func TestTicketService_Update_Status(t *testing.T) {
	db, cleanup := setupTicketTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	created, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		userID,
		ticketdomain.CreateTicketRequest{
			Kind:  "bug",
			Title: "Login button broken",
			Body:  "Clicking Sign in does nothing on mobile.",
		},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		ticketdomain.UpdateTicketRequest{Status: "in_progress"},
	)
	require.NoError(t, err)
	require.Equal(t, ticketdomain.StatusInProgress, updated.Status)

	listed, err := service.List(context.Background(), organizationID, clientID)
	require.NoError(t, err)
	require.Equal(t, ticketdomain.StatusInProgress, listed[0].Status)
}

func TestTicketService_Update_WrongOrgNotFound(t *testing.T) {
	db, cleanup := setupTicketTestDatabase(t)
	defer cleanup()

	service := setupTicketService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")

	created, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		userID,
		ticketdomain.CreateTicketRequest{
			Kind:  "other",
			Title: "Need a call",
			Body:  "Can someone ring us this week?",
		},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		otherOrganizationID,
		created.ID,
		ticketdomain.UpdateTicketRequest{Status: "closed"},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}
