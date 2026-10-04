package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"
	"github.com/chuuch/gorest/internal/invoices/domain"
)

func Generate(invoice *domain.Invoice) ([]byte, error) {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(18, 18, 18)
	doc.AddPage()
	tr := doc.UnicodeTranslatorFromDescriptor("")

	doc.SetTextColor(20, 32, 27)
	doc.SetFont("Helvetica", "B", 18)
	doc.CellFormat(0, 10, tr(invoice.Number), "", 1, "L", false, 0, "")

	doc.SetFont("Helvetica", "", 12)
	doc.CellFormat(0, 6, tr(invoice.OrganizationName), "", 1, "L", false, 0, "")
	doc.SetFont("Helvetica", "", 10)
	writeAddress(doc, tr, invoice.SellerAddressLine1, invoice.SellerAddressLine2, invoice.SellerCity, invoice.SellerPostalCode, invoice.SellerCountry)
	if invoice.SellerRegistrationNumber != "" {
		doc.CellFormat(0, 5, tr("Reg. "+invoice.SellerRegistrationNumber), "", 1, "L", false, 0, "")
	}
	if invoice.SellerVATID != "" {
		doc.CellFormat(0, 5, tr("VAT "+invoice.SellerVATID), "", 1, "L", false, 0, "")
	}
	doc.Ln(4)

	doc.SetFont("Helvetica", "", 10)
	doc.SetTextColor(93, 110, 102)
	doc.CellFormat(0, 5, "Bill to", "", 1, "L", false, 0, "")
	doc.SetTextColor(20, 32, 27)
	doc.SetFont("Helvetica", "", 12)
	doc.CellFormat(0, 6, tr(invoice.ClientName), "", 1, "L", false, 0, "")
	doc.SetFont("Helvetica", "", 10)
	writeAddress(doc, tr, invoice.BuyerAddressLine1, invoice.BuyerAddressLine2, invoice.BuyerCity, invoice.BuyerPostalCode, invoice.BuyerCountry)
	if invoice.BuyerVATID != "" {
		doc.CellFormat(0, 5, tr("VAT "+invoice.BuyerVATID), "", 1, "L", false, 0, "")
	}
	doc.Ln(2)

	inclusiveTo := invoice.PeriodTo.Add(-time.Nanosecond)
	doc.SetFont("Helvetica", "", 10)
	doc.SetTextColor(93, 110, 102)
	doc.CellFormat(
		0,
		5,
		tr(fmt.Sprintf(
			"Issued %s · Due %s",
			invoice.IssuedAt.UTC().Format("2 Jan 2006"),
			invoice.DueAt.UTC().Format("2 Jan 2006"),
		)),
		"",
		1,
		"L",
		false,
		0,
		"",
	)
	doc.CellFormat(
		0,
		5,
		tr(fmt.Sprintf(
			"Period %s - %s",
			invoice.PeriodFrom.UTC().Format("2 Jan 2006"),
			inclusiveTo.UTC().Format("2 Jan 2006"),
		)),
		"",
		1,
		"L",
		false,
		0,
		"",
	)
	doc.CellFormat(
		0,
		5,
		tr(fmt.Sprintf("Rate %s / h", formatEUR(invoice.RateCents))),
		"",
		1,
		"L",
		false,
		0,
		"",
	)
	doc.Ln(6)

	colW := []float64{55, 70, 25, 27}
	doc.SetFont("Helvetica", "", 9)
	doc.SetTextColor(93, 110, 102)
	headers := []string{"Project", "Task", "Hours", "Amount"}
	for i, header := range headers {
		align := "L"
		if i >= 2 {
			align = "R"
		}
		doc.CellFormat(colW[i], 7, header, "B", 0, align, false, 0, "")
	}
	doc.Ln(-1)

	doc.SetFont("Helvetica", "", 10)
	doc.SetTextColor(20, 32, 27)
	for _, line := range invoice.Lines {
		doc.CellFormat(colW[0], 8, tr(line.ProjectName), "", 0, "L", false, 0, "")
		doc.CellFormat(colW[1], 8, tr(line.TaskTitle), "", 0, "L", false, 0, "")
		doc.CellFormat(colW[2], 8, formatHours(line.Minutes), "", 0, "R", false, 0, "")
		doc.CellFormat(colW[3], 8, tr(formatEUR(line.AmountCents)), "", 0, "R", false, 0, "")
		doc.Ln(-1)
	}

	labelW := colW[0] + colW[1] + colW[2]
	doc.Ln(4)
	doc.SetFont("Helvetica", "", 11)
	doc.CellFormat(labelW, 7, "Subtotal", "", 0, "R", false, 0, "")
	doc.CellFormat(colW[3], 7, tr(formatEUR(invoice.SubtotalCents)), "", 0, "R", false, 0, "")
	doc.Ln(-1)

	switch invoice.VATRegime {
	case domain.RegimeStandard:
		doc.CellFormat(
			labelW,
			7,
			tr(fmt.Sprintf("VAT %s%%", formatRate(invoice.VATRateBPS))),
			"",
			0,
			"R",
			false,
			0,
			"",
		)
		doc.CellFormat(colW[3], 7, tr(formatEUR(invoice.VATCents)), "", 0, "R", false, 0, "")
		doc.Ln(-1)
	default:
		if note := domain.VATNote(invoice.VATRegime); note != "" {
			doc.SetFont("Helvetica", "", 9)
			doc.SetTextColor(93, 110, 102)
			doc.MultiCell(0, 5, tr(note), "", "R", false)
			doc.SetTextColor(20, 32, 27)
			doc.SetFont("Helvetica", "", 11)
		}
	}

	doc.SetFont("Helvetica", "B", 12)
	doc.CellFormat(labelW, 8, "Total", "", 0, "R", false, 0, "")
	doc.CellFormat(colW[3], 8, tr(formatEUR(invoice.TotalCents)), "", 0, "R", false, 0, "")
	doc.Ln(-1)

	if strings.TrimSpace(invoice.BankIBAN) != "" {
		doc.Ln(6)
		doc.SetFont("Helvetica", "", 10)
		doc.SetTextColor(93, 110, 102)
		doc.CellFormat(0, 5, "Payment", "", 1, "L", false, 0, "")
		doc.SetTextColor(20, 32, 27)
		if invoice.BankName != "" {
			doc.CellFormat(0, 5, tr(invoice.BankName), "", 1, "L", false, 0, "")
		}
		doc.CellFormat(0, 5, tr("IBAN "+invoice.BankIBAN), "", 1, "L", false, 0, "")
		if invoice.BankBIC != "" {
			doc.CellFormat(0, 5, tr("BIC "+invoice.BankBIC), "", 1, "L", false, 0, "")
		}
	}

	if err := doc.Error(); err != nil {
		return nil, fmt.Errorf("build invoice pdf: %w", err)
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, fmt.Errorf("write invoice pdf: %w", err)
	}

	return buf.Bytes(), nil
}

func writeAddress(
	doc *fpdf.Fpdf,
	tr func(string) string,
	line1, line2, city, postal, country string,
) {
	if line1 != "" {
		doc.CellFormat(0, 5, tr(line1), "", 1, "L", false, 0, "")
	}
	if line2 != "" {
		doc.CellFormat(0, 5, tr(line2), "", 1, "L", false, 0, "")
	}
	if cityLine := formatCityLine(city, postal, country); cityLine != "" {
		doc.CellFormat(0, 5, tr(cityLine), "", 1, "L", false, 0, "")
	}
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

func formatHours(minutes int) string {
	return fmt.Sprintf("%.2f", float64(minutes)/60)
}

func formatEUR(cents int) string {
	return fmt.Sprintf("€%.2f", float64(cents)/100)
}

func formatRate(bps int) string {
	return fmt.Sprintf("%.2f", float64(bps)/100)
}
