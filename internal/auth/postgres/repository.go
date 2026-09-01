package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
	}
}

func (r *RefreshTokenRepository) Create(
	ctx context.Context,
	token *auth.RefreshToken,
) error {
	const query = `
			INSERT INTO refresh_tokens (
				id,
				user_id,
				token_hash,
				expires_at,
				created_at,
				revoked_at
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`

	_, err := r.db.Exec(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
		token.RevokedAt,
	)

	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetByHash(
	ctx context.Context,
	tokenHash string,
) (*auth.RefreshToken, error) {
	const query = `
			SELECT
				id,
				user_id,
				token_hash,
				expires_at,
				created_at,
				revoked_at
			FROM refresh_tokens
			WHERE token_hash = $1
		`

	var token auth.RefreshToken

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, auth.ErrInvalidToken
		}

		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return &token, nil
}

func (r *RefreshTokenRepository) Revoke(
	ctx context.Context,
	id uuid.UUID,
) error {
	const query = `
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE id = $1
				AND revoked_at IS NULL
		`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return auth.ErrTokenRevoked
	}

	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(
	ctx context.Context,
	userID uuid.UUID,
) error {
	const query = `
			UPDATE refresh_tokens
			SET revoked_at = NOW()
			WHERE user_id = $1
				AND revoked_at IS NULL
		`

	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("revoke all refresh tokens for user: %w", err)
	}

	return nil
}
