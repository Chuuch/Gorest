package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/chuuch/gorest/internal/middleware"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	middlewareTestSecret = "test-secret"
	middlewareTestIssuer = "gorest-test"
	middlewareTestTTL    = 15 * time.Minute
)

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	tokenManager := security.NewJwtManager(
		middlewareTestSecret,
		middlewareTestIssuer,
		middlewareTestTTL,
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := middleware.Auth(tokenManager)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuth_InvalidAuthorizationHeader(t *testing.T) {
	tokenManager := security.NewJwtManager(
		middlewareTestSecret,
		middlewareTestIssuer,
		middlewareTestTTL,
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := middleware.Auth(tokenManager)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	req.Header.Set("Authorization", "Basic some-token")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	tokenManager := security.NewJwtManager(
		middlewareTestSecret,
		middlewareTestIssuer,
		middlewareTestTTL,
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	handler := middleware.Auth(tokenManager)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	req.Header.Set("Authorization", "Bearer invalid-token")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	tokenManager := security.NewJwtManager(
		middlewareTestSecret,
		middlewareTestIssuer,
		middlewareTestTTL,
	)

	userID := uuid.New()
	organizationID := uuid.New()

	token, err := tokenManager.GenerateAccessToken(userID, organizationID, "owner")
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUserID, ok := requestcontext.UserID(r.Context())

		require.True(t, ok)
		require.Equal(t, userID, authenticatedUserID)

		authenticatedOrganizationID, ok := requestcontext.OrganizationID(r.Context())

		require.True(t, ok)
		require.Equal(t, organizationID, authenticatedOrganizationID)

		role, ok := requestcontext.Role(r.Context())

		require.True(t, ok)
		require.Equal(t, "owner", role)

		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Auth(tokenManager)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}
