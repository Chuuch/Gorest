package mailer_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chuuch/gorest/internal/config"
	"github.com/chuuch/gorest/internal/mailer"
	"github.com/stretchr/testify/require"
)

func TestNew_Log(t *testing.T) {
	sender, err := mailer.New(config.MailerConfig{
		Driver: "log",
		From:   "Flourish <noreply@localhost>",
	})
	require.NoError(t, err)

	err = sender.Send(context.Background(), mailer.Message{
		To:      "ada@example.com",
		Subject: "Invite to Acme",
		Text:    "Set your password",
		HTML:    "<p>Set your password</p>",
	})
	require.NoError(t, err)
}

func TestNew_UnknownDriver(t *testing.T) {
	_, err := mailer.New(config.MailerConfig{
		Driver: "ses",
		From:   "Flourish <noreply@localhost>",
	})
	require.Error(t, err)
}

func TestNew_ResendRequiresAPIKey(t *testing.T) {
	_, err := mailer.New(config.MailerConfig{
		Driver: "resend",
		From:   "Flourish <noreply@localhost>",
	})
	require.Error(t, err)
}

func TestSend_MissingTo(t *testing.T) {
	sender := mailer.NewLogMailer("Flourish <noreply@localhost>")

	err := sender.Send(context.Background(), mailer.Message{
		Subject: "Invite to Acme",
		Text:    "Set your password",
	})
	require.Error(t, err)
}

func TestResendMailer_Send(t *testing.T) {
	var gotAuth string
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &payload))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_1"}`))
	}))
	defer server.Close()

	sender, err := mailer.New(config.MailerConfig{
		Driver: "resend",
		From:   "Flourish <noreply@localhost>",
		APIKey: "re_test",
		APIURL: server.URL,
	})
	require.NoError(t, err)

	err = sender.Send(context.Background(), mailer.Message{
		To:      "ada@example.com",
		Subject: "Invite to Acme",
		Text:    "Set your password",
		HTML:    "<p>Set your password</p>",
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer re_test", gotAuth)
	require.Equal(t, "Flourish <noreply@localhost>", payload["from"])
	require.Equal(t, []any{"ada@example.com"}, payload["to"])
	require.Equal(t, "Invite to Acme", payload["subject"])
	require.Equal(t, "Set your password", payload["text"])
	require.Equal(t, "<p>Set your password</p>", payload["html"])
}

func TestResendMailer_Send_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer server.Close()

	sender := mailer.NewResendMailer(
		"Flourish <noreply@localhost>",
		"re_bad",
		server.URL,
	)

	err := sender.Send(context.Background(), mailer.Message{
		To:      "ada@example.com",
		Subject: "Invite to Acme",
		Text:    "Set your password",
	})
	require.Error(t, err)
}
