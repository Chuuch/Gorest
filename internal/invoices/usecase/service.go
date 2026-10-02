package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	clientrepository "github.com/chuuch/gorest/internal/client/repository"
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	clientuserrepository "github.com/chuuch/gorest/internal/clientusers/repository"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/invoices/domain"
	invoicepdf "github.com/chuuch/gorest/internal/invoices/pdf"
	invoicerepository "github.com/chuuch/gorest/internal/invoices/repository"
	"github.com/chuuch/gorest/internal/mailer"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	userrepository "github.com/chuuch/gorest/internal/user/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
	) ([]*domain.Invoice, error)
	Get(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
	) (*domain.Invoice, error)
	Create(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		actorRole orgdomain.Role,
		from, to time.Time,
	) (*domain.Invoice, error)
	Update(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
		actorRole orgdomain.Role,
		from, to time.Time,
	) (*domain.Invoice, error)
	Delete(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
		actorRole orgdomain.Role,
	) error
	Send(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
		actorRole orgdomain.Role,
	) (*domain.Invoice, error)
	MarkPaid(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
		actorRole orgdomain.Role,
	) (*domain.Invoice, error)
	PDF(
		ctx context.Context,
		organizationID, invoiceID uuid.UUID,
	) ([]byte, string, error)
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
) ([]*domain.Invoice, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	invoices, err := s.invoices.ListByClientID(ctx, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	return invoices, nil
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
		ID:               uuid.New(),
		OrganizationID:   organizationID,
		ClientID:         clientID,
		Status:           domain.StatusDraft,
		Currency:         domain.CurrencyEUR,
		RateCents:        domain.DefaultRateCents,
		OrganizationName: org.Name,
		ClientName:       client.Name,
		PeriodFrom:       from,
		PeriodTo:         to,
		IssuedAt:         now,
		DueAt:            domain.DueAt(now),
		CreatedAt:        now,
	}
	domain.ApplySnapshot(invoice, lines, now)

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
	invoice.OrganizationName = org.Name
	invoice.ClientName = client.Name
	invoice.PeriodFrom = from
	invoice.PeriodTo = to
	domain.ApplySnapshot(invoice, lines, now)

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

	return mailer.InvoiceMail{
		Heading:     invoice.Number,
		AgencyName:  invoice.OrganizationName,
		ClientName:  invoice.ClientName,
		Number:      invoice.Number,
		Issued:      invoice.IssuedAt.UTC().Format("2 Jan 2006"),
		Period:      invoice.PeriodFrom.UTC().Format("2 Jan 2006") + " - " + inclusiveTo.UTC().Format("2 Jan 2006"),
		Due:         invoice.DueAt.UTC().Format("2 Jan 2006"),
		Rate:        formatEUR(invoice.RateCents) + " / h",
		Total:       formatEUR(invoice.TotalCents),
		Lines:       lines,
		ActionURL:   actionURL,
		ActionLabel: "Open portal",
	}
}

func invoiceText(invoice *domain.Invoice) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s from %s\n", invoice.Number, invoice.OrganizationName)
	fmt.Fprintf(&b, "Bill to: %s\n", invoice.ClientName)
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
	fmt.Fprintf(&b, "\nTotal: %s\n", formatEUR(invoice.TotalCents))
	return b.String()
}

func formatHours(minutes int) string {
	return fmt.Sprintf("%.2f", float64(minutes)/60)
}

func formatEUR(cents int) string {
	return fmt.Sprintf("€%.2f", float64(cents)/100)
}
