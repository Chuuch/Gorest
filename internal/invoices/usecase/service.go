package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/clients/domain"
	clientrepository "github.com/chuuch/gorest/internal/clients/repository"
	clientuserdomain "github.com/chuuch/gorest/internal/clients/users/domain"
	clientuserrepository "github.com/chuuch/gorest/internal/clients/users/repository"
	"github.com/chuuch/gorest/internal/invoices/domain"
	invoicepdf "github.com/chuuch/gorest/internal/invoices/pdf"
	invoicerepository "github.com/chuuch/gorest/internal/invoices/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	"github.com/chuuch/gorest/internal/platform/database"
	"github.com/chuuch/gorest/internal/platform/mailer"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/search"
	"github.com/chuuch/gorest/internal/platform/taxid"
	userrepository "github.com/chuuch/gorest/internal/user/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
	) ([]*domain.Invoice, *string, error)
	Get(ctx context.Context, organizationID, invoiceID uuid.UUID) (*domain.Invoice, error)
	Create(ctx context.Context, organizationID, clientID uuid.UUID, actorRole orgdomain.Role, from, to time.Time) (*domain.Invoice, error)
	Update(ctx context.Context, organizationID, invoiceID uuid.UUID, actorRole orgdomain.Role, from, to time.Time) (*domain.Invoice, error)
	Delete(ctx context.Context, organizationID, invoiceID uuid.UUID, actorRole orgdomain.Role) error
	Send(ctx context.Context, organizationID, invoiceID uuid.UUID, actorRole orgdomain.Role) (*domain.Invoice, error)
	MarkPaid(ctx context.Context, organizationID, invoiceID uuid.UUID, actorRole orgdomain.Role) (*domain.Invoice, error)
	PDF(ctx context.Context, organizationID, invoiceID uuid.UUID) ([]byte, string, error)
	ListPortal(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
	) ([]*domain.Invoice, *string, error)
	GetPortal(ctx context.Context, organizationID, clientID, invoiceID uuid.UUID) (*domain.Invoice, error)
	PDFPortal(ctx context.Context, organizationID, clientID, invoiceID uuid.UUID) ([]byte, string, error)
}

type service struct {
	invoices      invoicerepository.InvoiceRepository
	clients       clientrepository.ClientRepository
	organizations orgrepository.OrganizationRepository
	clientUsers   clientuserrepository.ClientUserRepository
	users         userrepository.UserRepository
	mailer        mailer.Mailer
	db            *pgxpool.Pool
	publicURL     string
}

func NewService(
	invoices invoicerepository.InvoiceRepository,
	clients clientrepository.ClientRepository,
	organizations orgrepository.OrganizationRepository,
	clientUsers clientuserrepository.ClientUserRepository,
	users userrepository.UserRepository,
	mailer mailer.Mailer,
	db *pgxpool.Pool,
	publicURL string,
) Service {
	return &service{
		invoices:      invoices,
		clients:       clients,
		organizations: organizations,
		clientUsers:   clientUsers,
		users:         users,
		mailer:        mailer,
		db:            db,
		publicURL:     strings.TrimRight(publicURL, "/"),
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	query string,
) ([]*domain.Invoice, *string, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, nil, err
	}

	invoices, err := s.invoices.ListByClientID(
		ctx,
		organizationID,
		clientID,
		limit+1,
		cursor,
		search.Normalize(query),
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list invoices: %w", err)
	}

	page, next := pagination.NextCursor(invoices, limit, func(invoice *domain.Invoice) pagination.Cursor {
		return pagination.Cursor{
			CreatedAt: invoice.CreatedAt,
			ID:        invoice.ID,
		}
	})
	return page, next, nil
}

func (s *service) Get(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
) (*domain.Invoice, error) {
	return s.invoices.GetByID(ctx, invoiceID, organizationID)
}

