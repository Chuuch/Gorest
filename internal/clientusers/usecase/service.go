package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	authdomain "github.com/chuuch/gorest/internal/auth/domain"
	authrepository "github.com/chuuch/gorest/internal/auth/repository"
	"github.com/chuuch/gorest/internal/auth/security"
	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientrepository "github.com/chuuch/gorest/internal/client/repository"
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	clientuserrepository "github.com/chuuch/gorest/internal/clientusers/repository"
	"github.com/chuuch/gorest/internal/database"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Member struct {
	UserID         uuid.UUID
	Email          string
	OrganizationID uuid.UUID
	ClientID       uuid.UUID
	CreatedAt      time.Time
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *userdomain.User
	Organization *orgdomain.Organization
	Client       *clientdomain.Client
	Role         string
}

type Service interface {
	List(ctx context.Context, organizationID, clientID uuid.UUID) ([]Member, error)
	Create(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		actorRole orgdomain.Role,
		req clientuserdomain.CreateClientUserRequest,
	) (*Member, error)
	Login(ctx context.Context, req clientuserdomain.LoginRequest) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (*AuthResult, error)
}

type service struct {
	users         userusecase.Service
	clients       clientrepository.ClientRepository
	clientUsers   clientuserrepository.ClientUserRepository
	memberships   orgrepository.MembershipRepository
	organizations orgrepository.OrganizationRepository
	refreshTokens authrepository.RefreshTokenRepository
	tokens        security.TokenManager
	passwords     userusecase.PasswordHasher
	db            *pgxpool.Pool
	refreshTTL    time.Duration
}

func NewService(
	users userusecase.Service,
	clients clientrepository.ClientRepository,
	clientUsers clientuserrepository.ClientUserRepository,
	memberships orgrepository.MembershipRepository,
	organizations orgrepository.OrganizationRepository,
	refreshTokens authrepository.RefreshTokenRepository,
	tokens security.TokenManager,
	passwords userusecase.PasswordHasher,
	db *pgxpool.Pool,
	refreshTTL time.Duration,
) Service {
	return &service{
		users:         users,
		clients:       clients,
		clientUsers:   clientUsers,
		memberships:   memberships,
		organizations: organizations,
		refreshTokens: refreshTokens,
		tokens:        tokens,
		passwords:     passwords,
		db:            db,
		refreshTTL:    refreshTTL,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]Member, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	rows, err := s.clientUsers.ListByClientID(ctx, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list client users: %w", err)
	}

	members := make([]Member, 0, len(rows))

	for _, row := range rows {
		user, err := s.users.GetByID(ctx, row.UserID)
		if err != nil {
			return nil, fmt.Errorf("get client user: %w", err)
		}

		members = append(members, Member{
			UserID:         user.ID,
			Email:          user.Email,
			OrganizationID: row.OrganizationID,
			ClientID:       row.ClientID,
			CreatedAt:      row.CreatedAt,
		})
	}

	return members, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	req clientuserdomain.CreateClientUserRequest,
) (*Member, error) {
	if !actorRole.CanManageMembers() {
		return nil, clientuserdomain.ErrForbidden
	}

	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	var member *Member

	err := database.WithTransaction(ctx, s.db, func(ctx context.Context) error {
		existing, getErr := s.users.GetByEmail(ctx, req.Email)
		if getErr != nil && !errors.Is(getErr, userdomain.ErrUserNotFound) {
			return fmt.Errorf("get user by email: %w", getErr)
		}

		var user *userdomain.User

		if existing != nil {
			_, membershipErr := s.memberships.GetByUserID(ctx, existing.ID)
			if membershipErr == nil {
				return clientuserdomain.ErrUserIsStaff
			}
			if !errors.Is(membershipErr, orgdomain.ErrMembershipNotFound) {
				return fmt.Errorf("get membership: %w", membershipErr)
			}

			_, clientUserErr := s.clientUsers.GetByUserID(ctx, existing.ID)
			if clientUserErr == nil {
				return clientuserdomain.ErrClientUserAlreadyExists
			}
			if !errors.Is(clientUserErr, clientuserdomain.ErrClientUserNotFound) {
				return fmt.Errorf("get client user: %w", clientUserErr)
			}

			user = existing
		} else {
			created, createErr := s.users.Create(ctx, userdomain.CreateUserRequest{
				Email:    req.Email,
				Password: req.Password,
			})
			if createErr != nil {
				return fmt.Errorf("create user: %w", createErr)
			}

			user = created
		}

		now := time.Now().UTC()

		clientUser := &clientuserdomain.ClientUser{
			ID:             uuid.New(),
			OrganizationID: organizationID,
			ClientID:       clientID,
			UserID:         user.ID,
			CreatedAt:      now,
		}

		if err := s.clientUsers.Create(ctx, clientUser); err != nil {
			return err
		}

		member = &Member{
			UserID:         user.ID,
			Email:          user.Email,
			OrganizationID: organizationID,
			ClientID:       clientID,
			CreatedAt:      now,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (s *service) Login(
	ctx context.Context,
	req clientuserdomain.LoginRequest,
) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, userdomain.ErrUserNotFound) {
			return nil, authdomain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if err := s.passwords.Compare(req.Password, user.PasswordHash); err != nil {
		if errors.Is(err, password.ErrPasswordMismatch) {
			return nil, authdomain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("compare password: %w", err)
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *service) Refresh(
	ctx context.Context,
	refreshToken string,
) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, authdomain.ErrInvalidToken
	}

	storedToken, err := s.refreshTokens.GetByHash(ctx, s.tokens.HashRefreshToken(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return nil, authdomain.ErrTokenRevoked
	}

	if !time.Now().UTC().Before(storedToken.ExpiresAt) {
		return nil, authdomain.ErrTokenExpired
	}

	if err := s.refreshTokens.Revoke(ctx, storedToken.ID); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	return s.issueTokens(ctx, storedToken.UserID)
}

func (s *service) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	if refreshToken == "" {
		return authdomain.ErrInvalidToken
	}

	storedToken, err := s.refreshTokens.GetByHash(ctx, s.tokens.HashRefreshToken(refreshToken))
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}

	if storedToken.RevokedAt != nil {
		return authdomain.ErrTokenRevoked
	}

	return s.refreshTokens.Revoke(ctx, storedToken.ID)
}

func (s *service) Me(
	ctx context.Context,
	userID uuid.UUID,
) (*AuthResult, error) {
	return s.sessionFor(ctx, userID, "")
}

func (s *service) issueTokens(
	ctx context.Context,
	userID uuid.UUID,
) (*AuthResult, error) {
	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()

	if err := s.refreshTokens.Create(ctx, &authdomain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: s.tokens.HashRefreshToken(refreshToken),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return s.sessionFor(ctx, userID, refreshToken)
}

func (s *service) sessionFor(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
) (*AuthResult, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	clientUser, err := s.clientUsers.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, clientuserdomain.ErrClientUserNotFound) {
			return nil, authdomain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get client user: %w", err)
	}

	client, err := s.clients.GetByID(ctx, clientUser.ClientID, clientUser.OrganizationID)
	if err != nil {
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, clientUser.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}

	accessToken, err := s.tokens.GenerateAccessToken(
		userID,
		clientUser.OrganizationID,
		clientUser.ClientID,
		clientuserdomain.RoleClient,
	)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		Organization: org,
		Client:       client,
		Role:         clientuserdomain.RoleClient,
	}, nil
}
