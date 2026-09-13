package security

import (
	"time"

	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, organizationID uuid.UUID, role string) (string, error)
	ParseAccessToken(token string) (*AccessTokenClaims, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}

type AccessTokenClaims struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	Issuer         string
	ExpiresAt      time.Time
	IssuedAt       time.Time
}
