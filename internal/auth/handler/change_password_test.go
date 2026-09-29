package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/auth/domain"
	authhandler "github.com/chuuch/gorest/internal/auth/handler"
	authusecase "github.com/chuuch/gorest/internal/auth/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (m *mockService) ChangePassword(
	_ context.Context,
	userID uuid.UUID,
	req domain.ChangePasswordRequest,
) (*authusecase.AuthResult, error) {
	return nil, nil
}

type changePasswordService struct {
	mockService
	fn func(uuid.UUID, domain.ChangePasswordRequest) (*authusecase.AuthResult, error)
}

func (s *changePasswordService) ChangePassword(
	_ context.Context,
	userID uuid.UUID,
	req domain.ChangePasswordRequest,
) (*authusecase.AuthResult, error) {
	return s.fn(userID, req)
}

func TestHandler_ChangePassword(t *testing.T) {
	user := testAuthUser()

	service := &changePasswordService{
		fn: func(userID uuid.UUID, req domain.ChangePasswordRequest) (*authusecase.AuthResult, error) {
			require.Equal(t, user.ID, userID)
			require.Equal(t, "password123", req.CurrentPassword)
			require.Equal(t, "newpassword", req.Password)
			return testAuthResult("access-token", "new-refresh"), nil
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true, &stubInviter{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/change-password",
		bytes.NewBufferString(`{"current_password":"password123","password":"newpassword"}`),
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), user.ID))

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	cookie := rec.Result().Cookies()[0]
	require.Equal(t, "refresh_token", cookie.Name)
	require.Equal(t, "new-refresh", cookie.Value)
	require.Equal(t, "/api/v1/auth", cookie.Path)
}

func TestHandler_ChangePassword_Unauthorized(t *testing.T) {
	handler := authhandler.NewHandler(&mockService{}, 30*24*time.Hour, true, &stubInviter{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/change-password",
		bytes.NewBufferString(`{"current_password":"password123","password":"newpassword"}`),
	)

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ChangePassword_WrongPassword(t *testing.T) {
	user := testAuthUser()

	service := &changePasswordService{
		fn: func(uuid.UUID, domain.ChangePasswordRequest) (*authusecase.AuthResult, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}

	handler := authhandler.NewHandler(service, 30*24*time.Hour, true, &stubInviter{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/change-password",
		bytes.NewBufferString(`{"current_password":"wrong-password","password":"newpassword"}`),
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), user.ID))

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ChangePassword_InvalidBody(t *testing.T) {
	user := testAuthUser()
	handler := authhandler.NewHandler(&mockService{}, 30*24*time.Hour, true, &stubInviter{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/change-password",
		bytes.NewBufferString(`{"current_password":"password123","password":"short"}`),
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), user.ID))

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
