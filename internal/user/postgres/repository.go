package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	u *user.User,
) error {
	const query = `
			INSERT INTO users (
				id,
				email,
				password_hash,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5)
		`

	_, err := r.db.Exec(
		ctx,
		query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.CreatedAt,
		u.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*user.User, error) {
	const query = `
			SELECT
				id,
				email,
				password_hash,
				created_at,
				updated_at
			FROM users
			WHERE id = $1
		`

	var u user.User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*user.User, error) {
	const query = `
			SELECT
				id,
				email,
				password_hash,
				created_at,
				updated_at
			FROM users
			WHERE	email = $1
		`

	var u user.User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	const query = `
			SELECT EXISTS (
				SELECT 1
				FROM users
				WHERE email = $1
			)
		`

	var exists bool

	if err := r.db.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user by email: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	const query = `
		DELETE FROM users
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Update(
	ctx context.Context,
	u *user.User,
) error {
	const query = `
			UPDATE users
			SET
				email = $2,
				password_hash = $3,
				updated_at = $4
			WHERE id = $1
		`

	result, err := r.db.Exec(
		ctx,
		query,
		u.ID,
		u.Email,
		u.PasswordHash,
		u.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return user.ErrUserNotFound
	}

	return nil
}
