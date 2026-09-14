package usecase_test

import (
	"context"
	"testing"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	clientusecase "github.com/chuuch/gorest/internal/client/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupClientTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT clients_org_name_unique UNIQUE (organization_id, name)
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

func TestClientService_CreateAndList(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")

	client, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{
			Name:  "Northwind",
			Notes: "Retail",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Northwind", client.Name)
	require.Equal(t, organizationID, client.OrganizationID)

	own, err := service.List(context.Background(), organizationID)
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, "Northwind", own[0].Name)

	other, err := service.List(context.Background(), otherOrganizationID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestClientService_Create_AdminCanAdd(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	client, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleAdmin,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)

	require.NoError(t, err)
	require.Equal(t, "Northwind", client.Name)
}

func TestClientService_Create_Forbidden(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleMember,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)

	require.ErrorIs(t, err, clientdomain.ErrForbidden)
}

func TestClientService_Create_NameExists(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)
	require.NoError(t, err)

	_, err = service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)

	require.ErrorIs(t, err, clientdomain.ErrClientNameExists)
}