func (s *service) Create(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*domain.Invoice, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	client, err := s.clients.GetByID(ctx, clientID, organizationID)
	if err != nil {
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	lines, err := s.invoices.SnapshotLines(ctx, organizationID, clientID, from, to)
	if err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, domain.ErrNoLineItems
	}

	now := time.Now().UTC()
	invoice := &domain.Invoice{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ClientID:       clientID,
		Status:         domain.StatusDraft,
		Currency:       domain.CurrencyEUR,
		RateCents:      domain.DefaultRateCents,
		PeriodFrom:     from,
		PeriodTo:       to,
		IssuedAt:       now,
		DueAt:          domain.DueAt(now),
		CreatedAt:      now,
	}
	fillInvoiceBilling(invoice, org, client)
	domain.ApplySnapshot(invoice, lines, now)
	domain.ApplyTax(invoice)

	for attempt := 0; attempt < 2; attempt++ {
		err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
			number, err := s.invoices.NextNumber(ctx, organizationID, now.Year())
			if err != nil {
				return err
			}
			invoice.Number = number
			return s.invoices.Create(ctx, invoice)
		})
		if err == nil {
			return invoice, nil
		}
		if errors.Is(err, domain.ErrInvoiceNumberExists) {
			continue
		}
		return nil, err
	}
	return nil, domain.ErrInvoiceNumberExists
}

func (s *service) Update(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*domain.Invoice, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != domain.StatusDraft {
		return nil, domain.ErrNotDraft
	}

	client, err := s.clients.GetByID(ctx, invoice.ClientID, organizationID)
	if err != nil {
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	lines, err := s.invoices.SnapshotLines(ctx, organizationID, invoice.ClientID, from, to)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, domain.ErrNoLineItems
	}

	now := time.Now().UTC()
	fillInvoiceBilling(invoice, org, client)
	invoice.PeriodFrom = from
	invoice.PeriodTo = to
	domain.ApplySnapshot(invoice, lines, now)
	domain.ApplyTax(invoice)

	if err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		if err := s.invoices.Update(ctx, invoice); err != nil {
			return err
		}

		return s.invoices.ReplaceLines(ctx, invoice)
	}); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return domain.ErrForbidden
	}

	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return err
	}

	if invoice.Status != domain.StatusDraft {
		return domain.ErrNotDraft
	}

	return s.invoices.Delete(ctx, invoiceID, organizationID)
}

func (s *service) Send(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) (*domain.Invoice, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, err
	}

	if invoice.Status != domain.StatusDraft {
		return nil, domain.ErrNotDraft
	}

	if err := invoice.BillingReady(); err != nil {
		return nil, err
	}

	clientUsers, err := s.clientUsers.ListByClientID(ctx, organizationID, invoice.ClientID)
	if err != nil {
		return nil, fmt.Errorf("list client users: %w", err)
	}
	if len(clientUsers) == 0 {
		return nil, domain.ErrNoClientUsers
	}

	now := time.Now().UTC()
	invoice.Status = domain.StatusSent
	invoice.IssuedAt = now
	invoice.DueAt = domain.DueAt(now)
	invoice.SentAt = &now
	invoice.UpdatedAt = now

	if err := s.invoices.Update(ctx, invoice); err != nil {
		return nil, err
	}

	if err := s.mailInvoice(ctx, invoice, clientUsers); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *service) MarkPaid(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) (*domain.Invoice, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, err
	}

	if invoice.Status != domain.StatusSent {
		return nil, domain.ErrNotSent
	}

	now := time.Now().UTC()
	invoice.Status = domain.StatusPaid
	invoice.PaidAt = &now
	invoice.UpdatedAt = now

	if err := s.invoices.Update(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *service) PDF(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
) ([]byte, string, error) {
	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, "", err
	}

	data, err := invoicepdf.Generate(invoice)
	if err != nil {
		return nil, "", fmt.Errorf("generate invoice pdf: %w", err)
	}

	return data, invoice.Number + ".pdf", nil
}

func (s *service) ListPortal(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	query string,
) ([]*domain.Invoice, *string, error) {
	invoices, err := s.invoices.ListByClientID(
		ctx,
		organizationID,
		clientID,
		limit+1,
		cursor,
		search.Normalize(query),
		[]string{domain.StatusSent, domain.StatusPaid},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list portal invoices: %w", err)
	}

	page, next := pagination.NextCursor(invoices, limit, func(invoice *domain.Invoice) pagination.Cursor {
		return pagination.Cursor{
			CreatedAt: invoice.CreatedAt,
			ID:        invoice.ID,
		}
	})
	return page, next, nil
}

func (s *service) GetPortal(
	ctx context.Context,
	organizationID, clientID, invoiceID uuid.UUID,
) (*domain.Invoice, error) {
	invoice, err := s.invoices.GetByID(ctx, invoiceID, organizationID)
	if err != nil {
		return nil, err
	}

	if invoice.ClientID != clientID || invoice.Status == domain.StatusDraft {
		return nil, domain.ErrInvoiceNotFound
	}

	return invoice, nil
}

