package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/invoices/domain"
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

func (r *Repository) Create(ctx context.Context, invoice *domain.Invoice) error {
	const query = `
			INSERT INTO invoices (
				id,
				organization_id,
				client_id,
				number,
				status,
				currency,
				rate_cents,
				organization_name,
				client_name,
				period_from,
				period_to,
				issued_at,
				due_at,
				sent_at,
				paid_at,
				total_minutes,
				total_cents,
				created_at,
				updated_at
		)
		VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19
			)
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		invoice.ID,
		invoice.OrganizationID,
		invoice.ClientID,
		invoice.Number,
		invoice.Status,
		invoice.Currency,
		invoice.RateCents,
		invoice.OrganizationName,
		invoice.ClientName,
		invoice.PeriodFrom,
		invoice.PeriodTo,
		invoice.IssuedAt,
		invoice.DueAt,
		invoice.SentAt,
		invoice.PaidAt,
		invoice.TotalMinutes,
		invoice.TotalCents,
		invoice.CreatedAt,
		invoice.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code != "23505" {
			return domain.ErrInvoiceNumberExists
		}
		return fmt.Errorf("create invoice: %w", err)
	}
	return r.insertLines(ctx, invoice.Lines)
}

func (r *Repository) GetByID(
	ctx context.Context,
	id, organizationID uuid.UUID,
) (*domain.Invoice, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				number,
				status,
				currency,
				rate_cents,
				organization_name,
				client_name,
				period_from,
				period_to,
				issued_at,
				due_at,
				sent_at,
				paid_at,
				total_minutes,
				total_cents,
				created_at,
				updated_at
			FROM invoices
			WHERE id = $1 AND organization_id = $2
		`

	invoice, err := scanInvoice(database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		id,
		organizationID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	if err := r.attachLines(ctx, []*domain.Invoice{invoice}); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (r *Repository) ListByClientID(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*domain.Invoice, error) {
	const query = `
			SELECT
				id,
				organization_id,
				client_id,
				number,
				status,
				currency,
				rate_cents,
				organization_name,
				client_name,
				period_from,
				period_to,
				issued_at,
				due_at,
				sent_at,
				paid_at,
				total_minutes,
				total_cents,
				created_at,
				updated_at
			FROM invoices
			WHERE organization_id = $1 AND client_id = $2
			ORDER BY created_at DESC
		`

	rows, err := r.db.Query(ctx, query, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	invoices := make([]*domain.Invoice, 0)
	for rows.Next() {
		invoice, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	if err := r.attachLines(ctx, invoices); err != nil {
		return nil, err
	}

	return invoices, nil
}

func (r *Repository) Update(ctx context.Context, invoice *domain.Invoice) error {
	const query = `
			UPDATE invoices
			SET
					status = $1,
					organization_name = $2,
					client_name = $3,
					period_from = $4,
					period_to = $5,
					issued_at = $6,
					due_at = $7,
					sent_at = $8,
					paid_at = $9,
					total_minutes = $10,
					total_cents = $11,
					updated_at = $12
			WHERE id = $13 AND organization_id = $14
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		invoice.Status,
		invoice.OrganizationName,
		invoice.ClientName,
		invoice.PeriodFrom,
		invoice.PeriodTo,
		invoice.IssuedAt,
		invoice.DueAt,
		invoice.SentAt,
		invoice.PaidAt,
		invoice.TotalMinutes,
		invoice.TotalCents,
		invoice.UpdatedAt,
		invoice.ID,
		invoice.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrInvoiceNotFound
	}

	return nil
}

func (r *Repository) ReplaceLines(ctx context.Context, invoice *domain.Invoice) error {
	const query = `
			DELETE FROM invoice_line_items
			WHERE invoice_id = $1 AND organization_id = $2
		`

	_, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		invoice.ID,
		invoice.OrganizationID,
	)
	if err != nil {
		return fmt.Errorf("replace invoice lines: %w", err)
	}

	return r.insertLines(ctx, invoice.Lines)
}

func (r *Repository) Delete(ctx context.Context, id, organizationID uuid.UUID) error {
	const query = `
			DELETE FROM invoices
			WHERE id = $1 AND organization_id = $2
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(ctx, query, id, organizationID)
	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvoiceNotFound
	}

	return nil
}

func (r *Repository) NextNumber(
	ctx context.Context,
	organizationID uuid.UUID,
	year int,
) (string, error) {
	prefix := fmt.Sprintf("INV-%d-", year)
	const query = `
			SELECT COALESCE(MAX(CAST(split_part(number, '-', 3) AS INTEGER)), 0)
			FROM invoices
			WHERE organization_id = $1 AND number LIKE $2
		`

	var max int64
	if err := database.QuerierFrom(ctx, r.db).QueryRow(
		ctx,
		query,
		organizationID,
		prefix+"%",
	).Scan(&max); err != nil {
		return "", fmt.Errorf("next invoice number: %w", err)
	}

	return fmt.Sprintf("INV-%d-%04d", year, max+1), nil
}

func (r *Repository) SnapshotLines(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	from, to time.Time,
) ([]domain.SnapshotLine, error) {
	const query = `
		SELECT
			p.name,
			t.title,
			COALESCE(SUM(te.minutes), 0)
		FROM time_entries te
		JOIN tasks t ON t.id = te.task_id
		JOIN projects p ON p.id = t.project_id
		WHERE te.organization_id = $1
			AND p.client_id = $2
			AND te.created_at >= $3
			AND te.created_at < $4
		GROUP BY t.id, t.title, p.name
		ORDER BY p.name ASC, t.title ASC, t.id ASC
	`

	rows, err := r.db.Query(ctx, query, organizationID, clientID, from, to)
	if err != nil {
		return nil, fmt.Errorf("snapshot invoice lines: %w", err)
	}
	defer rows.Close()

	lines := make([]domain.SnapshotLine, 0)
	for rows.Next() {
		var line domain.SnapshotLine
		var minutes int64
		if err := rows.Scan(&line.ProjectName, &line.TaskTitle, &minutes); err != nil {
			return nil, fmt.Errorf("scan invoice snapshot: %w", err)
		}
		line.Minutes = int(minutes)
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("snapshot invoice lines: %w", err)
	}

	return lines, nil
}

func (r *Repository) insertLines(ctx context.Context, lines []domain.LineItem) error {
	const query = `
			INSERT INTO invoice_line_items (
				id,
				invoice_id,
				organization_id,
				project_name,
				task_title,
				minutes,
				amount_cents,
				position,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`

	q := database.QuerierFrom(ctx, r.db)
	for _, line := range lines {
		_, err := q.Exec(
			ctx,
			query,
			line.ID,
			line.InvoiceID,
			line.OrganizationID,
			line.ProjectName,
			line.TaskTitle,
			line.Minutes,
			line.AmountCents,
			line.Position,
			line.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create invoice line: %w", err)
		}
	}
	return nil
}

func (r *Repository) attachLines(ctx context.Context, invoices []*domain.Invoice) error {
	if len(invoices) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(invoices))
	byID := make(map[uuid.UUID]*domain.Invoice, len(invoices))
	for _, invoice := range invoices {
		invoice.Lines = make([]domain.LineItem, 0)
		ids = append(ids, invoice.ID)
		byID[invoice.ID] = invoice
	}

	const query = `
			SELECT
				id,
				invoice_id,
				organization_id,
				project_name,
				task_title,
				minutes,
				amount_cents,
				position,
				created_at
			FROM invoice_line_items
			WHERE invoice_id = ANY($1)
			ORDER BY position ASC, id ASC
		`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("line invoice lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var line domain.LineItem
		var minutes int64
		var amount int64
		if err := rows.Scan(
			&line.ID,
			&line.InvoiceID,
			&line.OrganizationID,
			&line.ProjectName,
			&line.TaskTitle,
			&minutes,
			&amount,
			&line.Position,
			&line.CreatedAt,
		); err != nil {
			return fmt.Errorf("scan invoice line: %w", err)
		}
		line.Minutes = int(minutes)
		line.AmountCents = int(amount)
		invoice, ok := byID[line.InvoiceID]
		if !ok {
			continue
		}
		invoice.Lines = append(invoice.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list invoice lines: %w", err)
	}

	return nil
}

type invoiceRow interface {
	Scan(dest ...any) error
}

func scanInvoice(row invoiceRow) (*domain.Invoice, error) {
	var invoice domain.Invoice
	var totalMinutes int64
	var totalCents int64

	if err := row.Scan(
		&invoice.ID,
		&invoice.OrganizationID,
		&invoice.ClientID,
		&invoice.Number,
		&invoice.Status,
		&invoice.Currency,
		&invoice.RateCents,
		&invoice.OrganizationName,
		&invoice.ClientName,
		&invoice.PeriodFrom,
		&invoice.PeriodTo,
		&invoice.IssuedAt,
		&invoice.DueAt,
		&invoice.SentAt,
		&invoice.PaidAt,
		&totalMinutes,
		&totalCents,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
	); err != nil {
		return nil, err
	}

	invoice.TotalMinutes = int(totalMinutes)
	invoice.TotalCents = int(totalCents)
	invoice.Lines = make([]domain.LineItem, 0)

	return &invoice, nil
}
