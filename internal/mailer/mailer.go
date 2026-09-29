package mailer

import (
	"context"
	"fmt"
	"strings"

	"github.com/chuuch/gorest/internal/config"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

func New(cfg config.MailerConfig) (Mailer, error) {
	from := strings.TrimSpace(cfg.From)
	if from == "" {
		return nil, fmt.Errorf("mailer from address is required")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "log":
		return NewLogMailer(from), nil

	case "mailpit":
		return NewSMTPMailer(from, cfg.SMTPHost, cfg.SMTPPort), nil

	case "resend":
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("mailer api key is required for resend")
		}
		return NewResendMailer(from, cfg.APIKey, cfg.APIURL), nil

	default:
		return nil, fmt.Errorf("unknown mailer driver %q", cfg.Driver)
	}
}

func (m Message) validate() error {
	if strings.TrimSpace(m.To) == "" {
		return fmt.Errorf("mailer to address is required")
	}
	if strings.TrimSpace(m.Subject) == "" {
		return fmt.Errorf("mailer subject is required")
	}
	if strings.TrimSpace(m.Text) == "" && strings.TrimSpace(m.HTML) == "" {
		return fmt.Errorf("mailer body is required")
	}
	return nil
}
