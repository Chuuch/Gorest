package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/client/domain"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/search"
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
				legal_name,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		client.ID,
		client.OrganizationID,
		client.Name,
		client.Notes,
		client.LegalName,
		client.VATID,
		client.AddressLine1,
		client.AddressLine2,
		client.City,
		client.PostalCode,
		client.Country,
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
				legal_name,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
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
		&client.Notes,
		&client.LegalName,
		&client.VATID,
		&client.AddressLine1,
		&client.AddressLine2,
		&client.City,
		&client.PostalCode,
		&client.Country,
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

func (r *Repository) Update(
	ctx context.Context,
	client *domain.Client,
) error {
	const query = `
			UPDATE clients
			SET
					name = $1,
					notes = $2,
					legal_name = $3,
					vat_id = $4,
					address_line1 = $5,
					address_line2 = $6,
					city = $7,
					postal_code = $8,
					country = $9,
					updated_at = $10
			WHERE id = $11 AND organization_id = $12
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		client.Name,
		client.Notes,
		client.LegalName,
		client.VATID,
		client.AddressLine1,
		client.AddressLine2,
		client.City,
		client.PostalCode,
		client.Country,
		client.UpdatedAt,
		client.ID,
		client.OrganizationID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrClientNameExists
		}

		return fmt.Errorf("update client: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrClientNotFound
	}
	return nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	const query = `
			DELETE FROM clients
			WHERE id = $1 AND organization_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		id,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("delete client: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrClientNotFound
	}
	return nil
}

func (r *Repository) ListByOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
	query string,
) ([]*domain.Client, error) {
	const listSQL = `
			SELECT
				id,
				organization_id,
				name,
				notes,
				legal_name,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
				created_at,
				updated_at
			FROM clients
			WHERE organization_id = $1
				AND (
					$2 = ''
					OR name ILIKE $2 ESCAPE '\'
					OR notes ILIKE $2 ESCAPE '\'
				)
			ORDER BY name ASC
		`

	pattern := search.LikePattern(query)

	rows, err := r.db.Query(ctx, listSQL, organizationID, pattern)
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
			&client.LegalName,
			&client.VATID,
			&client.AddressLine1,
			&client.AddressLine2,
			&client.City,
			&client.PostalCode,
			&client.Country,
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
