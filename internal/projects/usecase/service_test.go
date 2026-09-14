package usecase_test

import (
	"context"
	"testing"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
	projectusecase "github.com/chuuch/gorest/internal/projects/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupProjectTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
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

func setupProjectService(db *pgxpool.Pool) projectusecase.Service {
	return projectusecase.NewService(
		projectpostgres.NewRepository(db),
		clientpostgres.NewRepository(db),
	)
}

func TestProjectService_CreateAndList(t *testing.T) {
	db, cleanup := setupProjectTestDatabase(t)
	defer cleanup()

	service := setupProjectService(db)
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")

	project, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		projectdomain.CreateProjectRequest{
			Name:  "Website",
			Notes: "Launch",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Website", project.Name)
	require.Equal(t, clientID, project.ClientID)

	own, err := service.List(context.Background(), organizationID, clientID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	_, err = service.List(context.Background(), otherOrganizationID, clientID)
	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)

	other, err := service.List(context.Background(), otherOrganizationID, otherClientID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestProjectService_Create_AdminCanAdd(t *testing.T) {
	db, cleanup := setupProjectTestDatabase(t)
	defer cleanup()

	service := setupProjectService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	project, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleAdmin,
		projectdomain.CreateProjectRequest{Name: "Website"},
	)

	require.NoError(t, err)
	require.Equal(t, "Website", project.Name)
}

func TestProjectService_Create_Forbidden(t *testing.T) {
	db, cleanup := setupProjectTestDatabase(t)
	defer cleanup()

	service := setupProjectService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	_, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleMember,
		projectdomain.CreateProjectRequest{Name: "Website"},
	)

	require.ErrorIs(t, err, projectdomain.ErrForbidden)
}

func TestProjectService_Create_NameExists(t *testing.T) {
	db, cleanup := setupProjectTestDatabase(t)
	defer cleanup()

	service := setupProjectService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	_, err := service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		projectdomain.CreateProjectRequest{Name: "Website"},
	)
	require.NoError(t, err)

	_, err = service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		projectdomain.CreateProjectRequest{Name: "Website"},
	)

	require.ErrorIs(t, err, projectdomain.ErrProjectNameExists)
}

func TestProjectService_Create_ClientNotFound(t *testing.T) {
	db, cleanup := setupProjectTestDatabase(t)
	defer cleanup()

	service := setupProjectService(db)
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		orgdomain.RoleOwner,
		projectdomain.CreateProjectRequest{Name: "Website"},
	)

	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)
}
