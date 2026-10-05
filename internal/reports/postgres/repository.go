package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/reports/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) TimeReport(
	ctx context.Context,
	organizationID uuid.UUID,
	userID *uuid.UUID,
	from, to time.Time,
) (*domain.TimeReport, error) {
	report := &domain.TimeReport{
		From:      from,
		To:        to,
		ByClient:  make([]domain.ClientRow, 0),
		ByProject: make([]domain.ProjectRow, 0),
		ByMember:  make([]domain.MemberRow, 0),
	}

	total, err := r.sumTotal(ctx, organizationID, userID, from, to)
	if err != nil {
		return nil, err
	}
	report.TotalMinutes = total

	clients, err := r.sumByClient(ctx, organizationID, userID, from, to)
	if err != nil {
		return nil, err
	}
	report.ByClient = clients

	projects, err := r.sumByProject(ctx, organizationID, userID, from, to)
	if err != nil {
		return nil, err
	}
	report.ByProject = projects

	members, err := r.sumByMember(ctx, organizationID, userID, from, to)
	if err != nil {
		return nil, err
	}
	report.ByMember = members

	return report, nil
}

func (r *Repository) sumTotal(
	ctx context.Context,
	organizationID uuid.UUID,
	userID *uuid.UUID,
	from, to time.Time,
) (int, error) {
	const query = `
			SELECT COALESCE(SUM(minutes), 0)
			FROM time_entries
			WHERE organization_id = $1
				AND created_at >= $2
				AND created_at < $3
				AND ($4::uuid IS NULL OR user_id = $4)
		`

	var total int64
	if err := r.db.QueryRow(ctx, query, organizationID, from, to, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum time report: %w", err)
	}

	return int(total), nil
}

func (r *Repository) sumByClient(
	ctx context.Context,
	organizationID uuid.UUID,
	userID *uuid.UUID,
	from, to time.Time,
) ([]domain.ClientRow, error) {
	const query = `
			SELECT
				c.id,
				c.name,
				COALESCE(SUM(te.minutes), 0)
			FROM time_entries te
			JOIN tasks t ON t.id = te.task_id
			JOIN projects p ON p.id = t.project_id
			JOIN clients c ON c.id = p.client_id
			WHERE te.organization_id = $1
				AND te.created_at >= $2
				AND te.created_at < $3
				AND ($4::uuid IS NULL OR te.user_id = $4)
			GROUP BY c.id, c.name
			ORDER BY c.name ASC, c.id ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, from, to, userID)
	if err != nil {
		return nil, fmt.Errorf("sum time report by client: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ClientRow, 0)
	for rows.Next() {
		var row domain.ClientRow
		var minutes int64
		if err := rows.Scan(&row.ClientID, &row.ClientName, &minutes); err != nil {
			return nil, fmt.Errorf("scan time report client: %w", err)
		}
		row.Minutes = int(minutes)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sum time report by client: %w", err)
	}

	return items, nil
}

func (r *Repository) sumByProject(
	ctx context.Context,
	organizationID uuid.UUID,
	userID *uuid.UUID,
	from, to time.Time,
) ([]domain.ProjectRow, error) {
	const query = `
			SELECT
				p.id,
				p.name,
				c.id,
				c.name,
				COALESCE(SUM(te.minutes), 0)
			FROM time_entries te
			JOIN tasks t ON t.id = te.task_id
			JOIN projects p ON p.id = t.project_id
			JOIN clients c ON c.id = p.client_id
			WHERE te.organization_id = $1
				AND te.created_at >= $2
				AND te.created_at < $3
				AND ($4::uuid IS NULL OR te.user_id = $4)
			GROUP BY p.id, p.name, c.id, c.name
			ORDER BY p.name ASC, p.id ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, from, to, userID)
	if err != nil {
		return nil, fmt.Errorf("sum time report by project: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ProjectRow, 0)
	for rows.Next() {
		var row domain.ProjectRow
		var minutes int64
		if err := rows.Scan(
			&row.ProjectID,
			&row.ProjectName,
			&row.ClientID,
			&row.ClientName,
			&minutes,
		); err != nil {
			return nil, fmt.Errorf("scan time report project: %w", err)
		}
		row.Minutes = int(minutes)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sum time report by project: %w", err)
	}

	return items, nil
}

func (r *Repository) sumByMember(
	ctx context.Context,
	organizationID uuid.UUID,
	userID *uuid.UUID,
	from, to time.Time,
) ([]domain.MemberRow, error) {
	const query = `
			SELECT
				u.id,
				u.email,
				u.display_name,
				COALESCE(SUM(te.minutes), 0)
			FROM time_entries te
			JOIN users u ON u.id = te.user_id
			WHERE te.organization_id = $1
				AND te.created_at >= $2
				AND te.created_at < $3
				AND ($4::uuid IS NULL OR te.user_id = $4)
			GROUP BY u.id, u.email, u.display_name
			ORDER BY u.email ASC, u.id ASC
		`

	rows, err := r.db.Query(ctx, query, organizationID, from, to, userID)
	if err != nil {
		return nil, fmt.Errorf("sum time report by member: %w", err)
	}
	defer rows.Close()

	items := make([]domain.MemberRow, 0)
	for rows.Next() {
		var row domain.MemberRow
		var minutes int64
		if err := rows.Scan(&row.UserID, &row.Email, &row.DisplayName, &minutes); err != nil {
			return nil, fmt.Errorf("scan time report member: %w", err)
		}
		row.Minutes = int(minutes)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sum time report by member: %w", err)
	}

	return items, nil
}
