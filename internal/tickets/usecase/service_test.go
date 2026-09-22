package usecase_test

import (
	"context"
	"testing"
	"time"

	ticketfiledomain "github.com/chuuch/gorest/internal/ticketfiles/domain"
	ticketfilepostgres "github.com/chuuch/gorest/internal/ticketfiles/postgres"
	ticketfileusecase "github.com/chuuch/gorest/internal/ticketfiles/usecase"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type mockStore struct {
	putURL string
	getURL string
}

func (m *mockStore) PresignPut(context.Context, string, string) (string, error) {
	return m.putURL, nil
}

func (m *mockStore) PresignGet(context.Context, string, string) (string, error) {
	return m.getURL, nil
}

func setupTicketFileTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE ticket_files (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
			uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			object_key TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT ticket_files_size_bytes_check CHECK (size_bytes >= 1 AND size_bytes <= 10485760)
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

func seedTicket(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID, userID uuid.UUID,
	title string,
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
		"Clicking Sign in does nothing on mobile.",
		now,
		now,
	)
	require.NoError(t, err)

	return ticketID
}

func setupTicketFileService(db *pgxpool.Pool) ticketfileusecase.Service {
	return ticketfileusecase.NewService(
		ticketfilepostgres.NewRepository(db),
		ticketpostgres.NewRepository(db),
		&mockStore{putURL: "http://minio/put", getURL: "http://minio/get"},
	)
}

func TestTicketFileService_CreateAndList(t *testing.T) {
	db, cleanup := setupTicketFileTestDatabase(t)
	defer cleanup()

	service := setupTicketFileService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")
	otherTicketID := seedTicket(t, db, organizationID, otherClientID, userID, "Other login issue")

	created, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		clientID,
		ticketfiledomain.CreateFileRequest{
			Filename:    "bug.png",
			ContentType: "image/png",
			Size:        2048,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "bug.png", created.File.Filename)
	require.Equal(t, ticketID, created.File.TicketID)
	require.Equal(t, "http://minio/put", created.UploadURL)
	require.Contains(t, created.File.ObjectKey, "/tickets/")

	own, err := service.List(context.Background(), organizationID, ticketID, clientID)
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, "http://minio/get", own[0].DownloadURL)

	staff, err := service.List(context.Background(), organizationID, ticketID, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, staff, 1)

	_, err = service.List(context.Background(), organizationID, ticketID, otherClientID)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)

	other, err := service.List(context.Background(), organizationID, otherTicketID, otherClientID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestTicketFileService_Create_OtherClientNotFound(t *testing.T) {
	db, cleanup := setupTicketFileTestDatabase(t)
	defer cleanup()

	service := setupTicketFileService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	_, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		otherClientID,
		ticketfiledomain.CreateFileRequest{
			Filename:    "bug.png",
			ContentType: "image/png",
			Size:        2048,
		},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}

func TestTicketFileService_Create_InvalidFilename(t *testing.T) {
	db, cleanup := setupTicketFileTestDatabase(t)
	defer cleanup()

	service := setupTicketFileService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	_, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		uuid.Nil,
		ticketfiledomain.CreateFileRequest{
			Filename:    "../bug.png",
			ContentType: "image/png",
			Size:        2048,
		},
	)
	require.ErrorIs(t, err, ticketfiledomain.ErrInvalidFilename)
}

func TestTicketFileService_Create_UnsupportedType(t *testing.T) {
	db, cleanup := setupTicketFileTestDatabase(t)
	defer cleanup()

	service := setupTicketFileService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	_, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		uuid.Nil,
		ticketfiledomain.CreateFileRequest{
			Filename:    "payload.exe",
			ContentType: "application/octet-stream",
			Size:        2048,
		},
	)
	require.ErrorIs(t, err, ticketfiledomain.ErrUnsupportedContentType)
}

func TestTicketFileService_Create_TicketNotFound(t *testing.T) {
	db, cleanup := setupTicketFileTestDatabase(t)
	defer cleanup()

	service := setupTicketFileService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		uuid.Nil,
		ticketfiledomain.CreateFileRequest{
			Filename:    "bug.png",
			ContentType: "image/png",
			Size:        2048,
		},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}
