package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusDraft = "draft"
	StatusSent  = "sent"
	StatusPaid  = "paid"

	CurrencyEUR      = "EUR"
	CurrencyUSD      = "USD"
	CurrencyCAD      = "CAD"
	CurrencyGBP      = "GBP"
	CurrencyAUD      = "AUD"
	DefaultRateCents = 3000
)

type LineItem struct {
	ID             uuid.UUID
	InvoiceID      uuid.UUID
	OrganizationID uuid.UUID
	ProjectName    string
	TaskTitle      string
	Minutes        int
	AmountCents    int
	Position       int
	CreatedAt      time.Time
}

type Invoice struct {
	ID                       uuid.UUID
	OrganizationID           uuid.UUID
	ClientID                 uuid.UUID
	Number                   string
	Status                   string
	Currency                 string
	RateCents                int
	OrganizationName         string
	ClientName               string
	SellerLegalName          string
	SellerRegistrationNumber string
	SellerVATID              string
	SellerAddressLine1       string
	SellerAddressLine2       string
	SellerCity               string
	SellerPostalCode         string
	SellerCountry            string
	BuyerLegalName           string
	BuyerVATID               string
	BuyerAddressLine1        string
	BuyerAddressLine2        string
	BuyerCity                string
	BuyerPostalCode          string
	BuyerCountry             string
	VATRegime                string
	VATRateBPS               int
	SubtotalCents            int
	VATCents                 int
	BankIBAN                 string
	BankBIC                  string
	BankName                 string
	PeriodFrom               time.Time
	PeriodTo                 time.Time
	IssuedAt                 time.Time
	DueAt                    time.Time
	SentAt                   *time.Time
	PaidAt                   *time.Time
	TotalMinutes             int
	TotalCents               int
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Lines                    []LineItem
}

type SnapshotLine struct {
	ProjectName string
	TaskTitle   string
	Minutes     int
}

func AmountCents(minutes, rateCents int) int {
	return minutes * rateCents / 60
}

func DueAt(issuedAt time.Time) time.Time {
	return issuedAt.Add(14 * 24 * time.Hour)
}

func ApplySnapshot(invoice *Invoice, lines []SnapshotLine, now time.Time) {
	items := make([]LineItem, 0, len(lines))
	totalMinutes := 0
	subtotalCents := 0

	for i, line := range lines {
		amount := AmountCents(line.Minutes, invoice.RateCents)
		totalMinutes += line.Minutes
		subtotalCents += amount
		items = append(items, LineItem{
			ID:             uuid.New(),
			InvoiceID:      invoice.ID,
			OrganizationID: invoice.OrganizationID,
			ProjectName:    line.ProjectName,
			TaskTitle:      line.TaskTitle,
			Minutes:        line.Minutes,
			AmountCents:    amount,
			Position:       i + 1,
			CreatedAt:      now,
		})
	}

	invoice.Lines = items
	invoice.TotalMinutes = totalMinutes
	invoice.SubtotalCents = subtotalCents
	invoice.UpdatedAt = now
}
