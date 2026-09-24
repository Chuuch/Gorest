package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/ticketcomments/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	comment *domain.Comment,
) error {
	const query = `
			INSERT INTO ticket_comments (
				id,
				organization_id,
				ticket_id,
				user_id,
				body,
				created_at,
				updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		comment.ID,
		comment.OrganizationID,
		comment.TicketID,
		comment.UserID,
		comment.Body,
		comment.CreatedAt,
		comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create ticket comment: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Comment, error) {
	const query = `
			SELECT
				id,
				organization_id,
				ticket_id,
				user_id,
				body,
				created_at,
				updated_at
			FROM ticket_comments
			WHERE id = $1 AND organization_id = $2
		`

	var comment domain.Comment

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	).Scan(
		&comment.ID,
		&comment.OrganizationID,
		&comment.TicketID,
		&comment.UserID,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, fmt.Errorf("get ticket comment: %w", err)
	}
	return &comment, nil
}

func (r *Repository) ListByTicketID(
	ctx context.Context,
	organizationID, ticketID uuid.UUID,
) ([]*domain.Comment, error) {
	const query = `
			SELECT
				id,
				organization_id,
				ticket_id,
				user_id,
				body,
				created_at,
				updated_at
			FROM ticket_comments
			WHERE organization_id = $1 AND ticket_id = $2
			ORDER BY created_at ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list ticket coments: %w", err)
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)

	for rows.Next() {
		var comment domain.Comment

		if err := rows.Scan(
			&comment.ID,
			&comment.OrganizationID,
			&comment.TicketID,
			&comment.UserID,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ticket comments: %w", err)
		}
		comments = append(comments, &comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list ticket comments: %w", err)
	}

	return comments, nil
}

func (r *Repository) Update(
	ctx context.Context,
	comment *domain.Comment,
) error {
	const query = `
			UPDATE ticket_comments
			SET
					body = $1,
					updated_at = $2
			WHERE id = $3 AND organization_id = $4
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		comment.Body,
		comment.UpdatedAt,
		comment.ID,
		comment.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("update ticket comment: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *Repository) Delete(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	const query = `
			DELETE FROM ticket_comments
			WHERE id = $1 AND organization_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		id,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("delete ticket comment: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}

	return nil
}
