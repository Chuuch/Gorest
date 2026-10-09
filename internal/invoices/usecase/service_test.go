package usecase_test

import (
	"context"
	"testing"
	"time"

	clientpostgres "github.com/chuuch/gorest/internal/clients/postgres"
	clientuserpostgres "github.com/chuuch/gorest/internal/clients/users/postgres"
	invoicedomain "github.com/chuuch/gorest/internal/invoices/domain"
	invoicepostgres "github.com/chuuch/gorest/internal/invoices/postgres"
	invoicesusecase "github.com/chuuch/gorest/internal/invoices/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	"github.com/chuuch/gorest/internal/platform/mailer"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type mockMailer struct {
	messages []mailer.Message
}

func (m *mockMailer) Send(_ context.Context, msg mailer.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func setupInvoiceTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gorest_test"),
		tcpostgres.WithUsername("gorest"),
		tcpostgres.WithPassword("gorest"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	require.NoError(t, db.Ping(ctx))

	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE organizations (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			legal_name TEXT NOT NULL DEFAULT '',
			registration_number TEXT NOT NULL DEFAULT '',
			vat_id TEXT NOT NULL DEFAULT '',
			address_line1 TEXT NOT NULL DEFAULT '',
			address_line2 TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			postal_code TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			default_vat_rate_bps INTEGER NOT NULL DEFAULT 2000,
			bank_iban TEXT NOT NULL DEFAULT '',
			bank_bic TEXT NOT NULL DEFAULT '',
			bank_name TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE clients (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			legal_name TEXT NOT NULL DEFAULT '',
			vat_id TEXT NOT NULL DEFAULT '',
			address_line1 TEXT NOT NULL DEFAULT '',
			address_line2 TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			postal_code TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE client_users (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE projects (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			estimate_run_id UUID,
			estimated_hours NUMERIC,
			target_end_date DATE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT projects_client_name_unique UNIQUE (client_id, name)
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE tasks (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE time_entries (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			minutes INTEGER NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE invoices (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
			number TEXT NOT NULL,
			status TEXT NOT NULL,
			currency TEXT NOT NULL,
			rate_cents INTEGER NOT NULL,
			organization_name TEXT NOT NULL,
			client_name TEXT NOT NULL,
			seller_legal_name TEXT NOT NULL DEFAULT '',
			seller_registration_number TEXT NOT NULL DEFAULT '',
			seller_vat_id TEXT NOT NULL DEFAULT '',
			seller_address_line1 TEXT NOT NULL DEFAULT '',
			seller_address_line2 TEXT NOT NULL DEFAULT '',
			seller_city TEXT NOT NULL DEFAULT '',
			seller_postal_code TEXT NOT NULL DEFAULT '',
			seller_country TEXT NOT NULL DEFAULT '',
			buyer_legal_name TEXT NOT NULL DEFAULT '',
			buyer_vat_id TEXT NOT NULL DEFAULT '',
			buyer_address_line1 TEXT NOT NULL DEFAULT '',
			buyer_address_line2 TEXT NOT NULL DEFAULT '',
			buyer_city TEXT NOT NULL DEFAULT '',
			buyer_postal_code TEXT NOT NULL DEFAULT '',
			buyer_country TEXT NOT NULL DEFAULT '',
			vat_regime TEXT NOT NULL DEFAULT 'untaxed',
			vat_rate_bps INTEGER NOT NULL DEFAULT 0,
			subtotal_cents INTEGER NOT NULL DEFAULT 0,
			vat_cents INTEGER NOT NULL DEFAULT 0,
			bank_iban TEXT NOT NULL DEFAULT '',
			bank_bic TEXT NOT NULL DEFAULT '',
			bank_name TEXT NOT NULL DEFAULT '',
			period_from TIMESTAMPTZ NOT NULL,
			period_to TIMESTAMPTZ NOT NULL,
			issued_at TIMESTAMPTZ NOT NULL,
			due_at TIMESTAMPTZ NOT NULL,
			sent_at TIMESTAMPTZ,
			paid_at TIMESTAMPTZ,
			total_minutes INTEGER NOT NULL,
			total_cents INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (organization_id, number)
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE invoice_line_items (
			id UUID PRIMARY KEY,
			invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			project_name TEXT NOT NULL,
			task_title TEXT NOT NULL,
			minutes INTEGER NOT NULL,
			amount_cents INTEGER NOT NULL,
			position INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func newInvoiceService(db *pgxpool.Pool, mail *mockMailer) invoicesusecase.Service {
	return invoicesusecase.NewService(
		invoicepostgres.NewRepository(db),
		clientpostgres.NewRepository(db),
		orgpostgres.NewOrganizationRepository(db),
		clientuserpostgres.NewRepository(db),
		userpostgres.NewRepository(db),
		mail,
		db,
		"http://localhost:5173",
	)
}

func seedUser(t *testing.T, db *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		userID,
		email,
		"",
		"hash",
		now,
		now,
	)
	require.NoError(t, err)
	return userID
}

func seedOrganization(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	return seedOrganizationWithVAT(t, db, name, "")
}

func seedOrganizationWithVAT(
	t *testing.T,
	db *pgxpool.Pool,
	name string,
	vatID string,
) uuid.UUID {
	t.Helper()

	organizationID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
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
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14,
				$15, $16
			)
		`,
		organizationID,
		name,
		"",
		"",
		vatID,
		"1 Main St",
		"",
		"Sofia",
		"1000",
		"BG",
		2000,
		"",
		"",
		"",
		now,
		now,
	)
	require.NoError(t, err)
	return organizationID
}

func seedOrganizationIncomplete(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()

	organizationID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO organizations (id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		name,
		now,
		now,
	)
	require.NoError(t, err)
	return organizationID
}

func seedClient(t *testing.T, db *pgxpool.Pool, organizationID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	clientID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO clients (
				id,
				organization_id,
				name,
				notes,
				legal_name,
				vat_id,
				address_line1,
				address_line2,
				city,
				postal_code,
				country,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13
			)
		`,
		clientID,
		organizationID,
		name,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"BG",
		now,
		now,
	)
	require.NoError(t, err)
	return clientID
}

func seedClientUser(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID, userID uuid.UUID,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO client_users (id, organization_id, client_id, user_id, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		uuid.New(),
		organizationID,
		clientID,
		userID,
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func seedProject(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID uuid.UUID,
	name string,
) uuid.UUID {
	t.Helper()

	projectID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO projects (id, organization_id, client_id, name, notes, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		projectID,
		organizationID,
		clientID,
		name,
		"",
		now,
		now,
	)
	require.NoError(t, err)
	return projectID
}

func seedTask(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, projectID uuid.UUID,
	title string,
) uuid.UUID {
	t.Helper()

	taskID := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO tasks (id, organization_id, project_id, title, notes, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		taskID,
		organizationID,
		projectID,
		title,
		"",
		"todo",
		now,
		now,
	)
	require.NoError(t, err)
	return taskID
}

func seedTimeEntry(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, taskID, userID uuid.UUID,
	minutes int,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO time_entries (
				id,
				organization_id,
				task_id,
				user_id,
				minutes,
				notes,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		uuid.New(),
		organizationID,
		taskID,
		userID,
		minutes,
		"",
		createdAt,
		createdAt,
	)
	require.NoError(t, err)
}

func TestCreateInvoice_SnapshotsLines(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	mail := &mockMailer{}
	service := newInvoiceService(db, mail)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	seedTimeEntry(t, db, organizationID, taskID, userID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))
	seedTimeEntry(t, db, organizationID, taskID, userID, 15, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))

	invoice, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)
	require.Equal(t, invoicedomain.StatusDraft, invoice.Status)
	require.Equal(t, invoicedomain.CurrencyEUR, invoice.Currency)
	require.Equal(t, 3000, invoice.RateCents)
	require.Equal(t, "Acme", invoice.OrganizationName)
	require.Equal(t, "Northwind", invoice.ClientName)
	require.Equal(t, "INV-2026-0001", invoice.Number)
	require.Equal(t, 90, invoice.TotalMinutes)
	require.Equal(t, invoicedomain.RegimeUntaxed, invoice.VATRegime)
	require.Equal(t, 4500, invoice.SubtotalCents)
	require.Equal(t, 0, invoice.VATCents)
	require.Equal(t, 4500, invoice.TotalCents)
	require.Equal(t, "BG", invoice.SellerCountry)
	require.Equal(t, "1 Main St", invoice.SellerAddressLine1)
	require.Len(t, invoice.Lines, 1)
	require.Equal(t, "Portal", invoice.Lines[0].ProjectName)
	require.Equal(t, "Draw", invoice.Lines[0].TaskTitle)
	require.Equal(t, 90, invoice.Lines[0].Minutes)
	require.Equal(t, 4500, invoice.Lines[0].AmountCents)
}

func TestCreateInvoice_StandardVAT(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	service := newInvoiceService(db, &mockMailer{})
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganizationWithVAT(t, db, "Acme", "BG123456789")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	seedTimeEntry(t, db, organizationID, taskID, userID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))

	invoice, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)
	require.Equal(t, invoicedomain.RegimeStandard, invoice.VATRegime)
	require.Equal(t, 2000, invoice.VATRateBPS)
	require.Equal(t, 4500, invoice.SubtotalCents)
	require.Equal(t, 900, invoice.VATCents)
	require.Equal(t, 5400, invoice.TotalCents)
	require.Equal(t, "BG123456789", invoice.SellerVATID)
}

func TestCreateInvoice_MemberForbidden(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	service := newInvoiceService(db, &mockMailer{})
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	_, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleMember, from, to)
	require.ErrorIs(t, err, invoicedomain.ErrForbidden)
}

func TestCreateInvoice_NoLineItems(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	service := newInvoiceService(db, &mockMailer{})
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	_, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.ErrorIs(t, err, invoicedomain.ErrNoLineItems)
}

func TestSendInvoice_EmailsClientUsers(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	mail := &mockMailer{}
	service := newInvoiceService(db, mail)
	staffID := seedUser(t, db, "ada@example.com")
	portalID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	seedClientUser(t, db, organizationID, clientID, portalID)
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	seedTimeEntry(t, db, organizationID, taskID, staffID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))

	invoice, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)

	sent, err := service.Send(context.Background(), organizationID, invoice.ID, orgdomain.RoleOwner)
	require.NoError(t, err)
	require.Equal(t, invoicedomain.StatusSent, sent.Status)
	require.NotNil(t, sent.SentAt)
	require.Len(t, mail.messages, 1)
	require.Equal(t, "pat@example.com", mail.messages[0].To)
	require.Contains(t, mail.messages[0].Subject, "INV-2026-0001")
	require.Contains(t, mail.messages[0].HTML, "Portal")
	require.Contains(t, mail.messages[0].HTML, "Draw")
	require.Contains(t, mail.messages[0].HTML, "Subtotal")
	require.Contains(t, mail.messages[0].HTML, "€45.00")
	require.Contains(t, mail.messages[0].Text, "1.50 h")

	err = service.Delete(context.Background(), organizationID, invoice.ID, orgdomain.RoleOwner)
	require.ErrorIs(t, err, invoicedomain.ErrNotDraft)

	paid, err := service.MarkPaid(context.Background(), organizationID, invoice.ID, orgdomain.RoleOwner)
	require.NoError(t, err)
	require.Equal(t, invoicedomain.StatusPaid, paid.Status)
	require.NotNil(t, paid.PaidAt)
}

func TestSendInvoice_BillingIncomplete(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	service := newInvoiceService(db, &mockMailer{})
	staffID := seedUser(t, db, "ada@example.com")
	portalID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganizationIncomplete(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	seedClientUser(t, db, organizationID, clientID, portalID)
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	seedTimeEntry(t, db, organizationID, taskID, staffID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))

	invoice, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)

	_, err = service.Send(context.Background(), organizationID, invoice.ID, orgdomain.RoleOwner)
	require.ErrorIs(t, err, invoicedomain.ErrBillingProfileIncomplete)
}

func TestSendInvoice_NoClientUsers(t *testing.T) {
	db, cleanup := setupInvoiceTestDatabase(t)
	defer cleanup()

	service := newInvoiceService(db, &mockMailer{})
	staffID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	seedTimeEntry(t, db, organizationID, taskID, staffID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))

	invoice, err := service.Create(context.Background(), organizationID, clientID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)

	_, err = service.Send(context.Background(), organizationID, invoice.ID, orgdomain.RoleOwner)
	require.ErrorIs(t, err, invoicedomain.ErrNoClientUsers)
}
