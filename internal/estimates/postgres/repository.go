package postgres

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/chuuch/gorest/internal/estimates/domain"
	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/chuuch/gorest/internal/platform/pagination"
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

func (r *Repository) Create(ctx context.Context, run *domain.EstimateRun) error {
	inputJSON, err := json.Marshal(run.Input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}
	resultJSON, err := json.Marshal(run.Result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	const query = `
			INSERT INTO estimate_runs (
				id, organization_id, created_by_user_id, intake_id, client_id,
				category, mode, catalog_version_id, currency,
				input_json, result_json,
				estimated_hours, estimated_timeline_days, hours_per_month,
				minimum_price_cents, recommended_price_cents, risk_level, created_at
			) VALUES (
					$1, $2, $3, $4, $5,
					$6, $7, $8, $9,
					$10, $11,
					$12, $13, $14,
					$15, $16, $17, $18
			)`

	_, err = database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		run.ID,
		run.OrganizationID,
		run.CreatedByUserID,
		run.IntakeID,
		run.ClientID,
		run.Category,
		run.Mode,
		run.CatalogVersionID,
		run.Currency,
		inputJSON,
		resultJSON,
		run.EstimatedHours,
		run.EstimatedTimelineDays,
		run.HoursPerMonth,
		run.MinimumPriceCents,
		run.RecommendedPriceCents,
		run.RiskLevel,
		run.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create estimate run: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.EstimateRun, error) {
	const query = `
			SELECT
				id,
				organization_id,
				created_by_user_id,
				intake_id,
				client_id,
				category,
				mode,
				catalog_version_id,
				currency,
				input_json,
				result_json,
				estimated_hours,
				estimated_timeline_days,
				hours_per_month,
				minimum_price_cents,
				recommended_price_cents,
				risk_level,
				created_at
			FROM estimate_runs
			WHERE id = $1 AND organization_id = $2
		`
	return r.scanOne(ctx, query, id, organizationID)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	clientID *uuid.UUID,
) ([]*domain.EstimateRun, error) {
	var (
		listSQL string
		args    []any
	)

	const selectCols = `
			id,
			organization_id,
			created_by_user_id,
			intake_id,
			client_id,
			category,
			mode,
			catalog_version_id,
			currency,
			input_json,
			result_json,
			estimated_hours,
			estimated_timeline_days,
			hours_per_month,
			minimum_price_cents,
			recommended_price_cents,
			risk_level,
			created_at
		`

	if cursor == nil {
		listSQL = `
			SELECT` + selectCols + `
			FROM estimate_runs
			WHERE organization_id = $1
				AND ($2::uuid IS NULL OR client_id = $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`
		args = []any{organizationID, clientID, limit}
	} else {
		listSQL = `
			SELECT` + selectCols + `
			FROM estimate_runs
			WHERE organization_id = $1
				AND ($2::uuid IS NULL OR client_id = $2)
				AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $5
		`

		args = []any{organizationID, clientID, cursor.CreatedAt, cursor.ID, limit}
	}

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list estimate runs: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.EstimateRun, 0)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan estimate run: %w", err)
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list estimate runs: %w", err)
	}
	return out, nil
}

func (r *Repository) CreateIntake(ctx context.Context, intake *domain.EstimateIntake) error {
	scopeJSON, err := marshalScope(intake.Scope)
	if err != nil {
		return err
	}

	const query = `
			INSERT INTO estimate_intakes (
				id, organization_id, created_by_user_id, token_hash,
			category, mode, status, expires_at,
			respondent_email, respondent_name, company_name, project_name,
			preferred_deadline, client_id, scope_json, submitted_at, created_at
			) VALUES (
				$1, $2, $3, $4,
		 		$5, $6, $7, $8,
				$9, $10, $11, $12,
				$13, $14, $15, $16, $17
			)`

	_, err = database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		intake.ID,
		intake.OrganizationID,
		intake.CreatedByUserID,
		intake.TokenHash,
		intake.Category,
		intake.Mode,
		intake.Status,
		intake.ExpiresAt,
		intake.RespondentEmail,
		intake.RespondentName,
		intake.CompanyName,
		intake.ProjectName,
		intake.PreferredDeadline,
		intake.ClientID,
		scopeJSON,
		intake.SubmittedAt,
		intake.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create estimate intake: %w", err)
	}
	return nil
}

func (r *Repository) GetIntakeByID(ctx context.Context, id, organizationID uuid.UUID) (*domain.EstimateIntake, error) {
	const query = `
			SELECT
				id, organization_id, created_by_user_id, token_hash,
				category, mode, status, expires_at,
				respondent_email, respondent_name, company_name, project_name,
				preferred_deadline, client_id, scope_json, submitted_at, created_at
			FROM estimate_intakes
			WHERE id = $1 AND organization_id = $2
		`

	row := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, id, organizationID)
	intake, err := scanIntake(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrIntakeNotFound
		}
		return nil, err
	}
	return intake, nil
}

