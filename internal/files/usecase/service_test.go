package usecase_test

import (
	"context"
	"testing"
	"time"

	filedomain "github.com/chuuch/gorest/internal/files/domain"
	filepostgres "github.com/chuuch/gorest/internal/files/postgres"
	fileusecase "github.com/chuuch/gorest/internal/files/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
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

func setupFileTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
		CREATE TABLE projects (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT projects_client_name_unique UNIQUE (client_id, name)
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE files (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			object_key TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT files_size_bytes_check CHECK (size_bytes >= 1 AND size_bytes <= 10485760)
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
				id, organization_id, name, notes, created_at, updated_at
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

func seedProject(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID uuid.UUID,
	name string,
) uuid.UUID {
	t.Helper()

	projectID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO projects (
				id, organization_id, client_id, name, notes, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		projectID,
		organizationID,
		clientID,
		name,
		"",
		now,
		now,
	)
	require.NoError(t, err)

	return projectID
}

func setupFileService(db *pgxpool.Pool) fileusecase.Service {
	return fileusecase.NewService(
		filepostgres.NewRepository(db),
		projectpostgres.NewRepository(db),
		&mockStore{putURL: "http://minio/put", getURL: "http://minio/get"},
	)
}

func TestFileService_CreateAndList(t *testing.T) {
	db, cleanup := setupFileTestDatabase(t)
	defer cleanup()

	service := setupFileService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "Website")

	view, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		userID,
		orgdomain.RoleOwner,
		filedomain.CreateFileRequest{
			Filename:    "spec.pdf",
			ContentType: "application/pdf",
			Size:        2048,
		},
	)

	require.NoError(t, err)
	require.Equal(t, "spec.pdf", view.File.Filename)
	require.Equal(t, "http://minio/put", view.UploadURL)

	own, err := service.List(context.Background(), organizationID, projectID)
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, "http://minio/get", own[0].DownloadURL)

	_, err = service.List(context.Background(), otherOrganizationID, projectID)
	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)

	other, err := service.List(context.Background(), otherOrganizationID, otherProjectID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestFileService_Create_Forbidden(t *testing.T) {
	db, cleanup := setupFileTestDatabase(t)
	defer cleanup()

	service := setupFileService(db)
	userID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		userID,
		orgdomain.RoleMember,
		filedomain.CreateFileRequest{
			Filename:    "spec.pdf",
			ContentType: "application/pdf",
			Size:        2048,
		},
	)

	require.ErrorIs(t, err, filedomain.ErrForbidden)
}

func TestFileService_Create_UnsupportedContentType(t *testing.T) {
	db, cleanup := setupFileTestDatabase(t)
	defer cleanup()

	service := setupFileService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		userID,
		orgdomain.RoleOwner,
		filedomain.CreateFileRequest{
			Filename:    "payload.exe",
			ContentType: "application/x-msdownload",
			Size:        2048,
		},
	)

	require.ErrorIs(t, err, filedomain.ErrUnsupportedContentType)
}

func TestFileService_Create_InvalidFilename(t *testing.T) {
	db, cleanup := setupFileTestDatabase(t)
	defer cleanup()

	service := setupFileService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		userID,
		orgdomain.RoleOwner,
		filedomain.CreateFileRequest{
			Filename:    "../secret.pdf",
			ContentType: "application/pdf",
			Size:        2048,
		},
	)

	require.ErrorIs(t, err, filedomain.ErrInvalidFilename)
}

func TestFileService_Create_ProjectNotFound(t *testing.T) {
	db, cleanup := setupFileTestDatabase(t)
	defer cleanup()

	service := setupFileService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		orgdomain.RoleOwner,
		filedomain.CreateFileRequest{
			Filename:    "spec.pdf",
			ContentType: "application/pdf",
			Size:        2048,
		},
	)

	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
}