func (s *service) PDFPortal(
	ctx context.Context,
	organizationID, clientID, invoiceID uuid.UUID,
) ([]byte, string, error) {
	invoice, err := s.GetPortal(ctx, organizationID, clientID, invoiceID)
	if err != nil {
		return nil, "", err
	}

	data, err := invoicepdf.Generate(invoice)
	if err != nil {
		return nil, "", fmt.Errorf("generate invoice pdf: %w", err)
	}

	return data, invoice.Number + ".pdf", nil
}

func fillInvoiceBilling(
	invoice *domain.Invoice,
	org *orgdomain.Organization,
	client *clientdomain.Client,
) {
	invoice.OrganizationName = org.InvoiceLegalName()
	invoice.ClientName = client.InvoiceLegalName()
	invoice.SellerLegalName = invoice.OrganizationName
	invoice.SellerRegistrationNumber = strings.TrimSpace(org.RegistrationNumber)
	invoice.SellerVATID = taxid.NormalizeVATID(org.VATID)
	invoice.SellerAddressLine1 = strings.TrimSpace(org.AddressLine1)
	invoice.SellerAddressLine2 = strings.TrimSpace(org.AddressLine2)
	invoice.SellerCity = strings.TrimSpace(org.City)
	invoice.SellerPostalCode = strings.TrimSpace(org.PostalCode)
	invoice.SellerCountry = taxid.NormalizeCountry(org.Country)
	invoice.BuyerLegalName = client.InvoiceLegalName()
	invoice.BuyerVATID = taxid.NormalizeVATID(client.VATID)
	invoice.BuyerAddressLine1 = strings.TrimSpace(client.AddressLine1)
	invoice.BuyerAddressLine2 = strings.TrimSpace(client.AddressLine2)
	invoice.BuyerCity = strings.TrimSpace(client.City)
	invoice.BuyerPostalCode = strings.TrimSpace(client.PostalCode)
	invoice.BuyerCountry = taxid.NormalizeCountry(client.Country)
	invoice.BankIBAN = strings.TrimSpace(org.BankIBAN)
	invoice.BankBIC = strings.TrimSpace(org.BankBIC)
	invoice.BankName = strings.TrimSpace(org.BankName)

	rate := org.DefaultVATRateBPS
	if rate == 0 {
		rate = domain.DefaultVATRateBPS
	}
	invoice.VATRegime, invoice.VATRateBPS = domain.ResolveVATRegime(
		invoice.SellerVATID,
		invoice.SellerCountry,
		invoice.BuyerVATID,
		invoice.BuyerCountry,
		rate,
	)
}

func (s *service) mailInvoice(
	ctx context.Context,
	invoice *domain.Invoice,
	clientUsers []*clientuserdomain.ClientUser,
) error {
	html, err := mailer.RenderInvoice(toInvoiceMail(invoice, s.publicURL+"/portal"))
	if err != nil {
		return fmt.Errorf("render invoice mail: %w", err)
	}

	subject := invoice.Number + " from " + invoice.OrganizationName
	text := invoiceText(invoice)

	for _, clientUser := range clientUsers {
		user, err := s.users.GetByID(ctx, clientUser.UserID)
		if err != nil {
			return fmt.Errorf("get client user: %w", err)
		}

		if err := s.mailer.Send(ctx, mailer.Message{
			To:      user.Email,
			Subject: subject,
			Text:    text,
			HTML:    html,
		}); err != nil {
			return fmt.Errorf("send invoice email: %w", err)
		}
	}

	return nil
}

