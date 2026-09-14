package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/client/domain"
	"github.com/chuuch/gorest/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	client *domain.Client,
) error {
	const query = `
			INSERT INTO clients (
				id,
				organization_id,
				name,
				notes,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		client.ID,
		client.OrganizationID,
		client.Name,
		client.Notes,
		client.CreatedAt,
		client.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrClientNameExists
		}

		return fmt.Errorf("create client: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Client, error) {
	const query = `
			SELECT
				id,
				organization_id,
				name,
				notes,
				created_at,
				updated_at
			FROM clients
			WHERE id = $1 AND organization_id = $2
		`

	var client domain.Client

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	).Scan(
		&client.ID,
		&client.OrganizationID,
		&client.Name,
		&client.Name,
		&client.Notes,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClientNotFound
		}

		return nil, fmt.Errorf("get client: %w", err)
	}

	return &client, nil
}

func (r *Repository) ListByOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]*domain.Client, error) {
	const query = `
			SELECT
				id,
				organization_id,
				name,
				notes,
				created_at,
				updated_at
			FROM clients
			WHERE organization_id = $1
			ORDER BY name ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	clients := make([]*domain.Client, 0)

	for rows.Next() {
		var client domain.Client

		if err := rows.Scan(
			&client.ID,
			&client.OrganizationID,
			&client.Name,
			&client.Notes,
			&client.CreatedAt,
			&client.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}

		clients = append(clients, &client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}

	return clients, nil
}
