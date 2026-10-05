package postgres

import (
	"context"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/notifications/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, notification *domain.Notification) error {
	const query = `
			INSERT INTO notifications (
				id,
				organization_id,
				recipient_id,
				actor_id,
				kind,
				entity_type,
				entity_id,
				summary,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		notification.ID,
		notification.OrganizationID,
		notification.RecipientID,
		notification.ActorID,
		notification.Kind,
		notification.EntityType,
		notification.EntityID,
		notification.Summary,
		notification.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

func (r *Repository) ListByRecipientID(
	ctx context.Context,
	organizationID, recipientID uuid.UUID,
	limit int,
) ([]*domain.Notification, error) {
	const query = `
			SELECT
				n.id,
				n.organization_id,
				n.recipient_id,
				n.actor_id,
				u.email,
				u.display_name,
				n.kind,
				n.entity_type,
				n.entity_id,
				n.summary,
				n.read_at,
				n.created_at
			FROM notifications n
			JOIN users u ON u.id = n.actor_id
			WHERE n.organization_id = $1 AND n.recipient_id = $2
			ORDER BY n.created_at DESC
			LIMIT $3
		`

	rows, err := r.db.Query(ctx, query, organizationID, recipientID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]*domain.Notification, 0)
	for rows.Next() {
		var item domain.Notification
		if err := rows.Scan(
			&item.ID,
			&item.OrganizationID,
			&item.RecipientID,
			&item.ActorID,
			&item.ActorEmail,
			&item.ActorDisplayName,
			&item.Kind,
			&item.EntityType,
			&item.EntityID,
			&item.Summary,
			&item.ReadAt,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	return items, nil
}

func (r *Repository) MarkRead(
	ctx context.Context,
	id, organizationID, recipientID uuid.UUID,
) error {
	const query = `
			UPDATE notifications
			SET read_at = NOW()
			WHERE id = $1
				AND organization_id = $2
				AND recipient_id = $3
				AND read_at IS NULL
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		id,
		organizationID,
		recipientID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}

	return nil
}
