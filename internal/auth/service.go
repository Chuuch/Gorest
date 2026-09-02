package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/user"
	"github.com/google/uuid"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

type PasswordVerifier interface {
	Compare(password, hash string) error
}

type service struct {
	users           user.Service
	refreshTokens   RefreshTokenRepository
	tokens          TokenManager
	passwords       PasswordVerifier
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewService(
	users user.Service,
	refreshTokens RefreshTokenRepository,
	tokens TokenManager,
	passwords PasswordVerifier,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) Service {
	return &service{
		users:           users,
		refreshTokens:   refreshTokens,
		tokens:          tokens,
		passwords:       passwords,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *service) Register(
	ctx context.Context,
	req RegisterRequest,
) (*AuthResponse, error) {
	exists, err := s.users.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check user email: %w", err)
	}

	if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	u, err := s.users.Create(ctx, user.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueTokens(ctx, u.ID)
}

func (s *service) Login(
	ctx context.Context,
	req LoginRequest,
) (*AuthResponse, error) {
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if err := s.passwords.Compare(req.Password, u.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, u.ID)
}

func (s *service) Refresh(
	ctx context.Context,
	refreshToken string,
) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, ErrInvalidToken
	}

	tokenHash := s.tokens.HashRefreshToken(refreshToken)

	storedToken, err := s.refreshTokens.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return nil, ErrTokenRevoked
	}

	if !time.Now().UTC().Before(storedToken.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	if err := s.refreshTokens.Revoke(ctx, storedToken.ID); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	return s.issueTokens(ctx, storedToken.UserID)
}

func (s *service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if refreshToken == "" {
		return ErrInvalidToken
	}

	tokenHash := s.tokens.HashRefreshToken(refreshToken)

	storedToken, err := s.refreshTokens.GetByHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return ErrTokenRevoked
	}

	if err := s.refreshTokens.Revoke(ctx, storedToken.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *service) issueTokens(
	ctx context.Context,
	userID uuid.UUID,
) (*AuthResponse, error) {
	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("generate access  token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()

	storedToken := &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: s.tokens.HashRefreshToken(refreshToken),
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
	}

	if err := s.refreshTokens.Create(ctx, storedToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
