package postgres

import (
	"context"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/timeentries/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository{
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	entry *domain.TimeEntry,
) error {
	const query = `
			INSERT INTO time_entries (
				id,
				organization_id,
				task_id,
				user_id,
				minutes,
				notes,
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
			ctx,
			query,
			entry.ID,
			entry.OrganizationID,
			entry.TaskID,
			entry.UserID,
			entry.Minutes,
			entry.Notes,
			entry.CreatedAt,
			entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create time entry: %w", err)
	}

	return nil
}

func (r *Repository) ListByTaskID(
	ctx context.Context,
	organizationID, taskID uuid.UUID,
) ([]*domain.TimeEntry, error) {
	const query = `
			SELECT
				id,
				organization_id,
				task_id,
				user_id,
				minutes,
				notes,
				created_at,
				updated_at
			FROM time_entries
			WHERE organization_id = $1 AND task_id = $2
			ORDER BY created_at ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, taskID)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*domain.TimeEntry, 0)

	for rows.Next() {
		var entry domain.TimeEntry

		if err := rows.Scan(
			&entry.ID,
			&entry.OrganizationID,
			&entry.TaskID,
			&entry.UserID,
			&entry.Minutes,
			&entry.Notes,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan time entry: %w", err)
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}

	return entries, nil
}
