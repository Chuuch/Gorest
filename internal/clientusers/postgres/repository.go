package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/clientusers/domain"
	"github.com/chuuch/gorest/internal/database"
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

func (r *Repository) Create(
	ctx context.Context,
	clientuser *domain.ClientUser,
) error {
	const query = `
			INSERT INTO client_users (
				id,
				organization_id,
				client_id,
				user_id,
				created_at
		)
		VALUES ($1, $2, $3, $4, $5)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		clientuser.ID,
		clientuser.OrganizationID,
		clientuser.ClientID,
		clientuser.UserID,
		clientuser.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrClientUserAlreadyExists
		}
		return fmt.Errorf("create client user: %w", err)
	}

	return nil
}

func (r *Repository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.ClientUser, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				user_id,
				created_at
			FROM client_users
			WHERE user_id = $1
		`

	var clientUser domain.ClientUser

	err := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, userID).Scan(
		&clientUser.ID,
		&clientUser.OrganizationID,
		&clientUser.ClientID,
		&clientUser.UserID,
		&clientUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrClientUserNotFound
		}
		return nil, fmt.Errorf("get client user: %w", err)
	}

	return &clientUser, nil
}

func (r *Repository) ListByClientID(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*domain.ClientUser, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				user_id,
				created_at
			FROM client_users
			WHERE organization_id = $1 AND client_id = $2
			ORDER BY created_at ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list client users: %w", err)
	}
	defer rows.Close()

	clientUsers := make([]*domain.ClientUser, 0)

	for rows.Next() {
		var clientUser domain.ClientUser

		if err := rows.Scan(
			&clientUser.ID,
			&clientUser.OrganizationID,
			&clientUser.ClientID,
			&clientUser.UserID,
			&clientUser.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan client user: %w", err)
		}

		clientUsers = append(clientUsers, &clientUser)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list client users: %w", err)
	}

	return clientUsers, nil
}

func (r *Repository) Delete(
	ctx context.Context,
	organizationID, clientID, userID uuid.UUID,
) error {
	const query = `
			DELETE FROM client_users
			WHERE organization_id = $1 AND client_id = $2 AND user_id = $3
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		organizationID,
		clientID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete client user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrClientUserNotFound
	}

	return nil
}
