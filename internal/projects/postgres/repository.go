package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/projects/domain"
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
	project *domain.Project,
) error {
	const query = `
			INSERT INTO projects (
				id,
				organization_id,
				client_id,
				name,
				notes,
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		project.ID,
		project.OrganizationID,
		project.ClientID,
		project.Name,
		project.Notes,
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrProjectNameExists
		}

		return fmt.Errorf("create project: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Project, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				name,
				notes,
				created_at,
				updated_at
			FROM projects
			WHERE id = $1 AND organization_id = $2
		`

	var project domain.Project

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	).Scan(
		&project.ID,
		&project.OrganizationID,
		&project.ClientID,
		&project.Name,
		&project.Notes,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}

		return nil, fmt.Errorf("get project: %w", err)
	}

	return &project, nil
}

func (r *Repository) ListByClientID(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*domain.Project, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				name,
				notes,
				created_at,
				updated_at
			FROM
				projects
			WHERE
				organization_id = $1 AND client_id = $2
			ORDER BY
				name ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]*domain.Project, 0)

	for rows.Next() {
		var project domain.Project

		if err := rows.Scan(
			&project.ID,
			&project.OrganizationID,
			&project.ClientID,
			&project.Name,
			&project.Notes,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}

		projects = append(projects, &project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}
