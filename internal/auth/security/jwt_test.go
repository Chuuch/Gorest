package security_test

import (
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth/domain"
	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	jwtTestSecret = "test-secret"
	jwtTestIssuer = "gorest-test"
	jwtTestTTL    = 15 * time.Minute
)

func TestJwtManager_GenerateAccessToken(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	userID := uuid.New()
	organizationID := uuid.New()

	token, err := manager.GenerateAccessToken(userID, organizationID, "owner")

	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := manager.ParseAccessToken(token)

	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, organizationID, claims.OrganizationID)
	require.Equal(t, "owner", claims.Role)
	require.Equal(t, jwtTestIssuer, claims.Issuer)
	require.True(t, claims.IssuedAt.Before(time.Now().UTC()) || claims.IssuedAt.Equal(time.Now().UTC()))
	require.True(t, claims.ExpiresAt.After(time.Now().UTC()))
}

func TestJwtManager_ParseAccessToken_InvalidSignature(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	otherManager := security.NewJwtManager(
		"wrong-secret",
		jwtTestIssuer,
		jwtTestTTL,
	)

	token, err := otherManager.GenerateAccessToken(uuid.New(), uuid.New(), "owner")

	require.NoError(t, err)

	_, err = manager.ParseAccessToken(token)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestJwtManager_ParseAccessToken_WrongIssuer(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	otherManager := security.NewJwtManager(
		jwtTestSecret,
		"wrong-issuer",
		jwtTestTTL,
	)

	token, err := otherManager.GenerateAccessToken(uuid.New(), uuid.New(), "owner")

	require.NoError(t, err)

	_, err = manager.ParseAccessToken(token)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestJwtManager_ParseAccessToken_Expired(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		-1*time.Minute,
	)

	token, err := manager.GenerateAccessToken(uuid.New(), uuid.New(), "owner")

	require.NoError(t, err)

	_, err = manager.ParseAccessToken(token)

	require.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestJwtManager_ParseAccessToken_Malformed(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	_, err := manager.ParseAccessToken("not-a-jwt")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestJwtManager_ParseAccessToken_EmptyRole(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	token, err := manager.GenerateAccessToken(uuid.New(), uuid.New(), "")

	require.NoError(t, err)

	_, err = manager.ParseAccessToken(token)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestJwtManager_GenerateRefreshToken(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	first, err := manager.GenerateRefreshToken()

	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := manager.GenerateRefreshToken()

	require.NoError(t, err)
	require.NotEmpty(t, second)
	require.NotEqual(t, first, second)
}

func TestJwtManager_HashRefreshToken(t *testing.T) {
	manager := security.NewJwtManager(
		jwtTestSecret,
		jwtTestIssuer,
		jwtTestTTL,
	)

	token := "test-refresh-token"

	firstHash := manager.HashRefreshToken(token)
	secondHash := manager.HashRefreshToken(token)

	require.NotEmpty(t, firstHash)
	require.Equal(t, firstHash, secondHash)
	require.NotEqual(t, token, firstHash)
}
