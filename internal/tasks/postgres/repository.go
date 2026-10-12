package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/search"
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
		phase_id,
		title,
		notes,
		status,
		completed_at,
		created_by,
		assignee_id,
		starts_on,
		due_on,
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
		&task.PhaseID,
		&task.Title,
		&task.Notes,
		&task.Status,
		&task.CompletedAt,
		&task.CreatedBy,
		&task.AssigneeID,
		&task.StartsOn,
		&task.DueOn,
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
				phase_id,
				title,
				notes,
				status,
				completed_at,
				created_by,
				assignee_id,
				starts_on,
				due_on,
				"version",
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		task.ID,
		task.OrganizationID,
		task.ProjectID,
		task.TicketID,
		task.PhaseID,
		task.Title,
		task.Notes,
		task.Status,
		task.CompletedAt,
		task.CreatedBy,
		task.AssigneeID,
		task.StartsOn,
		task.DueOn,
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
					phase_id = $6,
					starts_on = $7,
					due_on = $8,
					updated_at = $9,
					"version" = tasks."version" + 1
			WHERE id = $10 AND organization_id = $11 AND "version" = $12
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
		task.PhaseID,
		task.StartsOn,
		task.DueOn,
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
	limit int,
	cursor *pagination.Cursor,
	query string,
) ([]*domain.Task, error) {
	pattern := search.LikePattern(query)

	var (
		listSQL string
		args    []any
	)

	if cursor == nil {
		listSQL = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE organization_id = $1 AND project_id = $2
				AND (
					$3 = ''
					OR title ILIKE $3 ESCAPE '\'
					OR notes ILIKE $3 ESCAPE '\'
				)
			ORDER BY created_at DESC, id DESC
			LIMIT $4
		`
		args = []any{organizationID, projectID, pattern, limit}
	} else {
		listSQL = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE organization_id = $1 AND project_id = $2
				AND (
					$3 = ''
					OR title ILIKE $3 ESCAPE '\'
					OR notes ILIKE $3 ESCAPE '\'
				)
				AND (created_at, id) < ($4, $5)
			ORDER BY created_at DESC, id DESC
			LIMIT $6
		`
		args = []any{organizationID, projectID, pattern, cursor.CreatedAt, cursor.ID, limit}
	}

	return r.list(ctx, listSQL, args...)
}

func (r *Repository) ListInbox(
	ctx context.Context,
	organizationID, userID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	query string,
) ([]*domain.Task, error) {
	pattern := search.LikePattern(query)

	var (
		listSQL string
		args    []any
	)

	if cursor == nil {
		listSQL = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE organization_id = $1
				AND (assignee_id = $2 OR assignee_id IS NULL)
				AND (
					$3 = ''
					OR title ILIKE $3 ESCAPE '\'
					OR notes ILIKE $3 ESCAPE '\'
				)
			ORDER BY created_at DESC, id DESC
			LIMIT $4
		`
		args = []any{organizationID, userID, pattern, limit}
	} else {
		listSQL = `
			SELECT ` + taskColumns + `
			FROM tasks
			WHERE organization_id = $1
				AND (assignee_id = $2 OR assignee_id IS NULL)
				AND (
					$3 = ''
					OR title ILIKE $3 ESCAPE '\'
					OR notes ILIKE $3 ESCAPE '\'
				)
				AND (created_at, id) < ($4, $5)
			ORDER BY created_at DESC, id DESC
			LIMIT $6
		`
		args = []any{organizationID, userID, pattern, cursor.CreatedAt, cursor.ID, limit}
	}

	return r.list(ctx, listSQL, args...)
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
