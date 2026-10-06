package postgres

import (
	"context"
	"fmt"

	"github.com/chuuch/gorest/internal/activity/domain"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/pagination"
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

func (r *Repository) Create(
	ctx context.Context,
	event *domain.Event,
) error {
	const query = `
			INSERT INTO activity_events (
				id,
				organization_id,
				actor_id,
				action,
				entity_type,
				entity_id,
				summary,
				created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		event.ID,
		event.OrganizationID,
		event.ActorID,
		event.Action,
		event.EntityType,
		event.EntityID,
		event.Summary,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create activity event: %w", err)
	}

	return nil
}

func (r *Repository) ListByOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
) ([]*domain.Event, error) {
	var (
		query string
		args  []any
	)

	if cursor == nil {
		query = `
			SELECT
				e.id,
				e.organization_id,
				e.actor_id,
				u.email,
				u.display_name,
				e.action,
				e.entity_type,
				e.entity_id,
				e.summary,
				e.created_at
			FROM activity_events e
			JOIN users u ON u.id = e.actor_id
			WHERE e.organization_id = $1
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT $2
		`
		args = []any{organizationID, limit}
	} else {
		query = `
			SELECT
				e.id,
				e.organization_id,
				e.actor_id,
				u.email,
				u.display_name,
				e.action,
				e.entity_type,
				e.entity_id,
				e.summary,
				e.created_at
			FROM activity_events e
			JOIN users u ON u.id = e.actor_id
			WHERE e.organization_id = $1
				AND (e.created_at, e.id) < ($2, $3)
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT $4
		`
		args = []any{organizationID, cursor.CreatedAt, cursor.ID, limit}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}
	defer rows.Close()

	events := make([]*domain.Event, 0)

	for rows.Next() {
		var event domain.Event

		if err := rows.Scan(
			&event.ID,
			&event.OrganizationID,
			&event.ActorID,
			&event.ActorEmail,
			&event.ActorDisplayName,
			&event.Action,
			&event.EntityType,
			&event.EntityID,
			&event.Summary,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan activity event: %w", err)
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity events: %w", err)
	}

	return events, nil
}
