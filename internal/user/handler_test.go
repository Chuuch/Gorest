package user

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
)

type mockService struct {
	createFn        func(context.Context, CreateUserRequest) (*User, error)
	getByIDFn       func(context.Context, uuid.UUID) (*User, error)
	getByEmailFn    func(context.Context, string) (*User, error)
	existsByEmailFn func(context.Context, string) (bool, error)
	updateFn        func(context.Context, uuid.UUID, UpdateUserRequest) (*User, error)
	deleteFn        func(context.Context, uuid.UUID) error
}

func (m *mockService) Create(
	ctx context.Context,
	dto CreateUserRequest,
) (*User, error) {
	return m.createFn(ctx, dto)
}

func (m *mockService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockService) GetByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockService) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	return m.existsByEmailFn(ctx, email)
}

func (m *mockService) Update(
	ctx context.Context,
	id uuid.UUID,
	dto UpdateUserRequest,
) (*User, error) {
	return m.updateFn(ctx, id, dto)
}

func (m *mockService) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return m.deleteFn(ctx, id)
}

func TestHandler_Create(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		body           string
		serviceFn      func(context.Context, CreateUserRequest) (*User, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: `{"email":"john@example.com","password":"password123"}`,
			serviceFn: func(
				_ context.Context,
				dto CreateUserRequest,
			) (*User, error) {
				return &User{
					ID:           userID,
					Email:        dto.Email,
					PasswordHash: "hashed",
					CreatedAt:    time.Now().UTC(),
					UpdatedAt:    time.Now().UTC(),
				}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid json",
			body: `{"email":`,
			serviceFn: func(
				_ context.Context,
				_ CreateUserRequest,
			) (*User, error) {
				t.Fatal("service should not be called")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "email already exists",
			body: `{"email":"john@example.com","password":"password123"}`,
			serviceFn: func(
				_ context.Context,
				_ CreateUserRequest,
			) (*User, error) {
				return nil, ErrEmailAlreadyExists
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "internal error",
			body: `{"email":"john@example.com","password":"password123"}`,
			serviceFn: func(
				_ context.Context,
				_ CreateUserRequest,
			) (*User, error) {
				return nil, errors.New("database failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockService{
				createFn: tt.serviceFn,
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodPost,
				"/users",
				bytes.NewBufferString(tt.body),
			)

			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}

func TestHandler_GetByID(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		id             string
		serviceFn      func(context.Context, uuid.UUID) (*User, error)
		expectedStatus int
	}{
		{
			name: "success",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				id uuid.UUID,
			) (*User, error) {
				return &User{
					ID:        id,
					Email:     "john@example.com",
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid uuid",
			id:   "not-a-uuid",
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (*User, error) {
				t.Fatal("service should not be called")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (*User, error) {
				return nil, ErrUserNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) (*User, error) {
				return nil, errors.New("database failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockService{
				getByIDFn: tt.serviceFn,
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodGet,
				"/users/"+tt.id,
				nil,
			)

			req.SetPathValue("id", tt.id)

			if tt.id == userID.String() {
				req = req.WithContext(
					requestcontext.WithUserID(req.Context(), userID),
				)
			}

			rec := httptest.NewRecorder()

			handler.GetByID(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		id             string
		body           string
		serviceFn      func(context.Context, uuid.UUID, UpdateUserRequest) (*User, error)
		expectedStatus int
	}{
		{
			name: "success",
			id:   userID.String(),
			body: `{"email":"updated@example.com"}`,
			serviceFn: func(
				_ context.Context,
				id uuid.UUID,
				dto UpdateUserRequest,
			) (*User, error) {
				return &User{
					ID:        id,
					Email:     dto.Email,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid uuid",
			id:   "not-a-uuid",
			body: `{"email":"updated@example.com"}`,
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ UpdateUserRequest,
			) (*User, error) {
				t.Fatal("service should not be called")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid json",
			id:   userID.String(),
			body: `{"email":`,
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ UpdateUserRequest,
			) (*User, error) {
				t.Fatal("service should not be called")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			id:   userID.String(),
			body: `{"email":"updated@example.com"}`,
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ UpdateUserRequest,
			) (*User, error) {
				return nil, ErrUserNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			id:   userID.String(),
			body: `{"email":"updated@example.com"}`,
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
				_ UpdateUserRequest,
			) (*User, error) {
				return nil, errors.New("database failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockService{
				updateFn: tt.serviceFn,
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodPut,
				"/users/"+tt.id,
				bytes.NewBufferString(tt.body),
			)

			req.SetPathValue("id", tt.id)

			if tt.id == userID.String() {
				req = req.WithContext(
					requestcontext.WithUserID(req.Context(), userID),
				)
			}

			rec := httptest.NewRecorder()

			handler.Update(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		id             string
		serviceFn      func(context.Context, uuid.UUID) error
		expectedStatus int
	}{
		{
			name: "success",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "invalid uuid",
			id:   "not-a-uuid",
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) error {
				t.Fatal("service should not be called")
				return nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) error {
				return ErrUserNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			id:   userID.String(),
			serviceFn: func(
				_ context.Context,
				_ uuid.UUID,
			) error {
				return errors.New("database failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockService{
				deleteFn: tt.serviceFn,
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/users/"+tt.id,
				nil,
			)

			req.SetPathValue("id", tt.id)

			if tt.id == userID.String() {
				req = req.WithContext(
					requestcontext.WithUserID(req.Context(), userID),
				)
			}

			rec := httptest.NewRecorder()

			handler.Delete(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}
		})
	}
}
