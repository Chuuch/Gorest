package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	"github.com/chuuch/gorest/internal/auth/repository"
	"github.com/chuuch/gorest/internal/auth/security"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/google/uuid"
)

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User *userdomain.User
}

type Service interface {
	Register(ctx context.Context, req authdomain.RegisterRequest) (*AuthResult, error)
	Login(ctx context.Context, req authdomain.LoginRequest) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (*AuthResult, error)
}

type PasswordVerifier interface {
	Compare(password, hash string) error
}

type service struct {
	users           userusecase.Service
	refreshTokens   repository.RefreshTokenRepository
	tokens          security.TokenManager
	passwords       PasswordVerifier
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewService(
	users userusecase.Service,
	refreshTokens repository.RefreshTokenRepository,
	tokens security.TokenManager,
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
	req authdomain.RegisterRequest,
) (*AuthResult, error) {
	exists, err := s.users.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check user email: %w", err)
	}

	if exists {
		return nil, userdomain.ErrEmailAlreadyExists
	}

	u, err := s.users.Create(ctx, userdomain.CreateUserRequest{
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
	req authdomain.LoginRequest,
) (*AuthResult, error) {
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, userdomain.ErrUserNotFound) {
			return nil, authdomain.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if err := s.passwords.Compare(req.Password, u.PasswordHash); err != nil {
		return nil, authdomain.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, u.ID)
}

func (s *service) Refresh(
	ctx context.Context,
	refreshToken string,
) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, authdomain.ErrInvalidToken
	}

	tokenHash := s.tokens.HashRefreshToken(refreshToken)

	storedToken, err := s.refreshTokens.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return nil, authdomain.ErrTokenRevoked
	}

	if !time.Now().UTC().Before(storedToken.ExpiresAt) {
		return nil, authdomain.ErrTokenExpired
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
		return authdomain.ErrInvalidToken
	}

	tokenHash := s.tokens.HashRefreshToken(refreshToken)

	storedToken, err := s.refreshTokens.GetByHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return authdomain.ErrTokenRevoked
	}

	if err := s.refreshTokens.Revoke(ctx, storedToken.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *service) issueTokens(
	ctx context.Context,
	userID uuid.UUID,
) (*AuthResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()

	storedToken := &authdomain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: s.tokens.HashRefreshToken(refreshToken),
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
	}

	if err := s.refreshTokens.Create(ctx, storedToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: u,
	}, nil
}

func (s *service) Me(
	ctx context.Context,
	userID uuid.UUID,
) (*AuthResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	accessToken, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &AuthResult{
		AccessToken: accessToken,
		User: u,
	}, nil
}
