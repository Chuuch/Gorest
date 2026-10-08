package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) TryClaim(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
	method string,
	path string,
	requestHash string,
	expiresAt time.Time,
) (*Record, error) {
	const insertQuery = `
		INSERT INTO idempotency_keys (
			organization_id,
			key,
			method,
			path,
			request_hash,
			created_at,
			expires_at
		) VALUES ($1, $2, $3, $4, $5, now(), $6)
		ON CONFLICT (organization_id, key) DO NOTHING
	`

	tag, err := s.db.Exec(
		ctx,
		insertQuery,
		organizationID,
		key,
		method,
		path,
		requestHash,
		expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("claim idempotency key: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil, nil
	}

	rec, err := s.get(ctx, organizationID, key)
	if err != nil {
		return nil, err
	}

	if time.Now().UTC().After(rec.ExpiresAt) {
		if err := s.Release(ctx, organizationID, key); err != nil {
			return nil, err
		}
		return s.TryClaim(ctx, organizationID, key, method, path, requestHash, expiresAt)
	}

	if rec.RequestHash != requestHash || rec.Method != method || rec.Path != path {
		return nil, ErrKeyMismatch
	}

	if !rec.Complete() {
		return nil, ErrKeyInProgress
	}

	return rec, nil
}

func (s *PostgresStore) Complete(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
	statusCode int,
	responseBody []byte,
) error {
	const query = `
		UPDATE idempotency_keys
		SET status_code = $3,
		    response_body = $4
		WHERE organization_id = $1
		  AND key = $2
	`

	if responseBody == nil {
		responseBody = []byte{}
	}

	tag, err := s.db.Exec(ctx, query, organizationID, key, statusCode, responseBody)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("complete idempotency key: not found")
	}
	return nil
}

func (s *PostgresStore) Release(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
) error {
	const query = `
		DELETE FROM idempotency_keys
		WHERE organization_id = $1
		  AND key = $2
	`

	_, err := s.db.Exec(ctx, query, organizationID, key)
	if err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}

func (s *PostgresStore) get(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
) (*Record, error) {
	const query = `
		SELECT
			organization_id,
			key,
			method,
			path,
			request_hash,
			status_code,
			response_body,
			created_at,
			expires_at
		FROM idempotency_keys
		WHERE organization_id = $1
		  AND key = $2
	`

	var rec Record
	var statusCode *int
	var body []byte

	err := s.db.QueryRow(ctx, query, organizationID, key).Scan(
		&rec.OrganizationID,
		&rec.Key,
		&rec.Method,
		&rec.Path,
		&rec.RequestHash,
		&statusCode,
		&body,
		&rec.CreatedAt,
		&rec.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("idempotency key missing after conflict: %w", err)
		}
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}

	rec.StatusCode = statusCode
	rec.ResponseBody = body
	return &rec, nil
}
