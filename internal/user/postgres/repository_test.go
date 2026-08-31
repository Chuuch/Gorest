package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("gorest_test"),
		postgres.WithUsername("gorest"),
		postgres.WithPassword("gorest"),
		postgres.BasicWaitStrategies(),
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

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}
	return db, cleanup
}

func newTestUser() *user.User {
	now := time.Now().UTC()

	return &user.User{
		ID:           uuid.New(),
		Email:        "john@example.com",
		PasswordHash: "hashed-password",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUserRepository_Create(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser()

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	var (
		id           uuid.UUID
		email        string
		passwordHash string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err = db.QueryRow(
		ctx,
		`
			SELECT
				id,
				email,
				password_hash,
				created_at,
				updated_at
			FROM users
			WHERE id = $1
		`,
		u.ID,
	).Scan(
		&id,
		&email,
		&passwordHash,
		&createdAt,
		&updatedAt,
	)

	require.NoError(t, err)
	require.Equal(t, u.ID, id)
	require.Equal(t, u.Email, email)
	require.Equal(t, u.PasswordHash, passwordHash)
	require.WithinDuration(t, u.CreatedAt, createdAt, time.Microsecond)
	require.WithinDuration(t, u.UpdatedAt, updatedAt, time.Microsecond)
}

func TestUserRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := newTestUser()

	require.NoError(t, repo.Create(ctx, expected))

	actual, err := repo.GetByID(ctx, expected.ID)

	require.NoError(t, err)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.PasswordHash, actual.PasswordHash)
	require.WithinDuration(t, expected.CreatedAt, actual.CreatedAt, time.Microsecond)
	require.WithinDuration(t, expected.UpdatedAt, actual.UpdatedAt, time.Microsecond)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())

	require.Error(t, err)
	require.True(t, errors.Is(err, user.ErrUserNotFound))
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := newTestUser()

	require.NoError(t, repo.Create(ctx, expected))

	actual, err := repo.GetByEmail(ctx, expected.Email)

	require.NoError(t, err)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.PasswordHash, actual.PasswordHash)
	require.WithinDuration(t, expected.CreatedAt, actual.CreatedAt, time.Microsecond)
	require.WithinDuration(t, expected.UpdatedAt, actual.UpdatedAt, time.Microsecond)
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "missing@example.com")

	require.Error(t, err)
	require.True(t, errors.Is(err, user.ErrUserNotFound))
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser()

	exists, err := repo.ExistsByEmail(ctx, u.Email)

	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, repo.Create(ctx, u))

	exists, err = repo.ExistsByEmail(ctx, u.Email)

	require.NoError(t, err)
	require.True(t, exists)
}

func TestUserRepository_Update(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser()

	require.NoError(t, repo.Create(ctx, u))

	updatedAt := time.Now().UTC()

	u.Email = "updated@example.com"
	u.PasswordHash = "new-password-hash"
	u.UpdatedAt = updatedAt

	err := repo.Update(ctx, u)

	require.NoError(t, err)

	actual, err := repo.GetByID(ctx, u.ID)

	require.NoError(t, err)
	require.Equal(t, u.ID, actual.ID)
	require.Equal(t, u.Email, actual.Email)
	require.Equal(t, u.PasswordHash, actual.PasswordHash)
	require.WithinDuration(t, u.CreatedAt, actual.CreatedAt, time.Microsecond)
	require.WithinDuration(t, u.UpdatedAt, actual.UpdatedAt, time.Microsecond)
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser()

	err := repo.Update(ctx, u)

	require.Error(t, err)
	require.True(t, errors.Is(err, user.ErrUserNotFound))
}

func TestUserRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser()

	require.NoError(t, repo.Create(ctx, u))

	err := repo.Delete(ctx, u.ID)

	require.NoError(t, err)

	_, err = repo.GetByID(ctx, u.ID)

	require.Error(t, err)
	require.True(t, errors.Is(err, user.ErrUserNotFound))
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())

	require.Error(t, err)
	require.True(t, errors.Is(err, user.ErrUserNotFound))
}
