package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var transactional = template.Must(
	template.ParseFS(templateFS, "templates/transactional.html"),
)

var invoiceTpl = template.Must(
	template.ParseFS(templateFS, "templates/invoice.html"),
)

type Transactional struct {
	Heading     string
	Body        string
	ActionURL   string
	ActionLabel string
	Expiry      string
}

type InvoiceMailLine struct {
	ProjectName string
	TaskTitle   string
	Hours       string
	Amount      string
}

type InvoiceMail struct {
	Heading     string
	AgencyName  string
	ClientName  string
	Number      string
	Issued      string
	Period      string
	Due         string
	Rate        string
	Total       string
	Lines       []InvoiceMailLine
	ActionURL   string
	ActionLabel string
}

func RenderTransactional(data Transactional) (string, error) {
	var buf bytes.Buffer
	if err := transactional.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render mail: %w", err)
	}
	return buf.String(), nil
}

func RenderInvoice(data InvoiceMail) (string, error) {
	var buf bytes.Buffer
	if err := invoiceTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render invoice mail: %w", err)
	}
	return buf.String(), nil
}
