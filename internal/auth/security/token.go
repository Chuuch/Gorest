package security

import (
	"time"

	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID) (string, error)
	ParseAccessToken(token string) (*AccessTokenClaims, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}

type AccessTokenClaims struct {
	UserID    uuid.UUID
	Issuer    string
	ExpiresAt time.Time
	IssuedAt  time.Time
}
