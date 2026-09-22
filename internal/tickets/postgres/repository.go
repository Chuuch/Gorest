package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/tickets/domain"
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
	ticket *domain.Ticket,
) error {
	const query = `
			INSERT INTO tickets (
				id,
				organization_id,
				client_id,
				user_id,
				kind,
				status,
				title,
				body,
				version,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		ticket.ID,
		ticket.OrganizationID,
		ticket.ClientID,
		ticket.UserID,
		ticket.Kind,
		ticket.Status,
		ticket.Title,
		ticket.Body,
		ticket.Version,
		ticket.CreatedAt,
		ticket.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Ticket, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				user_id,
				kind,
				status,
				title,
				body,
				version,
				created_at,
				updated_at
			FROM tickets
			WHERE id = $1 AND organization_id = $2
		`
	var ticket domain.Ticket

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	).Scan(
		&ticket.ID,
		&ticket.OrganizationID,
		&ticket.ClientID,
		&ticket.UserID,
		&ticket.Kind,
		&ticket.Status,
		&ticket.Title,
		&ticket.Body,
		&ticket.Version,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	return &ticket, nil
}

func (r *Repository) Update(
	ctx context.Context,
	ticket *domain.Ticket,
) error {
	const query = `
			UPDATE tickets
			SET
					status = $1,
					updated_at = $2
					version = version + 1
			WHERE id = $3 AND organization_id = $4 AND version = $5
		RETURNING version
		`

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		ticket.Status,
		ticket.UpdatedAt,
		ticket.ID,
		ticket.OrganizationID,
		ticket.Version,
	).Scan(&ticket.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.conflictOrNotFound(ctx, ticket.ID, ticket.OrganizationID)
		}
		return fmt.Errorf("update ticket: %w", err)
	}
	return nil
}

func (r *Repository) ListByClientID(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*domain.Ticket, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				user_id,
				kind,
				status,
				title,
				body,
				version,
				created_at,
				updated_at
			FROM tickets
			WHERE organization_id = $1 AND client_id = $2
			ORDER BY created_at DESC
		`

	rows, err := r.db.Query(ctx, query, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()

	tickets := make([]*domain.Ticket, 0)

	for rows.Next() {
		var ticket domain.Ticket

		if err := rows.Scan(
			&ticket.ID,
			&ticket.OrganizationID,
			&ticket.ClientID,
			&ticket.UserID,
			&ticket.Kind,
			&ticket.Status,
			&ticket.Title,
			&ticket.Body,
			&ticket.Version,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}

		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return tickets, nil
}

func (r *Repository) conflictOrNotFound(
	ctx context.Context,
	id, organizationID uuid.UUID,
) error {
	if _, err := r.GetByID(ctx, id, organizationID); err != nil {
		return err
	}
	return domain.ErrTicketVersionMismatch
}
