package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/organization/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MembershipRepository struct {
	db *pgxpool.Pool
}

func NewMembershipRepository(db *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{db: db}
}

func (r *MembershipRepository) Create(
	ctx context.Context,
	membership *domain.Membership,
) error {
	const query = `
			INSERT INTO memberships (id, organization_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`

	q := database.QuerierFrom(ctx, r.db)
	_, err := q.Exec(
		ctx,
		query,
		membership.ID,
		membership.OrganizationID,
		membership.UserID,
		membership.Role,
		membership.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}

	return nil
}

func (r *MembershipRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.Membership, error) {
	const query = `
			SELECT id, organization_id, user_id, role, created_at
			FROM memberships
			WHERE user_id = $1
		`

	var membership domain.Membership

	q := database.QuerierFrom(ctx, r.db)
	err := q.QueryRow(ctx, query, userID).Scan(
		&membership.ID,
		&membership.OrganizationID,
		&membership.UserID,
		&membership.Role,
		&membership.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}

		return nil, fmt.Errorf("get membership by user id: %w", err)
	}

	return &membership, nil
}

func (r *MembershipRepository) ListByOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]*domain.Membership, error) {
	const query = `
			SELECT id, organization_id, user_id, role, created_at
			FROM memberships
			WHERE organization_id = $1
			ORDER BY created_at ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	memberships := make([]*domain.Membership, 0)

	for rows.Next() {
		var membership domain.Membership

		if err := rows.Scan(
			&membership.ID,
			&membership.OrganizationID,
			&membership.UserID,
			&membership.Role,
			&membership.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}

		memberships = append(memberships, &membership)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list memberhips: %w", err)
	}

	return memberships, nil
}
