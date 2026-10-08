package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/invoices/domain"
	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/search"
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
				seller_legal_name,
				seller_registration_number,
				seller_vat_id,
				seller_address_line1,
				seller_address_line2,
				seller_city,
				seller_postal_code,
				seller_country,
				buyer_legal_name,
				buyer_vat_id,
				buyer_address_line1,
				buyer_address_line2,
				buyer_city,
				buyer_postal_code,
				buyer_country,
				vat_regime,
				vat_rate_bps,
				subtotal_cents,
				vat_cents,
				bank_iban,
				bank_bic,
				bank_name,
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
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
				$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
				$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
				$41
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
		invoice.SellerLegalName,
		invoice.SellerRegistrationNumber,
		invoice.SellerVATID,
		invoice.SellerAddressLine1,
		invoice.SellerAddressLine2,
		invoice.SellerCity,
		invoice.SellerPostalCode,
		invoice.SellerCountry,
		invoice.BuyerLegalName,
		invoice.BuyerVATID,
		invoice.BuyerAddressLine1,
		invoice.BuyerAddressLine2,
		invoice.BuyerCity,
		invoice.BuyerPostalCode,
		invoice.BuyerCountry,
		invoice.VATRegime,
		invoice.VATRateBPS,
		invoice.SubtotalCents,
		invoice.VATCents,
		invoice.BankIBAN,
		invoice.BankBIC,
		invoice.BankName,
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
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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
				seller_legal_name,
				seller_registration_number,
				seller_vat_id,
				seller_address_line1,
				seller_address_line2,
				seller_city,
				seller_postal_code,
				seller_country,
				buyer_legal_name,
				buyer_vat_id,
				buyer_address_line1,
				buyer_address_line2,
				buyer_city,
				buyer_postal_code,
				buyer_country,
				vat_regime,
				vat_rate_bps,
				subtotal_cents,
				vat_cents,
				bank_iban,
				bank_bic,
				bank_name,
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
	limit int,
	cursor *pagination.Cursor,
	query string,
	statuses []string,
) ([]*domain.Invoice, error) {
	pattern := search.LikePattern(query)
	args := []any{organizationID, clientID, pattern}

	statusClause := ""
	if len(statuses) > 0 {
		args = append(args, statuses)
		statusClause = fmt.Sprintf(" AND status = ANY($%d)", len(args))
	}

	cursorClause := ""
	if cursor != nil {
		args = append(args, cursor.CreatedAt, cursor.ID)
		cursorClause = fmt.Sprintf(
			" AND (created_at, id) < ($%d, $%d)",
			len(args)-1,
			len(args),
		)
	}

	args = append(args, limit)
	listSQL := fmt.Sprintf(`
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
				seller_legal_name,
				seller_registration_number,
				seller_vat_id,
				seller_address_line1,
				seller_address_line2,
				seller_city,
				seller_postal_code,
				seller_country,
				buyer_legal_name,
				buyer_vat_id,
				buyer_address_line1,
				buyer_address_line2,
				buyer_city,
				buyer_postal_code,
				buyer_country,
				vat_regime,
				vat_rate_bps,
				subtotal_cents,
				vat_cents,
				bank_iban,
				bank_bic,
				bank_name,
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
				AND (
					$3 = ''
					OR number ILIKE $3 ESCAPE '\'
					OR client_name ILIKE $3 ESCAPE '\'
					OR status ILIKE $3 ESCAPE '\'
				)%s%s
			ORDER BY created_at DESC, id DESC
			LIMIT $%d
		`, statusClause, cursorClause, len(args))

	rows, err := r.db.Query(ctx, listSQL, args...)
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
					seller_legal_name = $4,
					seller_registration_number = $5,
					seller_vat_id = $6,
					seller_address_line1 = $7,
					seller_address_line2 = $8,
					seller_city = $9,
					seller_postal_code = $10,
					seller_country = $11,
					buyer_legal_name = $12,
					buyer_vat_id = $13,
					buyer_address_line1 = $14,
					buyer_address_line2 = $15,
					buyer_city = $16,
					buyer_postal_code = $17,
					buyer_country = $18,
					vat_regime = $19,
					vat_rate_bps = $20,
					subtotal_cents = $21,
					vat_cents = $22,
					bank_iban = $23,
					bank_bic = $24,
					bank_name = $25,
					period_from = $26,
					period_to = $27,
					issued_at = $28,
					due_at = $29,
					sent_at = $30,
					paid_at = $31,
					total_minutes = $32,
					total_cents = $33,
					updated_at = $34
			WHERE id = $35 AND organization_id = $36
		`

	tag, err := database.QuerierFrom(ctx, r.db).Exec(
		ctx,
		query,
		invoice.Status,
		invoice.OrganizationName,
		invoice.ClientName,
		invoice.SellerLegalName,
		invoice.SellerRegistrationNumber,
		invoice.SellerVATID,
		invoice.SellerAddressLine1,
		invoice.SellerAddressLine2,
		invoice.SellerCity,
		invoice.SellerPostalCode,
		invoice.SellerCountry,
		invoice.BuyerLegalName,
		invoice.BuyerVATID,
		invoice.BuyerAddressLine1,
		invoice.BuyerAddressLine2,
		invoice.BuyerCity,
		invoice.BuyerPostalCode,
		invoice.BuyerCountry,
		invoice.VATRegime,
		invoice.VATRateBPS,
		invoice.SubtotalCents,
		invoice.VATCents,
		invoice.BankIBAN,
		invoice.BankBIC,
		invoice.BankName,
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
	var subtotalCents int64
	var vatCents int64
	var vatRateBPS int64

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
		&invoice.SellerLegalName,
		&invoice.SellerRegistrationNumber,
		&invoice.SellerVATID,
		&invoice.SellerAddressLine1,
		&invoice.SellerAddressLine2,
		&invoice.SellerCity,
		&invoice.SellerPostalCode,
		&invoice.SellerCountry,
		&invoice.BuyerLegalName,
		&invoice.BuyerVATID,
		&invoice.BuyerAddressLine1,
		&invoice.BuyerAddressLine2,
		&invoice.BuyerCity,
		&invoice.BuyerPostalCode,
		&invoice.BuyerCountry,
		&invoice.VATRegime,
		&vatRateBPS,
		&subtotalCents,
		&vatCents,
		&invoice.BankIBAN,
		&invoice.BankBIC,
		&invoice.BankName,
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

	invoice.VATRateBPS = int(vatRateBPS)
	invoice.SubtotalCents = int(subtotalCents)
	invoice.VATCents = int(vatCents)
	invoice.TotalMinutes = int(totalMinutes)
	invoice.TotalCents = int(totalCents)
	invoice.Lines = make([]domain.LineItem, 0)

	return &invoice, nil
}
