package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/chuuch/gorest/internal/projects/phases/domain"
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

func (r *Repository) Create(ctx context.Context, phase *domain.Phase) error {
	const query = `
		INSERT INTO project_phases (
			id, organization_id, project_id, name, sort_order, status,
			starts_on, due_on, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		phase.ID,
		phase.OrganizationID,
		phase.ProjectID,
		phase.Name,
		phase.SortOrder,
		phase.Status,
		phase.StartsOn,
		phase.DueOn,
		phase.Version,
		phase.CreatedAt,
		phase.UpdatedAt,
	)
	if err != nil {
		return mapWriteError(err, "create phase")
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.Phase, error) {
	const query = `
		SELECT
			id, organization_id, project_id, name, sort_order, status,
			starts_on, due_on, version, created_at, updated_at
		FROM project_phases
		WHERE id = $1 AND organization_id = $2`

	var phase domain.Phase
	err := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, id, organizationID).Scan(
		&phase.ID,
		&phase.OrganizationID,
		&phase.ProjectID,
		&phase.Name,
		&phase.SortOrder,
		&phase.Status,
		&phase.StartsOn,
		&phase.DueOn,
		&phase.Version,
		&phase.CreatedAt,
		&phase.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPhaseNotFound
		}
		return nil, fmt.Errorf("get phase: %w", err)
	}
	return &phase, nil
}

func (r *Repository) Update(ctx context.Context, phase *domain.Phase) error {
	const query = `
		UPDATE project_phases
		SET
			name = $1,
			sort_order = $2,
			status = $3,
			starts_on = $4,
			due_on = $5,
			updated_at = $6,
			version = version + 1
		WHERE id = $7 AND organization_id = $8 AND version = $9
		RETURNING version`

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		phase.Name,
		phase.SortOrder,
		phase.Status,
		phase.StartsOn,
		phase.DueOn,
		phase.UpdatedAt,
		phase.ID,
		phase.OrganizationID,
		phase.Version,
	).Scan(&phase.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.conflictOrNotFound(ctx, phase.ID, phase.OrganizationID)
		}
		return mapWriteError(err, "update phase")
	}
	return nil
}

func (r *Repository) ListByProjectID(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*domain.Phase, error) {
	const query = `
		SELECT
			id, organization_id, project_id, name, sort_order, status,
			starts_on, due_on, version, created_at, updated_at
		FROM project_phases
		WHERE organization_id = $1 AND project_id = $2
		ORDER BY sort_order ASC, id ASC`

	rows, err := r.db.Query(ctx, query, organizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list phases: %w", err)
	}
	defer rows.Close()

	phases := make([]*domain.Phase, 0)
	for rows.Next() {
		var phase domain.Phase
		if err := rows.Scan(
			&phase.ID,
			&phase.OrganizationID,
			&phase.ProjectID,
			&phase.Name,
			&phase.SortOrder,
			&phase.Status,
			&phase.StartsOn,
			&phase.DueOn,
			&phase.Version,
			&phase.CreatedAt,
			&phase.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan phase: %w", err)
		}
		phases = append(phases, &phase)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list phases: %w", err)
	}
	return phases, nil
}

func (r *Repository) NextSortOrder(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) (int, error) {
	const query = `
		SELECT COALESCE(MAX(sort_order) + 1, 0)
		FROM project_phases
		WHERE organization_id = $1 AND project_id = $2`

	var next int
	if err := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, organizationID, projectID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next sort order: %w", err)
	}
	return next, nil
}

func (r *Repository) conflictOrNotFound(ctx context.Context, id, organizationID uuid.UUID) error {
	if _, err := r.GetByID(ctx, id, organizationID); err != nil {
		return err
	}
	return domain.ErrPhaseVersionMismatch
}

func mapWriteError(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "project_phases_project_name_unique":
			return domain.ErrPhaseNameExists
		case "project_phases_project_sort_unique":
			return domain.ErrPhaseSortExists
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
