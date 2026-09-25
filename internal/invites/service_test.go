package invites_test

import (
	"context"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/chuuch/gorest/internal/invites"
	"github.com/chuuch/gorest/internal/mailer"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testBcryptCost     = 4
	testAccessSecret   = "test-access-token-secret"
	testIssuer         = "gorest-test"
	testAccessTokenTTL = 15 * time.Minute
	testInviteTTL      = 7 * 24 * time.Hour
	testPublicURL      = "http://localhost:5173"
	testPassword       = "password123"
)

type recordingMailer struct {
	messages []mailer.Message
}

func (m *recordingMailer) Send(_ context.Context, msg mailer.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func setupInviteTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gorest_test"),
		tcpostgres.WithUsername("gorest"),
		tcpostgres.WithPassword("gorest"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping(ctx))

	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE auth_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			purpose TEXT NOT NULL CHECK (purpose IN ('invite')),
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

type inviteTestDependencies struct {
	service   invites.Service
	users     userusecase.Service
	mailer    *recordingMailer
	tokens    security.TokenManager
	passwords *password.BcryptHasher
}

func setupInviteService(t *testing.T, db *pgxpool.Pool, ttl time.Duration) inviteTestDependencies {
	t.Helper()

	hasher := password.NewBcryptHasher(testBcryptCost)
	users := userusecase.NewService(userpostgres.NewRepository(db), hasher)
	tokens := security.NewJwtManager(testAccessSecret, testIssuer, testAccessTokenTTL)
	mail := &recordingMailer{}

	return inviteTestDependencies{
		service: invites.NewService(
			invites.NewRepository(db),
			users,
			tokens,
			mail,
			db,
			ttl,
			testPublicURL,
		),
		users:     users,
		mailer:    mail,
		tokens:    tokens,
		passwords: hasher,
	}
}

func TestInviteService_IssueAndAccept(t *testing.T) {
	db, cleanup := setupInviteTestDatabase(t)
	defer cleanup()

	deps := setupInviteService(t, db, testInviteTTL)
	ctx := context.Background()

	user, err := deps.users.Create(ctx, userdomain.CreateUserRequest{
		Email:    "ada@example.com",
		Password: "discarded-password",
	})
	require.NoError(t, err)

	require.Error(t, deps.passwords.Compare(testPassword, user.PasswordHash))

	err = deps.service.Issue(ctx, invites.IssueInput{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationName: "Acme",
		Kind:             invites.KindStaff,
	})
	require.NoError(t, err)
	require.Len(t, deps.mailer.messages, 1)
	require.Equal(t, "ada@example.com", deps.mailer.messages[0].To)
	require.Contains(t, deps.mailer.messages[0].Subject, "Acme")
	require.Contains(t, deps.mailer.messages[0].Text, testPublicURL+"/accept-invite?token=")

	raw := inviteTokenFromMail(t, deps.mailer.messages[0].Text)

	err = deps.service.Accept(ctx, raw, testPassword)
	require.NoError(t, err)

	updated, err := deps.users.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NoError(t, deps.passwords.Compare(testPassword, updated.PasswordHash))

	err = deps.service.Accept(ctx, raw, "another-password")
	require.ErrorIs(t, err, invites.ErrInviteUsed)
}

func TestInviteService_Accept_InvalidToken(t *testing.T) {
	db, cleanup := setupInviteTestDatabase(t)
	defer cleanup()

	deps := setupInviteService(t, db, testInviteTTL)

	err := deps.service.Accept(context.Background(), "not-a-token", testPassword)
	require.ErrorIs(t, err, invites.ErrInviteNotFound)
}

func TestInviteService_Accept_Expired(t *testing.T) {
	db, cleanup := setupInviteTestDatabase(t)
	defer cleanup()

	deps := setupInviteService(t, db, time.Millisecond)
	ctx := context.Background()

	user, err := deps.users.Create(ctx, userdomain.CreateUserRequest{
		Email:    "ada@example.com",
		Password: "discarded-password",
	})
	require.NoError(t, err)

	err = deps.service.Issue(ctx, invites.IssueInput{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationName: "Acme",
		Kind:             invites.KindPortal,
		ClientName:       "Northwind",
	})
	require.NoError(t, err)
	require.Contains(t, deps.mailer.messages[0].Text, "Northwind")

	time.Sleep(5 * time.Millisecond)

	raw := inviteTokenFromMail(t, deps.mailer.messages[0].Text)
	err = deps.service.Accept(ctx, raw, testPassword)
	require.ErrorIs(t, err, invites.ErrInviteExpired)
}

func inviteTokenFromMail(t *testing.T, text string) string {
	t.Helper()

	const prefix = "/accept-invite?token="
	idx := -1
	for i := 0; i+len(prefix) <= len(text); i++ {
		if text[i:i+len(prefix)] == prefix {
			idx = i + len(prefix)
			break
		}
	}
	require.Greater(t, idx, 0)

	end := idx
	for end < len(text) && text[end] != '\n' {
		end++
	}

	return text[idx:end]
}
