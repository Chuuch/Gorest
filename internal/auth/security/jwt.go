package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/auth/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type jwtManager struct {
	accessTokenSecret []byte
	issuer            string
	accessTokenTTL    time.Duration
}

type jwtClaims struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	jwt.RegisteredClaims
}

func NewJwtManager(
	accessTokenSecret string,
	issuer string,
	accessTokenTTL time.Duration,
) TokenManager {
	return &jwtManager{
		accessTokenSecret: []byte(accessTokenSecret),
		issuer:            issuer,
		accessTokenTTL:    accessTokenTTL,
	}
}

func (m *jwtManager) GenerateAccessToken(
	userID, organizationID uuid.UUID,
) (string, error) {
	now := time.Now().UTC()

	claims := jwtClaims{
		OrganizationID: organizationID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(m.accessTokenSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}

func (m *jwtManager) ParseAccessToken(
	tokenString string,
) (*AccessTokenClaims, error) {
	var claims jwtClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.accessTokenSecret, nil
	},
		jwt.WithIssuer(m.issuer),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}

		return nil, domain.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	if claims.Subject == "" {
		return nil, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if claims.OrganizationID == uuid.Nil {
		return nil, domain.ErrInvalidToken
	}

	if claims.ExpiresAt == nil {
		return nil, domain.ErrInvalidToken
	}

	if claims.IssuedAt == nil {
		return nil, domain.ErrInvalidToken
	}

	return &AccessTokenClaims{
		UserID:         userID,
		OrganizationID: claims.OrganizationID,
		Issuer:         claims.Issuer,
		ExpiresAt:      claims.ExpiresAt.Time,
		IssuedAt:       claims.IssuedAt.Time,
	}, nil
}

func (m *jwtManager) GenerateRefreshToken() (string, error) {
	const tokenBytes = 32

	buf := make([]byte, tokenBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (m *jwtManager) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}
