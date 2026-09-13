package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth/domain"
	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testAuthUser() *userdomain.User {
	return &userdomain.User{
		ID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Email: "john@example.com",
	}
}

func testAuthOrg() *orgdomain.Organization {
	return &orgdomain.Organization{
		ID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name: "Acme",
	}
}

func testAuthResult(accessToken, refreshToken string) *authusecase.AuthResult {
	return &authusecase.AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         testAuthUser(),
		Organization: testAuthOrg(),
		Role:         orgdomain.RoleOwner,
	}
}

type mockService struct {
	registerFunc func(domain.RegisterRequest) (*authusecase.AuthResult, error)
	loginFunc    func(domain.LoginRequest) (*authusecase.AuthResult, error)
	refreshFunc  func(string) (*authusecase.AuthResult, error)
	logoutFunc   func(string) error
	meFunc       func(uuid.UUID) (*authusecase.AuthResult, error)
}

func (m *mockService) Register(
	_ context.Context,
	req domain.RegisterRequest,
) (*authusecase.AuthResult, error) {
	return m.registerFunc(req)
}

func (m *mockService) Login(
	_ context.Context,
	req domain.LoginRequest,
) (*authusecase.AuthResult, error) {
	return m.loginFunc(req)
}

func (m *mockService) Refresh(
	_ context.Context,
	refreshToken string,
) (*authusecase.AuthResult, error) {
	return m.refreshFunc(refreshToken)
}

func (m *mockService) Logout(
	_ context.Context,
	refreshToken string,
) error {
	return m.logoutFunc(refreshToken)
}

func (m *mockService) Me(
	_ context.Context,
	userID uuid.UUID,
) (*authusecase.AuthResult, error) {
	return m.meFunc(userID)
}

