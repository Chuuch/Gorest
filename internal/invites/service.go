package invites

import (
	"context"
	"errors"
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

type SessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type Service interface {
	Issue(ctx context.Context, in IssueInput) error
	Accept(ctx context.Context, token, password string) error
}

type PasswordResets interface {
	RequestReset(ctx context.Context, email string) error
	Reset(ctx context.Context, token, password string) error
}

type MailTokens interface {
	Service
	PasswordResets
}

type service struct {
	tokens    TokenStore
	users     userusecase.Service
	security  security.TokenManager
	mailer    mailer.Mailer
	revoker   SessionRevoker
	db        *pgxpool.Pool
	ttl       time.Duration
	resetTTL  time.Duration
	publicURL string
}

func NewService(
	tokens TokenStore,
	users userusecase.Service,
	security security.TokenManager,
	mailer mailer.Mailer,
	revoker SessionRevoker,
	db *pgxpool.Pool,
	ttl time.Duration,
	resetTTL time.Duration,
	publicURL string,
) MailTokens {
	return &service{
		tokens:    tokens,
		users:     users,
		security:  security,
		mailer:    mailer,
		revoker:   revoker,
		db:        db,
		ttl:       ttl,
		resetTTL:  resetTTL,
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
	subject, text, html, err := inviteMail(in, acceptURL)
	if err != nil {
		return fmt.Errorf("render invite: %w", err)
	}

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

func (s *service) RequestReset(ctx context.Context, email string) error {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, userdomain.ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("get user by email: %w", err)
	}

	raw, err := s.security.GenerateRefreshToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	now := time.Now().UTC()

	if err := s.tokens.Create(ctx, &Token{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: s.security.HashRefreshToken(raw),
		Purpose:   PurposeReset,
		ExpiresAt: now.Add(s.resetTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}

	resetURL := s.publicURL + "/reset-password?token=" + raw
	subject := "Reset your Flourish password"
	text := "Reset your password:\n" + resetURL + "\n\nThis link expires in 1 hour.\n"
	html, err := mailer.RenderTransactional(mailer.Transactional{
		Heading:     "Reset your password",
		Body:        "Use the button below to choose a new password for your Flourish account.",
		ActionURL:   resetURL,
		ActionLabel: "Reset your password",
		Expiry:      "This link expires in 1 hour.",
	})
	if err != nil {
		return fmt.Errorf("render reset: %w", err)
	}

	if err := s.mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: subject,
		Text:    text,
		HTML:    html,
	}); err != nil {
		return fmt.Errorf("send reset: %w", err)
	}

	return nil
}

func (s *service) Reset(ctx context.Context, rawToken, password string) error {
	if rawToken == "" {
		return ErrResetNotFound
	}

	var userID uuid.UUID

	err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		stored, err := s.tokens.GetByHash(ctx, s.security.HashRefreshToken(rawToken))
		if err != nil {
			if errors.Is(err, ErrInviteNotFound) {
				return ErrResetNotFound
			}
			return err
		}

		if stored.Purpose != PurposeReset {
			return ErrResetNotFound
		}

		if stored.UsedAt != nil {
			return ErrResetUsed
		}

		if !time.Now().UTC().Before(stored.ExpiresAt) {
			return ErrResetExpired
		}

		user, err := s.users.GetByID(ctx, stored.UserID)
		if err != nil {
			return fmt.Errorf("get reset user: %w", err)
		}

		if _, err := s.users.Update(ctx, user.ID, userdomain.UpdateUserRequest{
			Email:    user.Email,
			Password: &password,
		}); err != nil {
			return fmt.Errorf("set reset password: %w", err)
		}

		if err := s.tokens.MarkUsed(ctx, stored.ID); err != nil {
			if errors.Is(err, ErrInviteUsed) {
				return ErrResetUsed
			}
			return err
		}
		userID = user.ID
		return nil
	})
	if err != nil {
		return err
	}

	if s.revoker != nil {
		if err := s.revoker.RevokeAllForUser(ctx, userID); err != nil {
			return fmt.Errorf("revoke refresh tokens: %w", err)
		}
	}
	return nil
}

func inviteMail(in IssueInput, acceptURL string) (string, string, string, error) {
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

	html, err := mailer.RenderTransactional(mailer.Transactional{
		Heading:     "You're invited",
		Body:        "You've been invited to " + who + ".",
		ActionURL:   acceptURL,
		ActionLabel: "Set your password",
		Expiry:      "This link expires in 7 days.",
	})
	if err != nil {
		return "", "", "", err
	}

	return subject, text, html, nil
}
