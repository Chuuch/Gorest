package usecase_test

import (
	"context"
	"testing"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	authpostgres "github.com/chuuch/gorest/internal/auth/postgres"
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/stretchr/testify/require"
)

func TestClientUserService_ChangePassword(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	ctx := context.Background()
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	user := seedPortalUser(
		t,
		deps.users,
		db,
		organizationID,
		clientID,
		"pat@northwind.test",
		testPassword,
	)

	loggedIn, err := deps.service.Login(ctx, clientuserdomain.LoginRequest{
		Email:    "pat@northwind.test",
		Password: testPassword,
	})
	require.NoError(t, err)

	result, err := deps.service.ChangePassword(ctx, user.ID, authdomain.ChangePasswordRequest{
		CurrentPassword: testPassword,
		Password:        "newpassword",
	})
	require.NoError(t, err)
	require.NotEqual(t, loggedIn.RefreshToken, result.RefreshToken)

	oldToken, err := authpostgres.NewRepository(db).GetByHash(
		ctx,
		deps.tokens.HashRefreshToken(loggedIn.RefreshToken),
	)
	require.NoError(t, err)
	require.NotNil(t, oldToken.RevokedAt)

	updated, err := deps.users.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.NoError(t, password.NewBcryptHasher(testBcryptCost).Compare("newpassword", updated.PasswordHash))
}

func TestClientUserService_ChangePassword_WrongPassword(t *testing.T) {
	db, cleanup := setupClientUserTestDatabase(t)
	defer cleanup()

	deps := setupClientUserService(t, db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	user := seedPortalUser(
		t,
		deps.users,
		db,
		organizationID,
		clientID,
		"pat@northwind.test",
		testPassword,
	)

	_, err := deps.service.ChangePassword(
		context.Background(),
		user.ID,
		authdomain.ChangePasswordRequest{
			CurrentPassword: "wrong-password",
			Password:        "newpassword",
		},
	)
	require.ErrorIs(t, err, authdomain.ErrInvalidCredentials)
}