func (r *Repository) GetIntakeByTokenHash(ctx context.Context, tokenHash string) (*domain.EstimateIntake, error) {
	const query = `
			SELECT
				id, organization_id, created_by_user_id, token_hash,
				category, mode, status, expires_at,
				respondent_email, respondent_name, company_name, project_name,
				preferred_deadline, client_id, scope_json, submitted_at, created_at
			FROM estimate_intakes
			WHERE token_hash = $1`

	row := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, tokenHash)
	intake, err := scanIntake(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrIntakeNotFound
		}
		return nil, err
	}
	return intake, nil
}

func (r *Repository) ListIntakes(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	status string,
) ([]*domain.EstimateIntake, error) {
	status = strings.TrimSpace(status)

	var (
		listSQL string
		args    []any
	)

	if cursor == nil {
		listSQL = `
			SELECT
				id, organization_id, created_by_user_id, token_hash,
				category, mode, status, expires_at,
				respondent_email, respondent_name, company_name, project_name,
				preferred_deadline, client_id, scope_json, submitted_at, created_at
			FROM estimate_intakes
			WHERE organization_id = $1
				AND ($2 = '' OR status = $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3`

		args = []any{organizationID, status, limit}
	} else {
		listSQL = `
			SELECT
				id, organization_id, created_by_user_id, token_hash,
				category, mode, status, expires_at,
				respondent_email, respondent_name, company_name, project_name,
				preferred_deadline, client_id, scope_json, submitted_at, created_at
			FROM estimate_intakes
			WHERE organization_id = $1
				AND ($2 = '' OR status = $2)
				AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $5`
		args = []any{organizationID, status, cursor.CreatedAt, cursor.ID, limit}
	}

	rows, err := r.db.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list estimate intakes: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.EstimateIntake, 0)
	for rows.Next() {
		intake, err := scanIntake(rows)
		if err != nil {
			return nil, fmt.Errorf("scan estimate intake: %w", err)
		}
		out = append(out, intake)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list estimate intakes: %w", err)
	}
	return out, nil
}

func (r *Repository) UpdateIntake(ctx context.Context, intake *domain.EstimateIntake) error {
	scopeJSON, err := marshalScope(intake.Scope)
	if err != nil {
		return err
	}

	const query = `
		UPDATE estimate_intakes SET
			status = $1,
			respondent_email = $2,
			respondent_name = $3,
			company_name = $4,
			project_name = $5,
			preferred_deadline = $6,
			client_id = $7,
			scope_json = $8,
			submitted_at = $9
		WHERE id = $10 AND organization_id = $11`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		intake.Status,
		intake.RespondentEmail,
		intake.RespondentName,
		intake.CompanyName,
		intake.ProjectName,
		intake.PreferredDeadline,
		intake.ClientID,
		scopeJSON,
		intake.SubmittedAt,
		intake.ID,
		intake.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("update estimate intake: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEstimateNotFound
	}
	return nil
}

func marshalScope(scope *domain.IntakeScope) ([]byte, error) {
	if scope == nil {
		return nil, nil
	}
	b, err := json.Marshal(scope)
	if err != nil {
		return nil, fmt.Errorf("marshal scope: %w", err)
	}
	return b, nil
}

func scanIntake(row rowScanner) (*domain.EstimateIntake, error) {
	var (
		intake    domain.EstimateIntake
		scopeJSON []byte
	)

	err := row.Scan(
		&intake.ID,
		&intake.OrganizationID,
		&intake.CreatedByUserID,
		&intake.TokenHash,
		&intake.Category,
		&intake.Mode,
		&intake.Status,
		&intake.ExpiresAt,
		&intake.RespondentEmail,
		&intake.RespondentName,
		&intake.CompanyName,
		&intake.ProjectName,
		&intake.PreferredDeadline,
		&intake.ClientID,
		&scopeJSON,
		&intake.SubmittedAt,
		&intake.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(scopeJSON) > 0 {
		var scope domain.IntakeScope
		if err := json.Unmarshal(scopeJSON, &scope); err != nil {
			return nil, fmt.Errorf("unmarshal scope: %w", err)
		}
		intake.Scope = &scope
	}
	return &intake, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (*domain.EstimateRun, error) {
	row := database.QuerierFrom(ctx, r.db).QueryRow(ctx, query, args...)
	run, err := scanRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEstimateNotFound
		}
		return nil, err
	}
	return run, nil
}

func scanRun(row rowScanner) (*domain.EstimateRun, error) {
	var (
		run        domain.EstimateRun
		inputJSON  []byte
		resultJSON []byte
	)

	err := row.Scan(
		&run.ID,
		&run.OrganizationID,
		&run.CreatedByUserID,
		&run.IntakeID,
		&run.ClientID,
		&run.Category,
		&run.Mode,
		&run.CatalogVersionID,
		&run.Currency,
		&inputJSON,
		&resultJSON,
		&run.EstimatedHours,
		&run.EstimatedTimelineDays,
		&run.HoursPerMonth,
		&run.MinimumPriceCents,
		&run.RecommendedPriceCents,
		&run.RiskLevel,
		&run.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(inputJSON, &run.Input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}
	if err := json.Unmarshal(resultJSON, &run.Result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}
	return &run, nil
}
