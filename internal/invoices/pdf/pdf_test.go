package pdf_test

import (
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/invoices/domain"
	invoicepdf "github.com/chuuch/gorest/internal/invoices/pdf"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	invoice := &domain.Invoice{
		ID:               uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		OrganizationID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		ClientID:         uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Number:           "INV-2026-0001",
		Status:           domain.StatusDraft,
		Currency:         domain.CurrencyEUR,
		RateCents:        3000,
		OrganizationName: "Acme",
		ClientName:       "Northwind",
		PeriodFrom:       time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		PeriodTo:         time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		IssuedAt:         time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		DueAt:            time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
		TotalMinutes:     90,
		TotalCents:       4500,
		Lines: []domain.LineItem{
			{
				ProjectName: "Portal",
				TaskTitle:   "Draw",
				Minutes:     90,
				AmountCents: 4500,
				Position:    1,
			},
		},
	}

	data, err := invoicepdf.Generate(invoice)
	require.NoError(t, err)
	require.Greater(t, len(data), 4)
	require.Equal(t, "%PDF", string(data[:4]))
}
