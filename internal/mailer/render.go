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

type Transactional struct {
	Heading     string
	Body        string
	ActionURL   string
	ActionLabel string
	Expiry      string
}

func RenderTransactional(data Transactional) (string, error) {
	var buf bytes.Buffer
	if err := transactional.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render mail: %w", err)
	}
	return buf.String(), nil
}
