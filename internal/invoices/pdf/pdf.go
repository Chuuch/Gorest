package pdf

import (
	"bytes"
	"fmt"
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
	doc.CellFormat(0, 7, tr(invoice.OrganizationName), "", 1, "L", false, 0, "")
	doc.Ln(4)

	doc.SetFont("Helvetica", "", 10)
	doc.SetTextColor(93, 110, 102)
	doc.CellFormat(0, 5, "Bill to", "", 1, "L", false, 0, "")
	doc.SetTextColor(20, 32, 27)
	doc.SetFont("Helvetica", "", 12)
	doc.CellFormat(0, 7, tr(invoice.ClientName), "", 1, "L", false, 0, "")
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

	doc.Ln(4)
	doc.SetFont("Helvetica", "B", 12)
	doc.CellFormat(colW[0]+colW[1]+colW[2], 8, "Total", "", 0, "R", false, 0, "")
	doc.CellFormat(colW[3], 8, tr(formatEUR(invoice.TotalCents)), "", 0, "R", false, 0, "")

	if err := doc.Error(); err != nil {
		return nil, fmt.Errorf("build invoice pdf: %w", err)
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, fmt.Errorf("write invoice pdf: %w", err)
	}

	return buf.Bytes(), nil
}

func formatHours(minutes int) string {
	return fmt.Sprintf("%.2f", float64(minutes)/60)
}

func formatEUR(cents int) string {
	return fmt.Sprintf("€%.2f", float64(cents)/100)
}
