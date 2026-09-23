package usecase_test

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/auth/security"
	clientpostgres "github.com/chuuch/gorest/internal/client/postgres"
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	clientuserpostgres "github.com/chuuch/gorest/internal/clientusers/postgres"
	clientuserusecase "github.com/chuuch/gorest/internal/clientusers/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testBcryptCost      = 4
	testPassword        = "password123"
	testAccessTokenTTL  = 15 * time.Minute
	testRefreshTokenTTL = 24 * time.Hour
	testAccessSecret    = "test-access-token-secret"
	testIssuer          = "gorest-test"
)

func setupClientUserTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
			created_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT memberships_user_id_unique UNIQUE (user_id),
			CONSTRAINT memberships_org_user_unique UNIQUE (organization_id, user_id)
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

	_, err = db.Exec(ctx, `
		CREATE TABLE client_users (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			created_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT client_users_user_id_unique UNIQUE (user_id)
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE refresh_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			revoked_at TIMESTAMPTZ
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

type clientUserTestDependencies struct {
	service clientuserusecase.Service
	users   userusecase.Service
	tokens  security.TokenManager
}

func setupClientUserService(t *testing.T, db *pgxpool.Pool) clientUserTestDependencies {
	t.Helper()

	passwordHasher := password.NewBcryptHasher(testBcryptCost)
	userService := userusecase.NewService(userpostgres.NewRepository(db), passwordHasher)
	tokenManager := security.NewJwtManager(testAccessSecret, testIssuer, testAccessTokenTTL)

	return clientUserTestDependencies{
		service: clientuserusecase.NewService(
			userService,
			clientpostgres.NewRepository(db),
			clientuserpostgres.NewRepository(db),
			orgpostgres.NewMembershipRepository(db),
			orgpostgres.NewOrganizationRepository(db),
			authpostgres.NewRepository(db),
			tokenManager,
			passwordHasher,
			db,
			testRefreshTokenTTL,
		),
		users:  userService,
		tokens: tokenManager,
	}
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

func seedStaff(
	t *testing.T,
	db *pgxpool.Pool,
	users userusecase.Service,
	organizationID uuid.UUID,
	email string,
	role orgdomain.Role,
) *userdomain.User {
	t.Helper()

	user, err := users.Create(context.Background(), userdomain.CreateUserRequest{
		Email:    email,
		Password: testPassword,
	})
	require.NoError(t, err)

	err = orgpostgres.NewMembershipRepository(db).Create(context.Background(), &orgdomain.Membership{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		UserID:         user.ID,
		Role:           role,
		CreatedAt:      time.Now().UTC(),
	})
	require.NoError(t, err)

	return user
}

func TestClientUserService_CreateAndList(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")

	member, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)

	require.NoError(t, err)
	require.Equal(t, "pat@northwind.test", member.Email)
	require.Equal(t, clientID, member.ClientID)

	own, err := deps.service.List(context.Background(), organizationID, clientID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	other, err := deps.service.List(context.Background(), otherOrganizationID, otherClientID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestClientUserService_Create_MemberForbidden(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	_, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleMember,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)

	require.ErrorIs(t, err, clientuserdomain.ErrForbidden)
}

func TestClientUserService_Create_UserIsStaff(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	_ = seedStaff(t, db, deps.users, organizationID, "ada@example.com", orgdomain.RoleMember)

	_, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "ada@example.com",
			Password: testPassword,
		},
	)

	require.ErrorIs(t, err, clientuserdomain.ErrUserIsStaff)
}

func TestClientUserService_Login(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	_, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	result, err := deps.service.Login(
		context.Background(),
		clientuserdomain.LoginRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)

	require.NoError(t, err)
	require.Equal(t, clientuserdomain.RoleClient, result.Role)
	require.Equal(t, clientID, result.Client.ID)
	require.NotEmpty(t, result.RefreshToken)

	claims, err := deps.tokens.ParseAccessToken(result.AccessToken)
	require.NoError(t, err)
	require.Equal(t, clientuserdomain.RoleClient, claims.Role)
	require.Equal(t, clientID, claims.ClientID)
}

func TestClientUserService_Login_StaffRejected(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	_ = seedStaff(t, db, deps.users, organizationID, "owner@example.com", orgdomain.RoleOwner)

	_, err := deps.service.Login(
		context.Background(),
		clientuserdomain.LoginRequest{
			Email:    "owner@example.com",
			Password: testPassword,
		},
	)

	require.ErrorIs(t, err, authdomain.ErrInvalidCredentials)
}

func TestClientUserService_Delete(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	member, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	err = deps.service.Delete(
		context.Background(),
		organizationID,
		clientID,
		member.UserID,
		orgdomain.RoleAdmin,
	)
	require.NoError(t, err)

	listed, err := deps.service.List(context.Background(), organizationID, clientID)
	require.NoError(t, err)
	require.Empty(t, listed)

	_, err = deps.users.GetByID(context.Background(), member.UserID)
	require.NoError(t, err)
}

func TestClientUserService_Delete_LastAllowed(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	member, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	err = deps.service.Delete(
		context.Background(),
		organizationID,
		clientID,
		member.UserID,
		orgdomain.RoleOwner,
	)
	require.NoError(t, err)
}

func TestClientUserService_Delete_MemberForbidden(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	member, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	err = deps.service.Delete(
		context.Background(),
		organizationID,
		clientID,
		member.UserID,
		orgdomain.RoleMember,
	)
	require.ErrorIs(t, err, clientuserdomain.ErrForbidden)
}

func TestClientUserService_Delete_WrongClient(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")

	member, err := deps.service.Create(
		context.Background(),
		organizationID,
		clientID,
		orgdomain.RoleOwner,
		clientuserdomain.CreateClientUserRequest{
			Email:    "pat@northwind.test",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	err = deps.service.Delete(
		context.Background(),
		organizationID,
		otherClientID,
		member.UserID,
		orgdomain.RoleOwner,
	)
	require.ErrorIs(t, err, clientuserdomain.ErrClientUserNotFound)
}

func TestClientUserService_Delete_NotFound(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")

	err := deps.service.Delete(
		context.Background(),
		organizationID,
		clientID,
		uuid.New(),
		orgdomain.RoleOwner,
	)
	require.ErrorIs(t, err, clientuserdomain.ErrClientUserNotFound)
}
