package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chuuch/gorest/internal/database"
	"github.com/chuuch/gorest/internal/invites"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	"github.com/chuuch/gorest/internal/search"
	"github.com/chuuch/gorest/internal/taxid"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Member struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Role        orgdomain.Role
	CreatedAt   time.Time
}

type Service interface {
	Update(
		ctx context.Context,
		organizationID uuid.UUID,
		actorRole orgdomain.Role,
		req orgdomain.UpdateOrganizationRequest,
	) (*orgdomain.Organization, error)
	ListMembers(
		ctx context.Context,
		organizationID uuid.UUID,
		query string,
	) ([]Member, error)
	CreateMember(
		ctx context.Context,
		organizationID uuid.UUID,
		actorRole orgdomain.Role,
		req orgdomain.CreateMemberRequest,
	) (*Member, error)
	UpdateMember(
		ctx context.Context,
		organizationID, memberUserID uuid.UUID,
		actorRole orgdomain.Role,
		req orgdomain.UpdateMemberRequest,
	) (*Member, error)
	DeleteMember(
		ctx context.Context,
		organizationID, memberUserID uuid.UUID,
		actorRole orgdomain.Role,
	) error
}

type service struct {
	users         userusecase.Service
	memberships   orgrepository.MembershipRepository
	organizations orgrepository.OrganizationRepository
	inviter       invites.Service
	db            *pgxpool.Pool
}

func NewService(
	users userusecase.Service,
	memberships orgrepository.MembershipRepository,
	organizations orgrepository.OrganizationRepository,
	inviter invites.Service,
	db *pgxpool.Pool,
) Service {
	return &service{
		users:         users,
		memberships:   memberships,
		organizations: organizations,
		inviter:       inviter,
		db:            db,
	}
}

func (s *service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	actorRole orgdomain.Role,
	req orgdomain.UpdateOrganizationRequest,
) (*orgdomain.Organization, error) {
	if !actorRole.CanManageMembers() {
		return nil, orgdomain.ErrForbidden
	}

	org, err := s.organizations.GetByID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	rate := req.DefaultVATRateBPS
	if rate == 0 {
		rate = 2000
	}

	org.Name = req.Name
	org.LegalName = strings.TrimSpace(req.LegalName)
	org.RegistrationNumber = strings.TrimSpace(req.RegistrationNumber)
	org.VATID = taxid.NormalizeVATID(req.VATID)
	org.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	org.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	org.City = strings.TrimSpace(req.City)
	org.PostalCode = strings.TrimSpace(req.PostalCode)
	org.Country = taxid.NormalizeCountry(req.Country)
	org.DefaultVATRateBPS = rate
	org.BankIBAN = strings.TrimSpace(req.BankIBAN)
	org.BankBIC = strings.TrimSpace(req.BankBIC)
	org.BankName = strings.TrimSpace(req.BankName)
	org.UpdatedAt = time.Now().UTC()

	if err := s.organizations.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *service) ListMembers(
	ctx context.Context,
	organizationID uuid.UUID,
	query string,
) ([]Member, error) {
	memberships, err := s.memberships.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}

	q := search.Normalize(query)
	members := make([]Member, 0, len(memberships))

	for _, membership := range memberships {
		user, err := s.users.GetByID(ctx, membership.UserID)
		if err != nil {
			return nil, fmt.Errorf("get member user: %w", err)
		}

		if !search.Matches(user.Email, q) && !search.Matches(user.DisplayName, q) {
			continue
		}

		members = append(members, Member{
			UserID:      user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        membership.Role,
			CreatedAt:   membership.CreatedAt,
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
			discarded, err := invites.DiscardedPassword()
			if err != nil {
				return fmt.Errorf("generate discarded password: %w", err)
			}

			user, err = s.users.Create(ctx, userdomain.CreateUserRequest{
				Email:    req.Email,
				Password: discarded,
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
			UserID:      user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        membership.Role,
			CreatedAt:   membership.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}

	if err := s.inviter.Issue(ctx, invites.IssueInput{
		UserID:           member.UserID,
		Email:            member.Email,
		OrganizationName: org.Name,
		Kind:             invites.KindStaff,
	}); err != nil {
		return nil, fmt.Errorf("issue invite: %w", err)
	}

	return member, nil
}

func (s *service) membershipInOrg(
	ctx context.Context,
	organizationID, memberUserID uuid.UUID,
) (*orgdomain.Membership, error) {
	membership, err := s.memberships.GetByUserID(ctx, memberUserID)
	if err != nil {
		return nil, err
	}

	if membership.OrganizationID != organizationID {
		return nil, orgdomain.ErrMembershipNotFound
	}

	return membership, nil
}

func (s *service) guardLastOwner(
	ctx context.Context,
	organizationID uuid.UUID,
	membership *orgdomain.Membership,
) error {
	if membership.Role != orgdomain.RoleOwner {
		return nil
	}

	count, err := s.memberships.CountOwners(ctx, organizationID)
	if err != nil {
		return err
	}

	if count <= 1 {
		return orgdomain.ErrLastOwner
	}

	return nil
}

func (s *service) UpdateMember(
	ctx context.Context,
	organizationID, memberUserID uuid.UUID,
	actorRole orgdomain.Role,
	req orgdomain.UpdateMemberRequest,
) (*Member, error) {
	if !actorRole.CanManageMembers() {
		return nil, orgdomain.ErrForbidden
	}

	if req.Role == string(orgdomain.RoleOwner) {
		return nil, orgdomain.ErrCannotAssignOwner
	}

	var member *Member

	err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		membership, err := s.membershipInOrg(ctx, organizationID, memberUserID)
		if err != nil {
			return err
		}

		if err := s.guardLastOwner(ctx, organizationID, membership); err != nil {
			return err
		}

		if err := s.memberships.UpdateRole(
			ctx,
			organizationID,
			memberUserID,
			orgdomain.Role(req.Role),
		); err != nil {
			return err
		}

		user, err := s.users.GetByID(ctx, memberUserID)
		if err != nil {
			return fmt.Errorf("get member user: %w", err)
		}

		member = &Member{
			UserID:      user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        orgdomain.Role(req.Role),
			CreatedAt:   membership.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return member, nil
}

func (s *service) DeleteMember(
	ctx context.Context,
	organizationID, memberUserID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return orgdomain.ErrForbidden
	}

	return database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		membership, err := s.membershipInOrg(ctx, organizationID, memberUserID)
		if err != nil {
			return err
		}

		if err := s.guardLastOwner(ctx, organizationID, membership); err != nil {
			return err
		}

		return s.memberships.Delete(ctx, organizationID, memberUserID)
	})
}
