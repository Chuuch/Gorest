package auth

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
	DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error
}
