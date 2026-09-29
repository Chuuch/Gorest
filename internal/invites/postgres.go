package invites

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, token *Token) error {
	const query = `
			INSERT INTO auth_tokens (
				id,
				user_id,
				token_hash,
				purpose,
				expires_at,
				created_at,
				used_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.Purpose,
		token.ExpiresAt,
		token.CreatedAt,
		token.UsedAt,
	)
	if err != nil {
		return fmt.Errorf("create auth token: %w", err)
	}
	return nil
}

func (r *Repository) GetByHash(
	ctx context.Context,
	tokenHash string,
) (*Token, error) {
	const query = `
			SELECT
				id,
				user_id,
				token_hash,
				purpose,
				expires_at,
				created_at,
				used_at
			FROM auth_tokens
			WHERE token_hash = $1
		`

	var token Token

	err := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.Purpose,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInviteNotFound
		}

		return nil, fmt.Errorf("get auth token by hash: %w", err)
	}
	return &token, nil
}

func (r *Repository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	const query = `
			UPDATE auth_tokens
			SET used_at = NOW()
			WHERE id = $1 
			AND used_at IS NULL
		`

	result, err := database.QuerierFrom(ctx, r.db).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark auth token used: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrInviteUsed
	}
	return nil
}
