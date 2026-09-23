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

func (r *MembershipRepository) CountOwners(
	ctx context.Context,
	organizationID uuid.UUID,
) (int, error) {
	const query = `
			SELECT COUNT(*)
			FROM memberships
			WHERE organization_id = $1 AND role = $2
		`

	var count int

	err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		organizationID,
		domain.RoleOwner,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}

	return count, nil
}

func (r *MembershipRepository) UpdateRole(
	ctx context.Context,
	organizationID, userID uuid.UUID,
	role domain.Role,
) error {
	const query = `
			UPDATE memberships
			SET role = $1
			WHERE organization_id = $2 AND user_id = $3
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		role,
		organizationID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("update membership role: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}

	return nil
}

func (r *MembershipRepository) Delete(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) error {
	const query = `
			DELETE FROM memberships
			WHERE organization_id = $1 AND user_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		organizationID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}

	return nil
}
