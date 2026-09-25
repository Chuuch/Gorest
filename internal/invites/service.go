package invites

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/mailer"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenStore interface {
	Create(ctx context.Context, token *Token) error
	GetByHash(ctx context.Context, tokenHash string) (*Token, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
}

type Service interface {
	Issue(ctx context.Context, in IssueInput) error
	Accept(ctx context.Context, token, password string) error
}

type service struct {
	tokens    TokenStore
	users     userusecase.Service
	security  security.TokenManager
	mailer    mailer.Mailer
	db        *pgxpool.Pool
	ttl       time.Duration
	publicURL string
}

func NewService(
	tokens TokenStore,
	users userusecase.Service,
	security security.TokenManager,
	mailer mailer.Mailer,
	db *pgxpool.Pool,
	ttl time.Duration,
	publicURL string,
) Service {
	return &service{
		tokens:    tokens,
		users:     users,
		security:  security,
		mailer:    mailer,
		db:        db,
		ttl:       ttl,
		publicURL: strings.TrimRight(publicURL, "/"),
	}
}

func (s *service) Issue(ctx context.Context, in IssueInput) error {
	raw, err := s.security.GenerateRefreshToken()
	if err != nil {
		return fmt.Errorf("generate invite token: %w", err)
	}

	now := time.Now().UTC()

	if err := s.tokens.Create(ctx, &Token{
		ID:        uuid.New(),
		UserID:    in.UserID,
		TokenHash: s.security.HashRefreshToken(raw),
		Purpose:   PurposeInvite,
		ExpiresAt: now.Add(s.ttl),
		CreatedAt: now,
	}); err != nil {
		return err
	}

	acceptURL := s.publicURL + "/accept-invite?token=" + raw
	subject, text, html := inviteMail(in, acceptURL)

	if err := s.mailer.Send(ctx, mailer.Message{
		To:      in.Email,
		Subject: subject,
		Text:    text,
		HTML:    html,
	}); err != nil {
		return fmt.Errorf("send invite: %w", err)
	}
	return nil
}

func (s *service) Accept(ctx context.Context, rawToken, password string) error {
	if rawToken == "" {
		return ErrInviteNotFound
	}

	return database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		stored, err := s.tokens.GetByHash(ctx, s.security.HashRefreshToken(rawToken))
		if err != nil {
			return err
		}

		if stored.Purpose != PurposeInvite {
			return ErrInviteNotFound
		}

		if stored.UsedAt != nil {
			return ErrInviteUsed
		}

		if !time.Now().UTC().Before(stored.ExpiresAt) {
			return ErrInviteExpired
		}

		user, err := s.users.GetByID(ctx, stored.UserID)
		if err != nil {
			return fmt.Errorf("get invited user: %w", err)
		}

		if _, err := s.users.Update(ctx, user.ID, userdomain.UpdateUserRequest{
			Email:    user.Email,
			Password: &password,
		}); err != nil {
			return fmt.Errorf("set invite password: %w", err)
		}

		return s.tokens.MarkUsed(ctx, stored.ID)
	})
}

func inviteMail(in IssueInput, acceptURL string) (string, string, string) {
	org := in.OrganizationName
	if org == "" {
		org = "Flourish"
	}

	subject := "You're invited to " + org + " on Flourish"
	who := "join " + org
	if in.Kind == KindPortal {
		client := in.ClientName
		if client == "" {
			client = "a client portal"
		}
		who = "the " + client + " portal at " + org
		subject = "You're invited to " + client + " on Flourish"
	}

	text := "You've been invited to " + who + ".\n\n" +
		"Set your password:\n" + acceptURL + "\n\n" +
		"This link expires in 7 days.\n"

	html := "<p>You've been invited to " + who + ".<p>" +
		"<p><a href=\"" + acceptURL + "\">Set your password</a></p>" +
		"<p>This link expires in 7 days.</p>"

	return subject, text, html
}
