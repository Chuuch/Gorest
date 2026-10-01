package mailer_test

import (
	"testing"

	"github.com/chuuch/gorest/internal/mailer"
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
	require.Contains(t, html, "#0f766e")
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
