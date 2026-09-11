package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	"github.com/chuuch/gorest/internal/auth/repository"
	"github.com/chuuch/gorest/internal/auth/security"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
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
	testAccessTokenSecret = "test-access-token-secret"
	testIssuer            = "gorest-test"
	testAccessTokenTTL    = 15 * time.Minute
	testRefreshTokenTTL   = 24 * time.Hour
	testBcryptCost        = 4
	testPassword          = "password123"
	testOrganizationName  = "Acme"
)

func setupAuthTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			role TEXT NOT NULL CHECK (role IN ('admin', 'member')),
			created_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT memberships_user_id_unique UNIQUE (user_id),
			CONSTRAINT memberships_org_user_unique UNIQUE (organization_id, user_id)
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

	_, err = db.Exec(ctx, `
		CREATE INDEX idx_refresh_tokens_user_id
		ON refresh_tokens(user_id)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE INDEX idx_refresh_tokens_expires_at
		ON refresh_tokens(expires_at)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func testRegisterRequest() authdomain.RegisterRequest {
	return authdomain.RegisterRequest{
		Email:            "john@example.com",
		Password:         testPassword,
		OrganizationName: testOrganizationName,
	}
}

type authTestDependencies struct {
	service       authusecase.Service
	users         userusecase.Service
	refreshTokens repository.RefreshTokenRepository
	tokens        security.TokenManager
}

func setupAuthService(t *testing.T, db *pgxpool.Pool) authTestDependencies {
	t.Helper()

	userRepository := userpostgres.NewRepository(db)

	passwordHasher := password.NewBcryptHasher(testBcryptCost)

	userService := userusecase.NewService(
		userRepository,
		passwordHasher,
	)

	refreshTokenRepository := authpostgres.NewRepository(db)
	organizationRepository := orgpostgres.NewOrganizationRepository(db)
	membershipRepository := orgpostgres.NewMembershipRepository(db)

	tokenManager := security.NewJwtManager(
		testAccessTokenSecret,
		testIssuer,
		testAccessTokenTTL,
	)

	authService := authusecase.NewService(
		userService,
		organizationRepository,
		membershipRepository,
		refreshTokenRepository,
		tokenManager,
		passwordHasher,
		db,
		testAccessTokenTTL,
		testRefreshTokenTTL,
	)

	return authTestDependencies{
		service:       authService,
		users:         userService,
		refreshTokens: refreshTokenRepository,
		tokens:        tokenManager,
	}
}

func TestAuthService_Register(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	response, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)
	require.NotEmpty(t, response.AccessToken)
	require.NotEmpty(t, response.RefreshToken)
	require.NotNil(t, response.User)
	require.Equal(t, "john@example.com", response.User.Email)
	require.NotNil(t, response.Organization)
	require.Equal(t, testOrganizationName, response.Organization.Name)

	createdUser, err := deps.users.GetByEmail(
		ctx,
		"john@example.com",
	)

	require.NoError(t, err)
	require.Equal(t, "john@example.com", createdUser.Email)
	require.Equal(t, createdUser.ID, response.User.ID)
	require.NotEqual(t, testPassword, createdUser.PasswordHash)

	require.NoError(
		t,
		password.NewBcryptHasher(testBcryptCost).Compare(
			testPassword,
			createdUser.PasswordHash,
		),
	)

	claims, err := deps.tokens.ParseAccessToken(response.AccessToken)

	require.NoError(t, err)
	require.Equal(t, createdUser.ID, claims.UserID)
	require.Equal(t, response.Organization.ID, claims.OrganizationID)

	tokenHash := deps.tokens.HashRefreshToken(response.RefreshToken)

	storedToken, err := deps.refreshTokens.GetByHash(
		ctx,
		tokenHash,
	)

	require.NoError(t, err)
	require.Equal(t, createdUser.ID, storedToken.UserID)
	require.Equal(t, tokenHash, storedToken.TokenHash)
	require.Nil(t, storedToken.RevokedAt)
	require.True(t, storedToken.ExpiresAt.After(time.Now().UTC()))
}

func TestAuthService_Register_EmailAlreadyExists(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	_, err = deps.service.Register(ctx, authdomain.RegisterRequest{
		Email:            "john@example.com",
		Password:         "another-password",
		OrganizationName: "Other",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, userdomain.ErrEmailAlreadyExists))
}

func TestAuthService_Login(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	response, err := deps.service.Login(ctx, authdomain.LoginRequest{
		Email:    "john@example.com",
		Password: testPassword,
	})

	require.NoError(t, err)
	require.NotEmpty(t, response.AccessToken)
	require.NotEmpty(t, response.RefreshToken)
	require.NotNil(t, response.User)
	require.Equal(t, "john@example.com", response.User.Email)
	require.NotNil(t, response.Organization)
	require.Equal(t, testOrganizationName, response.Organization.Name)

	claims, err := deps.tokens.ParseAccessToken(response.AccessToken)

	require.NoError(t, err)
	require.Equal(t, response.User.ID, claims.UserID)
	require.Equal(t, response.Organization.ID, claims.OrganizationID)
	require.Equal(t, testIssuer, claims.Issuer)
	require.True(t, claims.ExpiresAt.After(time.Now().UTC()))

	tokenHash := deps.tokens.HashRefreshToken(response.RefreshToken)

	storedToken, err := deps.refreshTokens.GetByHash(
		ctx,
		tokenHash,
	)

	require.NoError(t, err)
	require.Equal(t, claims.UserID, storedToken.UserID)
	require.Nil(t, storedToken.RevokedAt)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	_, err = deps.service.Login(ctx, authdomain.LoginRequest{
		Email:    "john@example.com",
		Password: "wrong-password",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidCredentials))
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Login(ctx, authdomain.LoginRequest{
		Email:    "missing@example.com",
		Password: testPassword,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidCredentials))
}

func TestAuthService_Refresh(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	initial, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	initialHash := deps.tokens.HashRefreshToken(
		initial.RefreshToken,
	)

	initialStored, err := deps.refreshTokens.GetByHash(
		ctx,
		initialHash,
	)

	require.NoError(t, err)
	require.Nil(t, initialStored.RevokedAt)

	refreshed, err := deps.service.Refresh(
		ctx,
		initial.RefreshToken,
	)

	require.NoError(t, err)
	require.NotEmpty(t, refreshed.AccessToken)
	require.NotEmpty(t, refreshed.RefreshToken)
	require.NotNil(t, refreshed.User)
	require.Equal(t, "john@example.com", refreshed.User.Email)
	require.Equal(t, initial.User.ID, refreshed.User.ID)
	require.NotNil(t, refreshed.Organization)
	require.Equal(t, initial.Organization.ID, refreshed.Organization.ID)

	require.NotEqual(
		t,
		initial.RefreshToken,
		refreshed.RefreshToken,
	)

	initialStored, err = deps.refreshTokens.GetByHash(
		ctx,
		initialHash,
	)

	require.NoError(t, err)
	require.NotNil(t, initialStored.RevokedAt)

	newHash := deps.tokens.HashRefreshToken(
		refreshed.RefreshToken,
	)

	newStored, err := deps.refreshTokens.GetByHash(
		ctx,
		newHash,
	)

	require.NoError(t, err)
	require.Equal(t, initialStored.UserID, newStored.UserID)
	require.Nil(t, newStored.RevokedAt)
	require.True(t, newStored.ExpiresAt.After(time.Now().UTC()))
}

func TestAuthService_Refresh_RevokedToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	initial, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	_, err = deps.service.Refresh(
		ctx,
		initial.RefreshToken,
	)

	require.NoError(t, err)

	_, err = deps.service.Refresh(
		ctx,
		initial.RefreshToken,
	)

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrTokenRevoked))
}

func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	userID := uuid.New()
	organizationID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		ctx,
		`
			INSERT INTO users (
				id,
				email,
				password_hash,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
		userID,
		"john@example.com",
		"hashed-password",
		now,
		now,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		ctx,
		`
			INSERT INTO organizations (
				id,
				name,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		testOrganizationName,
		now,
		now,
	)
	require.NoError(t, err)

	_, err = db.Exec(
		ctx,
		`
			INSERT INTO memberships (
				id,
				organization_id,
				user_id,
				role,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
		uuid.New(),
		organizationID,
		userID,
		"admin",
		now,
	)
	require.NoError(t, err)

	refreshToken, err := deps.tokens.GenerateRefreshToken()
	require.NoError(t, err)

	token := &authdomain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: deps.tokens.HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	require.NoError(
		t,
		deps.refreshTokens.Create(ctx, token),
	)

	_, err = deps.service.Refresh(ctx, refreshToken)

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrTokenExpired))
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Refresh(
		ctx,
		"invalid-refresh-token",
	)

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidToken))
}

