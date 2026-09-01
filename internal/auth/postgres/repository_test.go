package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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

func createTestUser(t *testing.T, db *pgxpool.Pool) uuid.UUID {
	t.Helper()

	id := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
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
		id,
		"john@example.com",
		"hashed-password",
		now,
		now,
	)

	require.NoError(t, err)

	return id
}

func newTestRefreshToken(userID uuid.UUID) *auth.RefreshToken {
	now := time.Now().UTC()

	return &auth.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: "refresh-token-hash",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		RevokedAt: nil,
	}
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	token := newTestRefreshToken(userID)

	err := repo.Create(ctx, token)

	require.NoError(t, err)

	var (
		id        uuid.UUID
		gotUserID uuid.UUID
		tokenHash string
		expiresAt time.Time
		createdAt time.Time
		revokedAt *time.Time
	)

	err = db.QueryRow(
		ctx,
		`
			SELECT
				id,
				user_id,
				token_hash,
				expires_at,
				created_at,
				revoked_at
			FROM refresh_tokens
			WHERE id = $1
		`,
		token.ID,
	).Scan(
		&id,
		&gotUserID,
		&tokenHash,
		&expiresAt,
		&createdAt,
		&revokedAt,
	)

	require.NoError(t, err)
	require.Equal(t, token.ID, id)
	require.Equal(t, token.UserID, gotUserID)
	require.Equal(t, token.TokenHash, tokenHash)
	require.WithinDuration(t, token.ExpiresAt, expiresAt, time.Microsecond)
	require.WithinDuration(t, token.CreatedAt, createdAt, time.Microsecond)
	require.Nil(t, revokedAt)
}

func TestRefreshTokenRepository_Create_WithRevokedAt(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	token := newTestRefreshToken(userID)

	revokedAt := time.Now().UTC()
	token.RevokedAt = &revokedAt

	err := repo.Create(ctx, token)

	require.NoError(t, err)

	actual, err := repo.GetByHash(ctx, token.TokenHash)

	require.NoError(t, err)
	require.NotNil(t, actual.RevokedAt)
	require.WithinDuration(
		t,
		revokedAt,
		*actual.RevokedAt,
		time.Microsecond,
	)
}

func TestRefreshTokenRepository_GetByHash(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	expected := newTestRefreshToken(userID)

	require.NoError(t, repo.Create(ctx, expected))

	actual, err := repo.GetByHash(ctx, expected.TokenHash)

	require.NoError(t, err)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.UserID, actual.UserID)
	require.Equal(t, expected.TokenHash, actual.TokenHash)
	require.WithinDuration(
		t,
		expected.ExpiresAt,
		actual.ExpiresAt,
		time.Microsecond,
	)
	require.WithinDuration(
		t,
		expected.CreatedAt,
		actual.CreatedAt,
		time.Microsecond,
	)
	require.Nil(t, actual.RevokedAt)
}

func TestRefreshTokenRepository_GetByHash_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByHash(ctx, "missing-token-hash")

	require.Error(t, err)
	require.True(t, errors.Is(err, auth.ErrInvalidToken))
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	token := newTestRefreshToken(userID)

	require.NoError(t, repo.Create(ctx, token))

	err := repo.Revoke(ctx, token.ID)

	require.NoError(t, err)

	actual, err := repo.GetByHash(ctx, token.TokenHash)

	require.NoError(t, err)
	require.NotNil(t, actual.RevokedAt)
}

func TestRefreshTokenRepository_Revoke_AlreadyRevoked(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	token := newTestRefreshToken(userID)

	require.NoError(t, repo.Create(ctx, token))
	require.NoError(t, repo.Revoke(ctx, token.ID))

	err := repo.Revoke(ctx, token.ID)

	require.Error(t, err)
	require.True(t, errors.Is(err, auth.ErrTokenRevoked))
}

func TestRefreshTokenRepository_Revoke_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.Revoke(ctx, uuid.New())

	require.Error(t, err)
	require.True(t, errors.Is(err, auth.ErrTokenRevoked))
}

func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	token1 := newTestRefreshToken(userID)
	token1.TokenHash = "refresh-token-hash-1"

	token2 := newTestRefreshToken(userID)
	token2.TokenHash = "refresh-token-hash-2"

	require.NoError(t, repo.Create(ctx, token1))
	require.NoError(t, repo.Create(ctx, token2))

	err := repo.RevokeAllForUser(ctx, userID)

	require.NoError(t, err)

	actual1, err := repo.GetByHash(ctx, token1.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, actual1.RevokedAt)

	actual2, err := repo.GetByHash(ctx, token2.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, actual2.RevokedAt)
}

func TestRefreshTokenRepository_RevokeAllForUser_DoesNotAffectOtherUsers(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	user1ID := createTestUser(t, db)

	user2ID := uuid.New()
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
		user2ID,
		"jane@example.com",
		"hashed-password",
		now,
		now,
	)
	require.NoError(t, err)

	token1 := newTestRefreshToken(user1ID)
	token1.TokenHash = "user-1-token"

	token2 := newTestRefreshToken(user2ID)
	token2.TokenHash = "user-2-token"

	require.NoError(t, repo.Create(ctx, token1))
	require.NoError(t, repo.Create(ctx, token2))

	err = repo.RevokeAllForUser(ctx, user1ID)

	require.NoError(t, err)

	user1Token, err := repo.GetByHash(ctx, token1.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, user1Token.RevokedAt)

	user2Token, err := repo.GetByHash(ctx, token2.TokenHash)
	require.NoError(t, err)
	require.Nil(t, user2Token.RevokedAt)
}

func TestRefreshTokenRepository_CascadeDeleteWithUser(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)
	token := newTestRefreshToken(userID)

	require.NoError(t, repo.Create(ctx, token))

	_, err := db.Exec(
		ctx,
		`DELETE FROM users WHERE id = $1`,
		userID,
	)
	require.NoError(t, err)

	_, err = repo.GetByHash(ctx, token.TokenHash)

	require.Error(t, err)
	require.True(t, errors.Is(err, auth.ErrInvalidToken))
}
