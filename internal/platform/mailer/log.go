package mailer

import (
	"context"
	"log/slog"
)

type LogMailer struct {
	from string
}

func NewLogMailer(from string) *LogMailer {
	return &LogMailer{from: from}
}

func (m *LogMailer) Send(_ context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}

	slog.Info(
		"mailer send",
		"driver", "log",
		"from", m.from,
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}