func TestAuthService_Refresh_EmptyToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	_, err := deps.service.Refresh(ctx, "")

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidToken))
}

func TestAuthService_Logout(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	response, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	tokenHash := deps.tokens.HashRefreshToken(
		response.RefreshToken,
	)

	storedToken, err := deps.refreshTokens.GetByHash(
		ctx,
		tokenHash,
	)

	require.NoError(t, err)
	require.Nil(t, storedToken.RevokedAt)

	err = deps.service.Logout(
		ctx,
		response.RefreshToken,
	)

	require.NoError(t, err)

	storedToken, err = deps.refreshTokens.GetByHash(
		ctx,
		tokenHash,
	)

	require.NoError(t, err)
	require.NotNil(t, storedToken.RevokedAt)
}

func TestAuthService_Logout_AlreadyRevoked(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	response, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	require.NoError(
		t,
		deps.service.Logout(ctx, response.RefreshToken),
	)

	err = deps.service.Logout(ctx, response.RefreshToken)

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrTokenRevoked))
}

func TestAuthService_Logout_InvalidToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	err := deps.service.Logout(
		ctx,
		"invalid-refresh-token",
	)

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidToken))
}

func TestAuthService_Logout_EmptyToken(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	err := deps.service.Logout(ctx, "")

	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidToken))
}

func TestAuthService_Me(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	registered, err := deps.service.Register(ctx, testRegisterRequest())

	require.NoError(t, err)

	response, err := deps.service.Me(ctx, registered.User.ID)

	require.NoError(t, err)
	require.NotEmpty(t, response.AccessToken)
	require.Empty(t, response.RefreshToken)
	require.NotNil(t, response.User)
	require.Equal(t, registered.User.ID, response.User.ID)
	require.Equal(t, "john@example.com", response.User.Email)
	require.NotNil(t, response.Organization)
	require.Equal(t, registered.Organization.ID, response.Organization.ID)
	require.Equal(t, testOrganizationName, response.Organization.Name)

	claims, err := deps.tokens.ParseAccessToken(response.AccessToken)

	require.NoError(t, err)
	require.Equal(t, registered.User.ID, claims.UserID)
	require.Equal(t, registered.Organization.ID, claims.OrganizationID)
}
