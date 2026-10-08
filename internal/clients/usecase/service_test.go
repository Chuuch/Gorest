package usecase_test

import (
	"context"
	"testing"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/clients/domain"
	clientpostgres "github.com/chuuch/gorest/internal/clients/postgres"
	clientusecase "github.com/chuuch/gorest/internal/clients/usecase"
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
			legal_name TEXT NOT NULL DEFAULT '',
			registration_number TEXT NOT NULL DEFAULT '',
			vat_id TEXT NOT NULL DEFAULT '',
			address_line1 TEXT NOT NULL DEFAULT '',
			address_line2 TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			postal_code TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			default_vat_rate_bps INTEGER NOT NULL DEFAULT 2000,
			bank_iban TEXT NOT NULL DEFAULT '',
			bank_bic TEXT NOT NULL DEFAULT '',
			bank_name TEXT NOT NULL DEFAULT '',
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

	own, err := service.List(context.Background(), organizationID, "")
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, "Northwind", own[0].Name)

	other, err := service.List(context.Background(), otherOrganizationID, "")
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

func TestClientService_Update(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	created, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind", Notes: "Retail"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		orgdomain.RoleAdmin,
		clientdomain.UpdateClientRequest{Name: "Contoso", Notes: "Wholesale"},
	)

	require.NoError(t, err)
	require.Equal(t, "Contoso", updated.Name)
	require.Equal(t, "Wholesale", updated.Notes)
}

func TestClientService_Update_Forbidden(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	created, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		created.ID,
		orgdomain.RoleMember,
		clientdomain.UpdateClientRequest{Name: "Contoso"},
	)

	require.ErrorIs(t, err, clientdomain.ErrForbidden)
}

func TestClientService_Update_WrongOrgNotFound(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")

	created, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		otherOrganizationID,
		created.ID,
		orgdomain.RoleOwner,
		clientdomain.UpdateClientRequest{Name: "Contoso"},
	)

	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)
}

func TestClientService_Update_NameExists(t *testing.T) {
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

	other, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Contoso"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		other.ID,
		orgdomain.RoleOwner,
		clientdomain.UpdateClientRequest{Name: "Northwind"},
	)

	require.ErrorIs(t, err, clientdomain.ErrClientNameExists)
}

func TestClientService_Delete(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	created, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		created.ID,
		orgdomain.RoleAdmin,
	)
	require.NoError(t, err)

	clients, err := service.List(context.Background(), organizationID, "")
	require.NoError(t, err)
	require.Empty(t, clients)
}

func TestClientService_Delete_Forbidden(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	created, err := service.Create(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		clientdomain.CreateClientRequest{Name: "Northwind"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		created.ID,
		orgdomain.RoleMember,
	)

	require.ErrorIs(t, err, clientdomain.ErrForbidden)
}

func TestClientService_Delete_NotFound(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	err := service.Delete(
		context.Background(),
		organizationID,
		uuid.New(),
		orgdomain.RoleOwner,
	)

	require.ErrorIs(t, err, clientdomain.ErrClientNotFound)
}

func TestClientService_List_Search(t *testing.T) {
	db, cleanup := setupClientTestDatabase(t)
	defer cleanup()

	service := clientusecase.NewService(clientpostgres.NewRepository(db))
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(context.Background(), organizationID, orgdomain.RoleOwner, clientdomain.CreateClientRequest{
		Name:  "Northwind",
		Notes: "Retail",
	})
	require.NoError(t, err)

	_, err = service.Create(context.Background(), organizationID, orgdomain.RoleOwner, clientdomain.CreateClientRequest{
		Name:  "Contoso",
		Notes: "Software",
	})
	require.NoError(t, err)

	matched, err := service.List(context.Background(), organizationID, "north")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	require.Equal(t, "Northwind", matched[0].Name)

	byNotes, err := service.List(context.Background(), organizationID, "software")
	require.NoError(t, err)
	require.Len(t, byNotes, 1)
	require.Equal(t, "Contoso", byNotes[0].Name)

	none, err := service.List(context.Background(), organizationID, "zzz")
	require.NoError(t, err)
	require.Empty(t, none)
}
