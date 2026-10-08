package mailer_test

import (
	"testing"

	"github.com/chuuch/gorest/internal/platform/mailer"
	"github.com/stretchr/testify/require"
)

func TestRenderTransactional(t *testing.T) {
	html, err := mailer.RenderTransactional(mailer.Transactional{
		Heading:     "You're invited",
		Body:        "You've been invited to join Acme.",
		ActionURL:   "http://localhost:5173/accept-invite?token=abc",
		ActionLabel: "Set your password",
		Expiry:      "This link expires in 7 days.",
	})
	require.NoError(t, err)
	require.Contains(t, html, "Flourish")
	require.Contains(t, html, "You&#39;re invited")
	require.Contains(t, html, "You&#39;ve been invited to join Acme.")
	require.Contains(t, html, "Set your password")
	require.Contains(t, html, "http://localhost:5173/accept-invite?token=abc")
	require.Contains(t, html, "#ff5c00")
	require.Contains(t, html, "#f3f6f4")
	require.Contains(t, html, "This link expires in 7 days.")
}

func TestRenderTransactional_EscapesHTML(t *testing.T) {
	html, err := mailer.RenderTransactional(mailer.Transactional{
		Heading:     "<script>alert(1)</script>",
		Body:        "Join <b>Acme</b>.",
		ActionURL:   "http://localhost:5173/accept-invite?token=abc",
		ActionLabel: "Set your password",
		Expiry:      "This link expires in 7 days.",
	})
	require.NoError(t, err)
	require.NotContains(t, html, "<script>alert(1)</script>")
	require.Contains(t, html, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.NotContains(t, html, "<b>Acme</b>")
	require.Contains(t, html, "Join &lt;b&gt;Acme&lt;/b&gt;.")
}

func TestRenderInvoice(t *testing.T) {
	html, err := mailer.RenderInvoice(mailer.InvoiceMail{
		Heading:                  "INV-2026-0001",
		AgencyName:               "Acme EOOD",
		SellerRegistrationNumber: "123456789",
		SellerVATID:              "BG123456789",
		SellerAddressLine1:       "1 Main St",
		SellerCityLine:           "1000 Sofia, BG",
		ClientName:               "Northwind GmbH",
		BuyerVATID:               "DE999999999",
		BuyerAddressLine1:        "2 River Rd",
		BuyerCityLine:            "10115 Berlin, DE",
		Number:                   "INV-2026-0001",
		Issued:                   "1 Oct 2026",
		Period:                   "28 Sep 2026 – 4 Oct 2026",
		Due:                      "15 Oct 2026",
		Rate:                     "€30.00 / h",
		Subtotal:                 "€45.00",
		VATNote:                  "VAT reverse charge — customer accounts for VAT.",
		ShowVATAmount:            false,
		Total:                    "€45.00",
		BankIBAN:                 "BG80BNBG96611020345678",
		BankBIC:                  "BNBGBGSF",
		BankName:                 "Demo Bank",
		ActionURL:                "http://localhost:5173/portal",
		ActionLabel:              "Open portal",
		Lines: []mailer.InvoiceMailLine{
			{
				ProjectName: "Portal",
				TaskTitle:   "Draw",
				Hours:       "1.50",
				Amount:      "€45.00",
			},
		},
	})
	require.NoError(t, err)
	require.Contains(t, html, "Flourish")
	require.Contains(t, html, "INV-2026-0001")
	require.Contains(t, html, "Acme EOOD")
	require.Contains(t, html, "VAT BG123456789")
	require.Contains(t, html, "Northwind GmbH")
	require.Contains(t, html, "VAT DE999999999")
	require.Contains(t, html, "Portal")
	require.Contains(t, html, "Draw")
	require.Contains(t, html, "1.50")
	require.Contains(t, html, "Subtotal")
	require.Contains(t, html, "€45.00")
	require.Contains(t, html, "VAT reverse charge")
	require.Contains(t, html, "IBAN BG80BNBG96611020345678")
	require.Contains(t, html, "#ff5c00")
	require.Contains(t, html, "http://localhost:5173/portal")
}

func TestRenderInvoice_StandardVAT(t *testing.T) {
	html, err := mailer.RenderInvoice(mailer.InvoiceMail{
		Heading:       "INV-2026-0002",
		AgencyName:    "Acme EOOD",
		ClientName:    "Local Client",
		Number:        "INV-2026-0002",
		Issued:        "1 Oct 2026",
		Period:        "28 Sep 2026 – 4 Oct 2026",
		Due:           "15 Oct 2026",
		Rate:          "€30.00 / h",
		Subtotal:      "€45.00",
		VATLabel:      "VAT 20.00%",
		VATAmount:     "€9.00",
		ShowVATAmount: true,
		Total:         "€54.00",
		ActionURL:     "http://localhost:5173/portal",
		ActionLabel:   "Open portal",
		Lines: []mailer.InvoiceMailLine{
			{ProjectName: "Portal", TaskTitle: "Draw", Hours: "1.50", Amount: "€45.00"},
		},
	})
	require.NoError(t, err)
	require.Contains(t, html, "VAT 20.00%")
	require.Contains(t, html, "€9.00")
	require.Contains(t, html, "€54.00")
}
