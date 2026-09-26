package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	clientuserhandler "github.com/chuuch/gorest/internal/clientusers/handler"
	clientuserusecase "github.com/chuuch/gorest/internal/clientusers/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (m *mockService) ChangePassword(
	_ context.Context,
	userID uuid.UUID,
	req authdomain.ChangePasswordRequest,
) (*clientuserusecase.AuthResult, error) {
	return nil, nil
}

type changePasswordService struct {
	mockService
	fn func(uuid.UUID, authdomain.ChangePasswordRequest) (*clientuserusecase.AuthResult, error)
}

func (s *changePasswordService) ChangePassword(
	_ context.Context,
	userID uuid.UUID,
	req authdomain.ChangePasswordRequest,
) (*clientuserusecase.AuthResult, error) {
	return s.fn(userID, req)
}

func TestHandler_ChangePassword(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	service := &changePasswordService{
		fn: func(got uuid.UUID, req authdomain.ChangePasswordRequest) (*clientuserusecase.AuthResult, error) {
			require.Equal(t, userID, got)
			require.Equal(t, "password123", req.CurrentPassword)
			require.Equal(t, "newpassword", req.Password)
			return &clientuserusecase.AuthResult{RefreshToken: "new-refresh"}, nil
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/change-password",
		bytes.NewBufferString(`{"current_password":"password123","password":"newpassword"}`),
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), userID))

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	cookie := rec.Result().Cookies()[0]
	require.Equal(t, "refresh_token", cookie.Name)
	require.Equal(t, "new-refresh", cookie.Value)
	require.Equal(t, "/api/v1/client-auth", cookie.Path)
}

func TestHandler_ChangePassword_Unauthorized(t *testing.T) {
	handler := clientuserhandler.NewHandler(&mockService{}, time.Hour, false)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/change-password",
		bytes.NewBufferString(`{"current_password":"password123","password":"newpassword"}`),
	)

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ChangePassword_WrongPassword(t *testing.T) {
	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	service := &changePasswordService{
		fn: func(uuid.UUID, authdomain.ChangePasswordRequest) (*clientuserusecase.AuthResult, error) {
			return nil, authdomain.ErrInvalidCredentials
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/change-password",
		bytes.NewBufferString(`{"current_password":"wrong-password","password":"newpassword"}`),
	)
	req = req.WithContext(requestcontext.WithUserID(req.Context(), userID))

	rec := httptest.NewRecorder()
	handler.ChangePassword(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
