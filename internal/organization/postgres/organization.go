package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	db *pgxpool.Pool
}

func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{
		db: db,
	}
}

func (r *OrganizationRepository) Create(
	ctx context.Context,
	org *domain.Organization,
) error {
	rate := org.DefaultVATRateBPS
	if rate == 0 {
		rate = 2000
	}

	const query = `
			INSERT INTO organizations (
				id,
				name,
				legal_name,
				registration_number,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
				default_vat_rate_bps,
				bank_iban,
				bank_bic,
				bank_name,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16
			)
		`

	q := database.QuerierFrom(ctx, r.db)
	_, err := q.Exec(
		ctx,
		query,
		org.ID,
		org.Name,
		org.LegalName,
		org.RegistrationNumber,
		org.VATID,
		org.AddressLine1,
		org.AddressLine2,
		org.City,
		org.PostalCode,
		org.Country,
		rate,
		org.BankIBAN,
		org.BankBIC,
		org.BankName,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	return nil
}

func (r *OrganizationRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Organization, error) {
	const query = `
			SELECT
				id,
				name,
				legal_name,
				registration_number,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
				default_vat_rate_bps,
				bank_iban,
				bank_bic,
				bank_name,
				created_at,
				updated_at
			FROM organizations
			WHERE id = $1
		`

	var org domain.Organization

	q := database.QuerierFrom(ctx, r.db)
	err := q.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&org.ID,
		&org.Name,
		&org.LegalName,
		&org.RegistrationNumber,
		&org.VATID,
		&org.AddressLine1,
		&org.AddressLine2,
		&org.City,
		&org.PostalCode,
		&org.Country,
		&org.DefaultVATRateBPS,
		&org.BankIBAN,
		&org.BankBIC,
		&org.BankName,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}

		return nil, fmt.Errorf("get organization by id: %w", err)
	}

	return &org, nil
}

func (r *OrganizationRepository) Update(
	ctx context.Context,
	org *domain.Organization,
) error {
	const query = `
			UPDATE organizations
			SET 
				name = $2,
				legal_name = $3,
				registration_number = $4,
				vat_id = $5,
				address_line1 = $6,
				address_line2 = $7,
				city = $8,
				postal_code = $9,
				country = $10,
				default_vat_rate_bps = $11,
				bank_iban = $12,
				bank_bic = $13,
				bank_name = $14,
				updated_at = $15
			WHERE id = $1
		`

	q := database.QuerierFrom(ctx, r.db)
	tag, err := q.Exec(
		ctx,
		query,
		org.ID,
		org.Name,
		org.LegalName,
		org.RegistrationNumber,
		org.VATID,
		org.AddressLine1,
		org.AddressLine2,
		org.City,
		org.PostalCode,
		org.Country,
		org.DefaultVATRateBPS,
		org.BankIBAN,
		org.BankBIC,
		org.BankName,
		org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrOrganizationNotFound
	}

	return nil
}
