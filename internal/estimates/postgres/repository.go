package postgres

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"

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
	if cursor == nil {
		listSQL = `
			SELECT
				id,
				organization_id,
				created_by_user_id,
				intake_id,
				client_id,
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
			WHERE organization_id = $1
				AND ($2::uuid IS NULL OR client_id = $2)
				AND (
					$3::timestamptz IS NULL OR (created_at, id) < ($3::timestamptz, $4::uuid)
				)
			ORDER BY created_at DESC, id DESC
			LIMIT $5
		`
		args = []any{organizationID, clientID, limit}
	} else {
		listSQL = `
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
