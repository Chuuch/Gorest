package mailer

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

type SMTPMailer struct {
	from string
	host string
	port int
}

func NewSMTPMailer(from, host string, port int) *SMTPMailer {
	if strings.TrimSpace(host) == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 1025
	}

	return &SMTPMailer{
		from: from,
		host: host,
		port: port,
	}
}

func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}

	fromAddr, err := addressEmail(m.from)
	if err != nil {
		return err
	}

	toAddr, err := addressEmail(msg.To)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	body := buildMIME(m.from, msg)

	if err := smtp.SendMail(addr, nil, fromAddr, []string{toAddr}, body); err != nil {
		return fmt.Errorf("send smtp mail: %w", err)
	}
	return nil
}

func addressEmail(value string) (string, error) {
	parsed, err := mail.ParseAddress(value)
	if err != nil {
		return "", fmt.Errorf("parse mail address %q: %w", value, err)
	}
	return parsed.Address, nil
}

func buildMIME(from string, msg Message) []byte {
	boundary := "gorest-mailer-boundary"
	var b strings.Builder

	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)

	if strings.TrimSpace(msg.Text) != "" {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text)
		b.WriteString("\r\n")
	}
	if strings.TrimSpace(msg.HTML) != "" {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.HTML)
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}
