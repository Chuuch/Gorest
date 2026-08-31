package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, dto CreateUserRequest) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password, hash string) error
}

type service struct {
	repository UserRepository
	hasher     PasswordHasher
}

func NewService(
	repository UserRepository,
	hasher PasswordHasher,
) Service {
	return &service{
		repository: repository,
		hasher:     hasher,
	}
}

func (s *service) Create(
	ctx context.Context,
	dto CreateUserRequest,
) (*User, error) {
	passwordHash, err := s.hasher.Hash(dto.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := time.Now().UTC()

	u := &User{
		ID:           uuid.New(),
		Email:        dto.Email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repository.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (s *service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *service) GetByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	return s.repository.GetByEmail(ctx, email)
}

func (s *service) Update(
	ctx context.Context,
	id uuid.UUID,
	dto UpdateUserRequest,
) (*User, error) {
	u, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user for update: %w", err)
	}

	u.Email = dto.Email

	if dto.Password != nil {
		passwordHash, err := s.hasher.Hash(*dto.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}

		u.PasswordHash = passwordHash
	}

	u.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return u, nil
}

func (s *service) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