func toInvoiceMail(invoice *domain.Invoice, actionURL string) mailer.InvoiceMail {
	lines := make([]mailer.InvoiceMailLine, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		lines = append(lines, mailer.InvoiceMailLine{
			ProjectName: line.ProjectName,
			TaskTitle:   line.TaskTitle,
			Hours:       formatHours(line.Minutes),
			Amount:      formatEUR(line.AmountCents),
		})
	}
	inclusiveTo := invoice.PeriodTo.Add(-time.Nanosecond)
	showVAT := invoice.VATRegime == domain.RegimeStandard
	return mailer.InvoiceMail{
		Heading:                  invoice.Number,
		AgencyName:               invoice.OrganizationName,
		SellerRegistrationNumber: invoice.SellerRegistrationNumber,
		SellerVATID:              invoice.SellerVATID,
		SellerAddressLine1:       invoice.SellerAddressLine1,
		SellerAddressLine2:       invoice.SellerAddressLine2,
		SellerCityLine: formatCityLine(
			invoice.SellerCity,
			invoice.SellerPostalCode,
			invoice.SellerCountry,
		),
		ClientName:        invoice.ClientName,
		BuyerVATID:        invoice.BuyerVATID,
		BuyerAddressLine1: invoice.BuyerAddressLine1,
		BuyerAddressLine2: invoice.BuyerAddressLine2,
		BuyerCityLine: formatCityLine(
			invoice.BuyerCity,
			invoice.BuyerPostalCode,
			invoice.BuyerCountry,
		),
		Number:        invoice.Number,
		Issued:        invoice.IssuedAt.UTC().Format("2 Jan 2006"),
		Period:        invoice.PeriodFrom.UTC().Format("2 Jan 2006") + " - " + inclusiveTo.UTC().Format("2 Jan 2006"),
		Due:           invoice.DueAt.UTC().Format("2 Jan 2006"),
		Rate:          formatEUR(invoice.RateCents) + " / h",
		Subtotal:      formatEUR(invoice.SubtotalCents),
		VATLabel:      "VAT " + formatRate(invoice.VATRateBPS) + "%",
		VATAmount:     formatEUR(invoice.VATCents),
		VATNote:       domain.VATNote(invoice.VATRegime),
		ShowVATAmount: showVAT,
		Total:         formatEUR(invoice.TotalCents),
		BankIBAN:      invoice.BankIBAN,
		BankBIC:       invoice.BankBIC,
		BankName:      invoice.BankName,
		Lines:         lines,
		ActionURL:     actionURL,
		ActionLabel:   "Open portal",
	}
}

func invoiceText(invoice *domain.Invoice) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s from %s\n", invoice.Number, invoice.OrganizationName)
	if invoice.SellerAddressLine1 != "" {
		fmt.Fprintf(&b, "%s\n", invoice.SellerAddressLine1)
	}
	if city := formatCityLine(invoice.SellerCity, invoice.SellerPostalCode, invoice.SellerCountry); city != "" {
		fmt.Fprintf(&b, "%s\n", city)
	}
	if invoice.SellerVATID != "" {
		fmt.Fprintf(&b, "VAT %s\n", invoice.SellerVATID)
	}
	fmt.Fprintf(&b, "Bill to: %s\n", invoice.ClientName)
	if invoice.BuyerVATID != "" {
		fmt.Fprintf(&b, "VAT %s\n", invoice.BuyerVATID)
	}
	fmt.Fprintf(&b, "Due: %s\n", invoice.DueAt.UTC().Format("2 Jan 2006"))
	fmt.Fprintf(&b, "Rate: %s / h\n\n", formatEUR(invoice.RateCents))
	for _, line := range invoice.Lines {
		fmt.Fprintf(
			&b,
			"%s | %s | %s h | %s\n",
			line.ProjectName,
			line.TaskTitle,
			formatHours(line.Minutes),
			formatEUR(line.AmountCents),
		)
	}
	fmt.Fprintf(&b, "\nSubtotal: %s\n", formatEUR(invoice.SubtotalCents))
	if invoice.VATRegime == domain.RegimeStandard {
		fmt.Fprintf(&b, "VAT %s%%: %s\n", formatRate(invoice.VATRateBPS), formatEUR(invoice.VATCents))
	} else if note := domain.VATNote(invoice.VATRegime); note != "" {
		fmt.Fprintf(&b, "%s\n", note)
	}
	fmt.Fprintf(&b, "Total: %s\n", formatEUR(invoice.TotalCents))
	if invoice.BankIBAN != "" {
		fmt.Fprintf(&b, "\nIBAN: %s\n", invoice.BankIBAN)
		if invoice.BankBIC != "" {
			fmt.Fprintf(&b, "BIC: %s\n", invoice.BankBIC)
		}
	}
	return b.String()
}
func formatHours(minutes int) string {
	return fmt.Sprintf("%.2f", float64(minutes)/60)
}
func formatEUR(cents int) string {
	return fmt.Sprintf("€%.2f", float64(cents)/100)
}
func formatRate(bps int) string {
	return fmt.Sprintf("%.2f", float64(bps)/100)
}
func formatCityLine(city, postal, country string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(postal) != "" {
		parts = append(parts, strings.TrimSpace(postal))
	}
	if strings.TrimSpace(city) != "" {
		parts = append(parts, strings.TrimSpace(city))
	}
	line := strings.Join(parts, " ")
	country = strings.TrimSpace(country)
	if country == "" {
		return line
	}
	if line == "" {
		return country
	}
	return line + ", " + country
}