func TestHandler_Register(t *testing.T) {
	service := &mockService{
		registerFunc: func(req domain.RegisterRequest) (*authusecase.AuthResult, error) {
			require.Equal(t, "john@example.com", req.Email)
			require.Equal(t, "password123", req.Password)
			require.Equal(t, "Acme", req.OrganizationName)

			return testAuthResult("access-token", "refresh-token"), nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	body := `{
        "email": "john@example.com",
        "password": "password123",
        "organization_name": "Acme"
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

	var response domain.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "access-token", response.AccessToken)
	require.Equal(t, "john@example.com", response.User.Email)
	require.Equal(t, "Acme", response.Organization.Name)
	require.Equal(t, "owner", response.Role)

	cookie := rec.Result().Cookies()[0]
	require.Equal(t, "refresh_token", cookie.Name)
	require.Equal(t, "refresh-token", cookie.Value)
	require.True(t, cookie.HttpOnly)
	require.True(t, cookie.Secure)
	require.Equal(t, "/api/v1/auth", cookie.Path)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestHandler_Register_InvalidBody(t *testing.T) {
	service := &mockService{
		registerFunc: func(domain.RegisterRequest) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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
		registerFunc: func(domain.RegisterRequest) (*authusecase.AuthResult, error) {
			return nil, userdomain.ErrEmailAlreadyExists
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	body := `{
        "email": "john@example.com",
        "password": "password123",
        "organization_name": "Acme"
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
		registerFunc: func(domain.RegisterRequest) (*authusecase.AuthResult, error) {
			return nil, errors.New("database failure")
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	body := `{
        "email": "john@example.com",
        "password": "password123",
        "organization_name": "Acme"
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
		loginFunc: func(req domain.LoginRequest) (*authusecase.AuthResult, error) {
			require.Equal(t, "john@example.com", req.Email)
			require.Equal(t, "password123", req.Password)

			return testAuthResult("access-token", "refresh-token"), nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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

	var response domain.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "access-token", response.AccessToken)
	require.Equal(t, "john@example.com", response.User.Email)
	require.Equal(t, "Acme", response.Organization.Name)
	require.Equal(t, "owner", response.Role)

	cookie := rec.Result().Cookies()[0]
	require.Equal(t, "refresh_token", cookie.Name)
	require.Equal(t, "refresh-token", cookie.Value)
	require.True(t, cookie.HttpOnly)
	require.True(t, cookie.Secure)
	require.Equal(t, "/api/v1/auth", cookie.Path)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	service := &mockService{
		loginFunc: func(domain.LoginRequest) (*authusecase.AuthResult, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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

func TestHandler_Login_NoOrganization(t *testing.T) {
	service := &mockService{
		loginFunc: func(domain.LoginRequest) (*authusecase.AuthResult, error) {
			return nil, domain.ErrNoOrganization
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Login_InvalidBody(t *testing.T) {
	service := &mockService{
		loginFunc: func(domain.LoginRequest) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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
		refreshFunc: func(refreshToken string) (*authusecase.AuthResult, error) {
			require.Equal(t, "refresh-token", refreshToken)

			return testAuthResult("new-access-token", "new-refresh-token"), nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "refresh-token",
	})

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response domain.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "new-access-token", response.AccessToken)
	require.Equal(t, "john@example.com", response.User.Email)
	require.Equal(t, "Acme", response.Organization.Name)
	require.Equal(t, "owner", response.Role)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "refresh_token", cookies[0].Name)
	require.Equal(t, "new-refresh-token", cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.True(t, cookies[0].Secure)
}

func TestHandler_Refresh_NoCookie(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Refresh_InvalidToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*authusecase.AuthResult, error) {
			return nil, domain.ErrInvalidToken
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token",
	})

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Refresh_ExpiredToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*authusecase.AuthResult, error) {
			return nil, domain.ErrTokenExpired
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "expired-token",
	})

	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Refresh_RevokedToken(t *testing.T) {
	service := &mockService{
		refreshFunc: func(string) (*authusecase.AuthResult, error) {
			return nil, domain.ErrTokenRevoked
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/refresh",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "revoked-token",
	})

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

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "refresh-token",
	})

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "refresh_token", cookies[0].Name)
	require.Equal(t, -1, cookies[0].MaxAge)
	require.True(t, cookies[0].HttpOnly)
	require.True(t, cookies[0].Secure)
}

func TestHandler_Logout_InvalidToken(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			return domain.ErrInvalidToken
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid-token",
	})

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout_RevokedToken(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			return domain.ErrTokenRevoked
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "revoked-token",
	})

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Logout_NoCookie(t *testing.T) {
	service := &mockService{
		logoutFunc: func(string) error {
			t.Fatal("service should not be called")
			return nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "refresh_token", cookies[0].Name)
	require.Equal(t, -1, cookies[0].MaxAge)
}

func TestHandler_Me(t *testing.T) {
	user := testAuthUser()

	service := &mockService{
		meFunc: func(userID uuid.UUID) (*authusecase.AuthResult, error) {
			require.Equal(t, user.ID, userID)

			return testAuthResult("access-token", ""), nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), user.ID))

	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response domain.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "access-token", response.AccessToken)
	require.Equal(t, "john@example.com", response.User.Email)
	require.Equal(t, "Acme", response.Organization.Name)
	require.Equal(t, "owner", response.Role)
	require.Empty(t, rec.Result().Cookies())
}

func TestHandler_Me_Unauthorized(t *testing.T) {
	service := &mockService{
		meFunc: func(uuid.UUID) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ErrorWrapping(t *testing.T) {
	service := &mockService{
		loginFunc: func(domain.LoginRequest) (*authusecase.AuthResult, error) {
			return nil, errors.Join(
				errors.New("some context"),
				domain.ErrInvalidCredentials,
			)
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

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

func TestHandler_Register_ValidationError(t *testing.T) {
	service := &mockService{
		registerFunc: func(domain.RegisterRequest) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	body := `{
        "email": "not-an-email",
        "password": "short",
        "organization_name": "Acme"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var response struct {
		Error struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Details map[string]string `json:"details"`
		} `json:"error"`
	}

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "validation_error", response.Error.Code)
	require.Equal(t, "request validation failed", response.Error.Message)
	require.Equal(t, "must be a valid email address", response.Error.Details["Email"])
	require.Equal(t, "must be at least 8", response.Error.Details["Password"])
}

func TestHandler_Login_ValidationError(t *testing.T) {
	service := &mockService{
		loginFunc: func(domain.LoginRequest) (*authusecase.AuthResult, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true)

	body := `{
        "email": "not-an-email",
        "password": ""
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var response struct {
		Error struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Details map[string]string `json:"details"`
		} `json:"error"`
	}

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, "validation_error", response.Error.Code)
	require.Equal(t, "request validation failed", response.Error.Message)
	require.Equal(t, "must be a valid email address", response.Error.Details["Email"])
	require.Equal(t, "is required", response.Error.Details["Password"])
}
