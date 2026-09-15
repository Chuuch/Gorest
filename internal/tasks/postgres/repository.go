package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/tasks/domain"
	"github.com/google/uuid"
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
	task *domain.Task,
) error {
	const query = `
			INSERT INTO tasks (
				id,
				organization_id,
				project_id,
				title,
				notes,
				status,
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
			ctx,
			query,
			task.ID,
			task.OrganizationID,
			task.ProjectID,
			task.Title,
			task.Notes,
			task.Status,
			task.CreatedAt,
			task.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrTaskTitleExists
		}
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (r *Repository) ListByProjectID(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*domain.Task, error) {
	const query = `
			SELECT
				id,
				organization_id,
				project_id,
				title,
				notes,
				status,
				created_at,
				updated_at,
			FROM tasks
			WHERE organization_id = $1 AND project_id = $2
			ORDER BY created_at ASC, title ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*domain.Task, 0)

	for rows.Next() {
		var task domain.Task

		if err := rows.Scan(
			&task.ID,
			&task.OrganizationID,
			&task.ProjectID,
			&task.Title,
			&task.Notes,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}
