package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/tasks/domain"
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

const taskColumns = `
		id,
		organization_id,
		project_id,
		ticket_id,
		title,
		notes,
		status,
		completed_at,
		created_by,
		assignee_id,
		"version",
		created_at,
		updated_at
	`

func scanTask(scanner interface{ Scan(dest ...any) error }, task *domain.Task) error {
	return scanner.Scan(
		&task.ID,
		&task.OrganizationID,
		&task.ProjectID,
		&task.TicketID,
		&task.Title,
		&task.Notes,
		&task.Status,
		&task.CompletedAt,
		&task.CreatedBy,
		&task.AssigneeID,
		&task.Version,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
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
				ticket_id,
				title,
				notes,
				status,
				completed_at,
				created_by,
				assignee_id,
				"version",
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		task.ID,
		task.OrganizationID,
		task.ProjectID,
		task.TicketID,
		task.Title,
		task.Notes,
		task.Status,
		task.CompletedAt,
		task.CreatedBy,
		task.AssigneeID,
		task.Version,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "idx_tasks_project_ticket_unique" {
				return domain.ErrTicketAlreadyConverted
			}
			return domain.ErrTaskTitleExists
		}
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Task, error) {
	const query = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE id = $1 AND organization_id = $2
		`

	var task domain.Task

	err := scanTask(database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	), &task)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}

		return nil, fmt.Errorf("get task: %w", err)
	}

	return &task, nil
}

func (r *Repository) Update(
	ctx context.Context,
	task *domain.Task,
) error {
	const query = `
			UPDATE tasks
			SET
					title = $1,
					notes = $2,
					status = $3,
					completed_at = $4,
					assignee_id = $5,
					updated_at = $6,
					"version" = tasks."version" + 1
			WHERE id = $7 AND organization_id = $8 AND "version" = $9
			RETURNING "version"
		`

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		task.Title,
		task.Notes,
		task.Status,
		task.CompletedAt,
		task.AssigneeID,
		task.UpdatedAt,
		task.ID,
		task.OrganizationID,
		task.Version,
	).Scan(&task.Version)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.conflictOrNotFound(ctx, task.ID, task.OrganizationID)
		}
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	const query = `
			DELETE FROM tasks
			WHERE id = $1 AND organization_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		id,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}

	return nil
}

func (r *Repository) ListByProjectID(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*domain.Task, error) {
	const query = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE organization_id = $1 AND project_id = $2
			ORDER BY created_at ASC, title ASC
		`

	return r.list(ctx, query, organizationID, projectID)
}

func (r *Repository) ListInbox(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) ([]*domain.Task, error) {
	query := `
		SELECT ` + taskColumns + `
		FROM tasks
		WHERE organization_id = $1
			AND (assignee_id = $2 OR assignee_id IS NULL)
		ORDER BY
			CASE WHEN assignee_id = $2 THEN 0 ELSE 1 END,
			created_at ASC,
			title ASC
	`

	return r.list(ctx, query, organizationID, userID)
}

func (r *Repository) list(
	ctx context.Context,
	query string,
	args ...any,
) ([]*domain.Task, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]*domain.Task, 0)
	for rows.Next() {
		var task domain.Task

		if err := scanTask(rows, &task); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func (r *Repository) conflictOrNotFound(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	if _, err := r.GetByID(ctx, id, organizationID); err != nil {
		return err
	}

	return domain.ErrTaskVersionMismatch
}
