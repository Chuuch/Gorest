package domain_test

import (
	"testing"

	"github.com/chuuch/gorest/internal/invoices/domain"
	"github.com/stretchr/testify/require"
)

func TestResolveVATRegime(t *testing.T) {
	regime, rate := domain.ResolveVATRegime("", "BG", "DE123", "DE", 2000)
	require.Equal(t, domain.RegimeUntaxed, regime)
	require.Equal(t, 0, rate)

	regime, rate = domain.ResolveVATRegime("BG123", "BG", "DE999", "DE", 2000)
	require.Equal(t, domain.RegimeReverseCharge, regime)
	require.Equal(t, 0, rate)

	regime, rate = domain.ResolveVATRegime("BG123", "BG", "", "US", 2000)
	require.Equal(t, domain.RegimeOutsideScope, regime)
	require.Equal(t, 0, rate)

	regime, rate = domain.ResolveVATRegime("BG123", "BG", "BG999", "BG", 2000)
	require.Equal(t, domain.RegimeStandard, regime)
	require.Equal(t, 2000, rate)

	regime, rate = domain.ResolveVATRegime("BG123", "BG", "", "DE", 2000)
	require.Equal(t, domain.RegimeStandard, regime)
	require.Equal(t, 2000, rate)
}

func TestVATCents(t *testing.T) {
	require.Equal(t, 900, domain.VATCents(4500, 2000))
	require.Equal(t, 0, domain.VATCents(0, 2000))
	require.Equal(t, 0, domain.VATCents(4500, 0))
}

func TestApplyTax(t *testing.T) {
	invoice := &domain.Invoice{
		SubtotalCents: 4500,
		VATRegime:     domain.RegimeStandard,
		VATRateBPS:    2000,
	}
	domain.ApplyTax(invoice)
	require.Equal(t, 900, invoice.VATCents)
	require.Equal(t, 5400, invoice.TotalCents)

	invoice = &domain.Invoice{
		SubtotalCents: 4500,
		VATRegime:     domain.RegimeReverseCharge,
		VATRateBPS:    2000,
	}
	domain.ApplyTax(invoice)
	require.Equal(t, 0, invoice.VATRateBPS)
	require.Equal(t, 0, invoice.VATCents)
	require.Equal(t, 4500, invoice.TotalCents)
}

func TestBillingReady(t *testing.T) {
	invoice := &domain.Invoice{
		SellerLegalName:    "Agency EOOD",
		SellerAddressLine1: "1 Main St",
		SellerCity:         "Sofia",
		SellerPostalCode:   "1000",
		SellerCountry:      "BG",
		SellerVATID:        "BG123",
		BuyerCountry:       "DE",
		VATRegime:          domain.RegimeReverseCharge,
	}
	require.NoError(t, invoice.BillingReady())

	invoice.SellerCountry = ""
	require.ErrorIs(t, invoice.BillingReady(), domain.ErrBillingProfileIncomplete)
}
