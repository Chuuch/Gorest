package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CountInboxActive(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) (int, error) {
	const query = `
			SELECT COUNT(*)
			FROM tasks
			WHERE organization_id = $1
				AND (assignee_id = $2 OR assignee_id IS NULL)
				AND status <> 'done'
		`

	var count int
	if err := r.db.QueryRow(ctx, query, organizationID, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count inbox tasks: %w", err)
	}
	return count, nil
}

func (r *Repository) CountActiveTickets(
	ctx context.Context,
	organizationID uuid.UUID,
) (int, error) {
	const query = `
			SELECT COUNT(*)
			FROM tickets
			WHERE organization_id = $1
				AND status IN ('open', 'in_progress')
		`

	var count int
	if err := r.db.QueryRow(ctx, query, organizationID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active tickets: %w", err)
	}
	return count, nil
}

func (r *Repository) CountUnreadNotifications(
	ctx context.Context,
	organizationID, recipientID uuid.UUID,
) (int, error) {
	const query = `
			SELECT COUNT(*)
			FROM notifications
			WHERE organization_id = $1
				AND recipient_id = $2
				AND read_at IS NULL
		`

	var count int
	if err := r.db.QueryRow(ctx, query, organizationID, recipientID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}
