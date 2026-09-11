package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/organization/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	db *pgxpool.Pool
}

func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{
		db: db,
	}
}

func (r *OrganizationRepository) Create(
	ctx context.Context,
	org *domain.Organization,
) error {
	const query = `
			INSERT INTO organizations (id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`

	q := database.QuerierFrom(ctx, r.db)
	_, err := q.Exec(
		ctx,
		query,
		org.ID,
		org.Name,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	return nil
}

func (r *OrganizationRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Organization, error) {
	const query = `
			SELECT id, name, created_at, updated_at
			FROM organizations
			WHERE id = $1
		`

	var org domain.Organization

	q := database.QuerierFrom(ctx, r.db)
	err := q.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&org.ID,
		&org.Name,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}

		return nil, fmt.Errorf("get organization by id: %w", err)
	}

	return &org, nil
}
