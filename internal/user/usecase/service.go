package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/user/domain"
	"github.com/chuuch/gorest/internal/user/repository"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, dto domain.CreateUserRequest) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, id uuid.UUID, dto domain.UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password, hash string) error
}

type service struct {
	repository repository.UserRepository
	hasher     PasswordHasher
}

func NewService(
	repository repository.UserRepository,
	hasher PasswordHasher,
) Service {
	return &service{
		repository: repository,
		hasher:     hasher,
	}
}

func (s *service) Create(
	ctx context.Context,
	dto domain.CreateUserRequest,
) (*domain.User, error) {
	passwordHash, err := s.hasher.Hash(dto.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()

	u := &domain.User{
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
) (*domain.User, error) {
	u, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return u, nil
}

func (s *service) GetByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	u, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return u, nil
}

func (s *service) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	return s.repository.ExistsByEmail(ctx, email)
}

func (s *service) Update(
	ctx context.Context,
	id uuid.UUID,
	dto domain.UpdateUserRequest,
) (*domain.User, error) {
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
