package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockUserRepository struct {
	createFn      func(context.Context, *User) error
	getByIDFn     func(context.Context, uuid.UUID) (*User, error)
	getByEmailFn  func(context.Context, string) (*User, error)
	existsByEmail func(ctx context.Context, emaid string) (bool, error)
	updateFn      func(context.Context, *User) error
	deleteFn      func(context.Context, uuid.UUID) error
}

func (m *mockUserRepository) Create(
	ctx context.Context,
	user *User,
) error {
	return m.createFn(ctx, user)
}

func (m *mockUserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserRepository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	return m.existsByEmail(ctx, email)
}

func (m *mockUserRepository) Update(
	ctx context.Context,
	user *User,
) error {
	return m.updateFn(ctx, user)
}

func (m *mockUserRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return m.deleteFn(ctx, id)
}

type mockPasswordHasher struct {
	hashFn    func(string) (string, error)
	compareFn func(string, string) error
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	return m.hashFn(password)
}

func (m *mockPasswordHasher) Compare(password, hash string) error {
	return m.compareFn(password, hash)
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("hash failed")
	repositoryErr := errors.New("repository failed")

	tests := []struct {
		name          string
		dto           CreateUserRequest
		hashResult    string
		hashErr       error
		repositoryErr error
		wantErr       error
	}{
		{
			name: "success",
			dto: CreateUserRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			hashResult: "hashed-password",
		},
		{
			name: "hash error",
			dto: CreateUserRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			hashErr: hashErr,
			wantErr: hashErr,
		},
		{
			name: "repository error",
			dto: CreateUserRequest{
				Email:    "john@example.com",
				Password: "password123",
			},
			hashResult:    "hashed-password",
			repositoryErr: repositoryErr,
			wantErr:       repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var createdUser *User

			repository := &mockUserRepository{
				createFn: func(
					ctx context.Context,
					user *User,
				) error {
					createdUser = user
					return tt.repositoryErr
				},
			}

			hasher := &mockPasswordHasher{
				hashFn: func(password string) (string, error) {
					if password != tt.dto.Password {
						t.Errorf(
							"Hash() password = %q, want %q",
							password,
							tt.dto.Password,
						)
					}

					return tt.hashResult, tt.hashErr
				},
			}

			service := NewService(repository, hasher)

			got, err := service.Create(
				context.Background(),
				tt.dto,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Create() expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf(
						"Create() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if got == nil {
				t.Fatal("Create() returned nil user")
			}

			if createdUser == nil {
				t.Fatal("repository.Create() was not called")
			}

			if got != createdUser {
				t.Fatal("Create() returned a different user than repository.Create() received")
			}

			if got.ID == uuid.Nil {
				t.Error("Create() generated nil UUID")
			}

			if got.Email != tt.dto.Email {
				t.Errorf(
					"Email = %q, want %q",
					got.Email,
					tt.dto.Email,
				)
			}

			if got.PasswordHash != tt.hashResult {
				t.Errorf(
					"PasswordHash = %q, want %q",
					got.PasswordHash,
					tt.hashResult,
				)
			}

			if got.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}

			if got.UpdatedAt.IsZero() {
				t.Error("UpdatedAt is zero")
			}

			if !got.CreatedAt.Equal(got.UpdatedAt) {
				t.Error("CreatedAt and UpdatedAt should initially be equal")
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repositoryErr := errors.New("repository failed")

	expectedUser := &User{
		ID:           userID,
		Email:        "john@example.com",
		PasswordHash: "hashed-password",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	tests := []struct {
		name         string
		repositoryFn func() (*User, error)
		wantUser     *User
		wantErr      error
	}{
		{
			name: "success",
			repositoryFn: func() (*User, error) {
				return expectedUser, nil
			},
			wantUser: expectedUser,
		},
		{
			name: "repository error",
			repositoryFn: func() (*User, error) {
				return nil, repositoryErr
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &mockUserRepository{
				getByIDFn: func(
					ctx context.Context,
					id uuid.UUID,
				) (*User, error) {
					if id != userID {
						t.Errorf(
							"id = %v, want %v",
							id,
							userID,
						)
					}

					return tt.repositoryFn()
				},
			}

			service := NewService(
				repository,
				&mockPasswordHasher{},
			)

			got, err := service.GetByID(
				context.Background(),
				userID,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("GetByID() expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf(
						"GetByID() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if got != tt.wantUser {
				t.Error("GetByID() returned unexpected user")
			}
		})
	}
}

func TestService_GetByEmail(t *testing.T) {
	t.Parallel()

	email := "john@example.com"
	repositoryErr := errors.New("repository failed")

	expectedUser := &User{
		ID:    uuid.New(),
		Email: email,
	}

	tests := []struct {
		name         string
		repositoryFn func() (*User, error)
		wantUser     *User
		wantErr      error
	}{
		{
			name: "success",
			repositoryFn: func() (*User, error) {
				return expectedUser, nil
			},
			wantUser: expectedUser,
		},
		{
			name: "repository error",
			repositoryFn: func() (*User, error) {
				return nil, repositoryErr
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &mockUserRepository{
				getByEmailFn: func(
					ctx context.Context,
					gotEmail string,
				) (*User, error) {
					if gotEmail != email {
						t.Errorf(
							"email = %q, want %q",
							gotEmail,
							email,
						)
					}

					return tt.repositoryFn()
				},
			}

			service := NewService(
				repository,
				&mockPasswordHasher{},
			)

			got, err := service.GetByEmail(
				context.Background(),
				email,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("GetByEmail() expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf(
						"GetByEmail() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("GetByEmail() error = %v", err)
			}

			if got != tt.wantUser {
				t.Error("GetByEmail() returned unexpected user")
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	getErr := errors.New("get user failed")
	hashErr := errors.New("hash failed")
	updateErr := errors.New("update failed")

	tests := []struct {
		name                string
		email               string
		password            string
		changePassword      bool
		existingUser        *User
		hashResult          string
		hashErr             error
		repositoryGetErr    error
		repositoryUpdateErr error
		wantPasswordHash    string
		wantErr             error
	}{
		{
			name:  "update email without password",
			email: "new@example.com",
			existingUser: &User{
				ID:           userID,
				Email:        "old@example.com",
				PasswordHash: "old-hash",
				CreatedAt:    time.Now().UTC().Add(-time.Hour),
				UpdatedAt:    time.Now().UTC().Add(-time.Hour),
			},
			wantPasswordHash: "old-hash",
		},
		{
			name:           "update email and password",
			email:          "new@example.com",
			password:       "new-password",
			changePassword: true,
			existingUser: &User{
				ID:           userID,
				Email:        "old@example.com",
				PasswordHash: "old-hash",
				CreatedAt:    time.Now().UTC().Add(-time.Hour),
				UpdatedAt:    time.Now().UTC().Add(-time.Hour),
			},
			hashResult:       "new-hash",
			wantPasswordHash: "new-hash",
		},
		{
			name:             "get user error",
			email:            "new@example.com",
			repositoryGetErr: getErr,
			wantErr:          getErr,
		},
		{
			name:           "hash error",
			email:          "new@example.com",
			password:       "new-password",
			changePassword: true,
			existingUser: &User{
				ID:           userID,
				Email:        "old@example.com",
				PasswordHash: "old-hash",
				CreatedAt:    time.Now().UTC().Add(-time.Hour),
				UpdatedAt:    time.Now().UTC().Add(-time.Hour),
			},
			hashErr: hashErr,
			wantErr: hashErr,
		},
		{
			name:  "repository update error",
			email: "new@example.com",
			existingUser: &User{
				ID:           userID,
				Email:        "old@example.com",
				PasswordHash: "old-hash",
				CreatedAt:    time.Now().UTC().Add(-time.Hour),
				UpdatedAt:    time.Now().UTC().Add(-time.Hour),
			},
			repositoryUpdateErr: updateErr,
			wantErr:             updateErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var updatedUser *User

			repository := &mockUserRepository{
				getByIDFn: func(
					ctx context.Context,
					id uuid.UUID,
				) (*User, error) {
					if id != userID {
						t.Errorf(
							"id = %v, want %v",
							id,
							userID,
						)
					}

					return tt.existingUser, tt.repositoryGetErr
				},
				updateFn: func(
					ctx context.Context,
					user *User,
				) error {
					updatedUser = user
					return tt.repositoryUpdateErr
				},
			}

			hasher := &mockPasswordHasher{
				hashFn: func(password string) (string, error) {
					if !tt.changePassword {
						t.Error("Hash() should not have been called")
					}

					if password != tt.password {
						t.Errorf(
							"Hash() password = %q, want %q",
							password,
							tt.password,
						)
					}

					return tt.hashResult, tt.hashErr
				},
			}

			service := NewService(repository, hasher)

			var originalUpdatedAt time.Time

			if tt.existingUser != nil {
				originalUpdatedAt = tt.existingUser.UpdatedAt
			}

			var password *string

			if tt.changePassword {
				password = &tt.password
			}

			dto := UpdateUserRequest{
				Email:    tt.email,
				Password: password,
			}

			got, err := service.Update(
				context.Background(),
				userID,
				dto,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Update() expected error")
				}

				if !errors.Is(err, tt.wantErr) {
					t.Errorf(
						"Update() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if got == nil {
				t.Fatal("Update() returned nil user")
			}

			if updatedUser == nil {
				t.Fatal("repository.Update() was not called")
			}

			if got != updatedUser {
				t.Fatal("Update() returned a different user than repository.Update() received")
			}

			if got.ID != userID {
				t.Errorf(
					"ID = %v, want %v",
					got.ID,
					userID,
				)
			}

			if got.Email != tt.email {
				t.Errorf(
					"Email = %q, want %q",
					got.Email,
					tt.email,
				)
			}

			if got.PasswordHash != tt.wantPasswordHash {
				t.Errorf(
					"PasswordHash = %q, want %q",
					got.PasswordHash,
					tt.wantPasswordHash,
				)
			}

			if !got.UpdatedAt.After(originalUpdatedAt) {
				t.Error("UpdatedAt was not updated")
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repositoryErr := errors.New("repository failed")

	tests := []struct {
		name          string
		repositoryErr error
		wantErr       error
	}{
		{
			name: "success",
		},
		{
			name:          "repository error",
			repositoryErr: repositoryErr,
			wantErr:       repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &mockUserRepository{
				deleteFn: func(
					ctx context.Context,
					id uuid.UUID,
				) error {
					if id != userID {
						t.Errorf(
							"id = %v, want %v",
							id,
							userID,
						)
					}

					return tt.repositoryErr
				},
			}

			service := NewService(
				repository,
				&mockPasswordHasher{},
			)

			err := service.Delete(
				context.Background(),
				userID,
			)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Delete() error = %v", err)
				}

				return
			}

			if err == nil {
				t.Fatal("Delete() expected error")
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"Delete() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
