package usecase_test

import (
	"context"
	"testing"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/stretchr/testify/require"
)

func TestAuthService_ChangePassword(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	registered, err := deps.service.Register(ctx, testRegisterRequest())
	require.NoError(t, err)

	result, err := deps.service.ChangePassword(ctx, registered.User.ID, authdomain.ChangePasswordRequest{
		CurrentPassword: testPassword,
		Password:        "newpassword",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.RefreshToken)
	require.NotEqual(t, registered.RefreshToken, result.RefreshToken)

	oldToken, err := deps.refreshTokens.GetByHash(
		ctx,
		deps.tokens.HashRefreshToken(registered.RefreshToken),
	)
	require.NoError(t, err)
	require.NotNil(t, oldToken.RevokedAt)

	updated, err := deps.users.GetByID(ctx, registered.User.ID)
	require.NoError(t, err)
	require.NoError(t, password.NewBcryptHasher(testBcryptCost).Compare("newpassword", updated.PasswordHash))

	_, err = deps.service.Login(ctx, authdomain.LoginRequest{
		Email:    "john@example.com",
		Password: testPassword,
	})
	require.ErrorIs(t, err, authdomain.ErrInvalidCredentials)

	_, err = deps.service.Login(ctx, authdomain.LoginRequest{
		Email:    "john@example.com",
		Password: "newpassword",
	})
	require.NoError(t, err)
}

func TestAuthService_ChangePassword_WrongPassword(t *testing.T) {
	db, cleanup := setupAuthTestDatabase(t)
	defer cleanup()

	deps := setupAuthService(t, db)
	ctx := context.Background()

	registered, err := deps.service.Register(ctx, testRegisterRequest())
	require.NoError(t, err)

	_, err = deps.service.ChangePassword(ctx, registered.User.ID, authdomain.ChangePasswordRequest{
		CurrentPassword: "wrong-password",
		Password:        "newpassword",
	})
	require.ErrorIs(t, err, authdomain.ErrInvalidCredentials)
}
