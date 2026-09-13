package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/database"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Member struct {
	UserID    uuid.UUID
	Email     string
	Role      orgdomain.Role
	CreatedAt time.Time
}

type Service interface {
	ListMembers(ctx context.Context, organizationID uuid.UUID) ([]Member, error)
	CreateMember(ctx context.Context, organizationID uuid.UUID, actorRole orgdomain.Role, req orgdomain.CreateMemberRequest) (*Member, error)
}

type service struct {
	users       userusecase.Service
	memberships orgrepository.MembershipRepository
	db          *pgxpool.Pool
}

func NewService(
	users userusecase.Service,
	memberships orgrepository.MembershipRepository,
	db *pgxpool.Pool,
) Service {
	return &service{
		users:       users,
		memberships: memberships,
		db:          db,
	}
}

func (s *service) ListMembers(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Member, error) {
	memberships, err := s.memberships.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}

	members := make([]Member, 0, len(memberships))

	for _, membership := range memberships {
		user, err := s.users.GetByID(ctx, membership.UserID)
		if err != nil {
			return nil, fmt.Errorf("get member user: %w", err)
		}

		members = append(members, Member{
			UserID:    user.ID,
			Email:     user.Email,
			Role:      membership.Role,
			CreatedAt: membership.CreatedAt,
		})
	}

	return members, nil
}

func (s *service) CreateMember(
	ctx context.Context,
	organizationID uuid.UUID,
	actorRole orgdomain.Role,
	req orgdomain.CreateMemberRequest,
) (*Member, error) {
	if !actorRole.CanManageMembers() {
		return nil, orgdomain.ErrForbidden
	}

	if req.Role == string(orgdomain.RoleOwner) {
		return nil, orgdomain.ErrCannotCreateOwner
	}

	var member *Member

	err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		existing, err := s.users.GetByEmail(ctx, req.Email)
		if err != nil && !errors.Is(err, userdomain.ErrUserNotFound) {
			return fmt.Errorf("get user by email: %w", err)
		}

		var user *userdomain.User

		if existing != nil {
			_, err := s.memberships.GetByUserID(ctx, existing.ID)
			if err == nil {
				return orgdomain.ErrMemberAlreadyExists
			}

			if !errors.Is(err, orgdomain.ErrMembershipNotFound) {
				return fmt.Errorf("get membership: %w", err)
			}
			user = existing
		} else {
			user, err = s.users.Create(ctx, userdomain.CreateUserRequest{
				Email:    req.Email,
				Password: req.Password,
			})
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
		}

		now := time.Now().UTC()

		membership := &orgdomain.Membership{
			ID:             uuid.New(),
			OrganizationID: organizationID,
			UserID:         user.ID,
			Role:           orgdomain.Role(req.Role),
			CreatedAt:      now,
		}

		if err := s.memberships.Create(ctx, membership); err != nil {
			return err
		}

		member = &Member{
			UserID:    user.ID,
			Email:     user.Email,
			Role:      membership.Role,
			CreatedAt: membership.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return member, nil
}
