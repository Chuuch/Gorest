package auth

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chuuch/gorest/internal/user"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	registerFunc func(RegisterRequest) (*AuthResponse, error)
	loginFunc    func(LoginRequest) (*AuthResponse, error)
	refreshFunc  func(string) (*AuthResponse, error)
	logoutFunc   func(string) error
}

func (m *mockService) Register(
	_ context.Context,
	req RegisterRequest,
) (*AuthResponse, error) {
	return m.registerFunc(req)
}

func (m *mockService) Login(
	_ context.Context,
	req LoginRequest,
) (*AuthResponse, error) {
	return m.loginFunc(req)
}

func (m *mockService) Refresh(
	_ context.Context,
	refreshToken string,
) (*AuthResponse, error) {
	return m.refreshFunc(refreshToken)
}

func (m *mockService) Logout(
	_ context.Context,
	refreshToken string,
) error {
	return m.logoutFunc(refreshToken)
}

func TestHandler_Register(t *testing.T) {
	service := &mockService{
		registerFunc: func(req RegisterRequest) (*AuthResponse, error) {
			require.Equal(t, "john@example.com", req.Email)
			require.Equal(t, "password123", req.Password)

			return &AuthResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "access-token", response.AccessToken)
	require.Equal(t, "refresh-token", response.RefreshToken)
}

func TestHandler_Register_InvalidBody(t *testing.T) {
	service := &mockService{
		registerFunc: func(RegisterRequest) (*AuthResponse, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(`invalid-json`),
	)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Register_EmailAlreadyExists(t *testing.T) {
	service := &mockService{
		registerFunc: func(RegisterRequest) (*AuthResponse, error) {
			return nil, user.ErrEmailAlreadyExists
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Register_InternalError(t *testing.T) {
	service := &mockService{
		registerFunc: func(RegisterRequest) (*AuthResponse, error) {
			return nil, errors.New("database failure")
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_Login(t *testing.T) {
	service := &mockService{
		loginFunc: func(req LoginRequest) (*AuthResponse, error) {
			require.Equal(t, "john@example.com", req.Email)
			require.Equal(t, "password123", req.Password)

			return &AuthResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "password123"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "access-token", response.AccessToken)
	require.Equal(t, "refresh-token", response.RefreshToken)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	service := &mockService{
		loginFunc: func(LoginRequest) (*AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "wrong-password"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Login_InvalidBody(t *testing.T) {
	service := &mockService{
		loginFunc: func(LoginRequest) (*AuthResponse, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`invalid-json`),
	)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Refresh(t *testing.T) {
	service := &mockService{
		refreshFunc: func(refreshToken string) (*AuthResponse, error) {
			require.Equal(t, "refresh-token", refreshToken)

			return &AuthResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
			}, nil
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "refresh-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "new-access-token", response.AccessToken)
	require.Equal(t, "new-refresh-token", response.RefreshToken)
}

func TestHandler_Refresh_InvalidToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*AuthResponse, error) {
			return nil, ErrInvalidToken
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "invalid-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Refresh_ExpiredToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*AuthResponse, error) {
			return nil, ErrTokenExpired
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "expired-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Refresh_RevokedToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*AuthResponse, error) {
			return nil, ErrTokenRevoked
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "revoked-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout(t *testing.T) {
	service := &mockService{
		logoutFunc: func(refreshToken string) error {
			require.Equal(t, "refresh-token", refreshToken)
			return nil
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "refresh-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestHandler_Logout_InvalidToken(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			return ErrInvalidToken
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "invalid-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout_RevokedToken(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			return ErrTokenRevoked
		},
	}

	handler := NewHandler(service)

	body := `{
		"refresh_token": "revoked-token"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout_InvalidBody(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			t.Fatal("service should not be called")
			return nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		bytes.NewBufferString(`invalid-json`),
	)

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Refresh_InvalidBody(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*AuthResponse, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		bytes.NewBufferString(`invalid-json`),
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_ErrorWrapping(t *testing.T) {
	service := &mockService{
		loginFunc: func(LoginRequest) (*AuthResponse, error) {
			return nil, errors.Join(
				errors.New("some context"),
				ErrInvalidCredentials,
			)
		},
	}

	handler := NewHandler(service)

	body := `{
		"email": "john@example.com",
		"password": "wrong-password"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
