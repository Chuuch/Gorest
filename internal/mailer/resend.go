package mailer

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ResendMailer struct {
	from string
	apiKey string
	apiURL string
	client *http.Client
}

type resendRequest struct {
	From string `json:"from"`
	To []string `json:"to"`
	Subject string `json:"subject"`
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

func NewResendMailer(from, apiKey, apiURL string) *ResendMailer {
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.resend.com/emails"
	}

	return &ResendMailer{
		from: from,
		apiKey: apiKey,
		apiURL: apiURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *ResendMailer) Send(ctx context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}

	payload, err := json.Marshal(resendRequest{
		From: m.from,
		To: []string{msg.To},
		Subject: msg.Subject,
		Text: msg.Text,
		HTML: msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("marshal resend pauload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend mail: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 {
		return fmt.Errorf("resend mail failed: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
