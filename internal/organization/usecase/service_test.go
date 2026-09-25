package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/invites"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgpostgres "github.com/chuuch/gorest/internal/organization/postgres"
	orgusecase "github.com/chuuch/gorest/internal/organization/usecase"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	userpostgres "github.com/chuuch/gorest/internal/user/postgres"
	userusecase "github.com/chuuch/gorest/internal/user/usecase"
	"github.com/chuuch/gorest/pkg/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testBcryptCost = 4
	testPassword   = "password123"
)

type recordingInviter struct {
	calls []invites.IssueInput
}

func (r *recordingInviter) Issue(_ context.Context, in invites.IssueInput) error {
	r.calls = append(r.calls, in)
	return nil
}

func (r *recordingInviter) Accept(context.Context, string, string) error {
	return nil
}

func setupOrganizationTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gorest_test"),
		tcpostgres.WithUsername("gorest"),
		tcpostgres.WithPassword("gorest"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	require.NoError(t, db.Ping(ctx))

	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE organizations (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE memberships (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
			created_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT memberships_user_id_unique UNIQUE (user_id),
			CONSTRAINT memberships_org_user_unique UNIQUE (organization_id, user_id)
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

type organizationTestDependencies struct {
	service     orgusecase.Service
	users       userusecase.Service
	memberships *orgpostgres.MembershipRepository
	inviter     *recordingInviter
}

func setupOrganizationService(
	t *testing.T,
	db *pgxpool.Pool,
) organizationTestDependencies {
	t.Helper()

	userRepository := userpostgres.NewRepository(db)
	passwordHasher := password.NewBcryptHasher(testBcryptCost)
	userService := userusecase.NewService(userRepository, passwordHasher)
	membershipRepository := orgpostgres.NewMembershipRepository(db)
	inviter := &recordingInviter{}

	return organizationTestDependencies{
		service: orgusecase.NewService(
			userService,
			membershipRepository,
			orgpostgres.NewOrganizationRepository(db),
			inviter,
			db,
		),
		users:       userService,
		memberships: membershipRepository,
		inviter:     inviter,
	}
}

func seedOrganization(
	t *testing.T,
	db *pgxpool.Pool,
) uuid.UUID {
	t.Helper()

	organizationID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO organizations (
				id,
				name,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		"Acme",
		now,
		now,
	)
	require.NoError(t, err)

	return organizationID
}

func seedMembership(
	t *testing.T,
	deps organizationTestDependencies,
	organizationID uuid.UUID,
	email string,
	role orgdomain.Role,
) *userdomain.User {
	t.Helper()

	ctx := context.Background()

	user, err := deps.users.Create(ctx, userdomain.CreateUserRequest{
		Email:    email,
		Password: testPassword,
	})
	require.NoError(t, err)

	err = deps.memberships.Create(ctx, &orgdomain.Membership{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		UserID:         user.ID,
		Role:           role,
		CreatedAt:      time.Now().UTC(),
	})
	require.NoError(t, err)

	return user
}

func TestOrganizationService_ListMembers(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	owner := seedMembership(
		t,
		deps,
		organizationID,
		"owner@example.com",
		orgdomain.RoleOwner,
	)
	member := seedMembership(
		t,
		deps,
		organizationID,
		"member@example.com",
		orgdomain.RoleMember,
	)

	members, err := deps.service.ListMembers(
		context.Background(),
		organizationID,
	)

	require.NoError(t, err)
	require.Len(t, members, 2)
	require.Equal(t, owner.ID, members[0].UserID)
	require.Equal(t, "owner@example.com", members[0].Email)
	require.Equal(t, orgdomain.RoleOwner, members[0].Role)
	require.Equal(t, member.ID, members[1].UserID)
	require.Equal(t, "member@example.com", members[1].Email)
	require.Equal(t, orgdomain.RoleMember, members[1].Role)
}

func TestOrganizationService_CreateMember(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(
		t,
		deps,
		organizationID,
		"owner@example.com",
		orgdomain.RoleOwner,
	)

	member, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "member",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "ada@example.com", member.Email)
	require.Equal(t, orgdomain.RoleMember, member.Role)
	require.Len(t, deps.inviter.calls, 1)
	require.Equal(t, member.UserID, deps.inviter.calls[0].UserID)
	require.Equal(t, "ada@example.com", deps.inviter.calls[0].Email)
	require.Equal(t, "Acme", deps.inviter.calls[0].OrganizationName)
	require.Equal(t, invites.KindStaff, deps.inviter.calls[0].Kind)

	members, err := deps.service.ListMembers(
		context.Background(),
		organizationID,
	)

	require.NoError(t, err)
	require.Len(t, members, 2)
}

func TestOrganizationService_CreateMember_AdminCanAdd(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(
		t,
		deps,
		organizationID,
		"admin@example.com",
		orgdomain.RoleAdmin,
	)

	member, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleAdmin,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "member",
		},
	)

	require.NoError(t, err)
	require.Equal(t, orgdomain.RoleMember, member.Role)
	require.Len(t, deps.inviter.calls, 1)
}

func TestOrganizationService_CreateMember_Forbidden(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)

	_, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleMember,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "member",
		},
	)

	require.ErrorIs(t, err, orgdomain.ErrForbidden)
	require.Empty(t, deps.inviter.calls)
}

func TestOrganizationService_CreateMember_CannotCreateOwner(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)

	_, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "owner",
		},
	)

	require.ErrorIs(t, err, orgdomain.ErrCannotCreateOwner)
	require.Empty(t, deps.inviter.calls)
}

func TestOrganizationService_CreateMember_AlreadyExists(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(
		t,
		deps,
		organizationID,
		"owner@example.com",
		orgdomain.RoleOwner,
	)
	_ = seedMembership(
		t,
		deps,
		organizationID,
		"ada@example.com",
		orgdomain.RoleMember,
	)

	_, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "member",
		},
	)

	require.ErrorIs(t, err, orgdomain.ErrMemberAlreadyExists)
	require.Empty(t, deps.inviter.calls)
}

func TestOrganizationService_CreateMember_ExistingUserWithoutMembership(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(
		t,
		deps,
		organizationID,
		"owner@example.com",
		orgdomain.RoleOwner,
	)

	existing, err := deps.users.Create(
		context.Background(),
		userdomain.CreateUserRequest{
			Email:    "ada@example.com",
			Password: testPassword,
		},
	)
	require.NoError(t, err)

	member, err := deps.service.CreateMember(
		context.Background(),
		organizationID,
		orgdomain.RoleOwner,
		orgdomain.CreateMemberRequest{
			Email: "ada@example.com",
			Role:  "admin",
		},
	)

	require.NoError(t, err)
	require.Equal(t, existing.ID, member.UserID)
	require.Equal(t, "ada@example.com", member.Email)
	require.Equal(t, orgdomain.RoleAdmin, member.Role)
	require.Len(t, deps.inviter.calls, 1)
}

func TestOrganizationService_UpdateMember(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	updated, err := deps.service.UpdateMember(
		context.Background(),
		organizationID,
		member.ID,
		orgdomain.RoleAdmin,
		orgdomain.UpdateMemberRequest{Role: "admin"},
	)

	require.NoError(t, err)
	require.Equal(t, member.ID, updated.UserID)
	require.Equal(t, orgdomain.RoleAdmin, updated.Role)
}

func TestOrganizationService_UpdateMember_Forbidden(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	_, err := deps.service.UpdateMember(
		context.Background(),
		organizationID,
		member.ID,
		orgdomain.RoleMember,
		orgdomain.UpdateMemberRequest{Role: "admin"},
	)

	require.ErrorIs(t, err, orgdomain.ErrForbidden)
}

func TestOrganizationService_UpdateMember_CannotAssignOwner(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	_, err := deps.service.UpdateMember(
		context.Background(),
		organizationID,
		member.ID,
		orgdomain.RoleOwner,
		orgdomain.UpdateMemberRequest{Role: "owner"},
	)

	require.ErrorIs(t, err, orgdomain.ErrCannotAssignOwner)
}

func TestOrganizationService_UpdateMember_LastOwner(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	owner := seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)

	_, err := deps.service.UpdateMember(
		context.Background(),
		organizationID,
		owner.ID,
		orgdomain.RoleOwner,
		orgdomain.UpdateMemberRequest{Role: "admin"},
	)

	require.ErrorIs(t, err, orgdomain.ErrLastOwner)
}

func TestOrganizationService_UpdateMember_SecondOwnerCanBeDemoted(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	otherOwner := seedMembership(t, deps, organizationID, "other@example.com", orgdomain.RoleOwner)

	updated, err := deps.service.UpdateMember(
		context.Background(),
		organizationID,
		otherOwner.ID,
		orgdomain.RoleOwner,
		orgdomain.UpdateMemberRequest{Role: "admin"},
	)

	require.NoError(t, err)
	require.Equal(t, orgdomain.RoleAdmin, updated.Role)
}

func TestOrganizationService_UpdateMember_WrongOrgNotFound(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	otherOrganizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	_, err := deps.service.UpdateMember(
		context.Background(),
		otherOrganizationID,
		member.ID,
		orgdomain.RoleOwner,
		orgdomain.UpdateMemberRequest{Role: "admin"},
	)

	require.ErrorIs(t, err, orgdomain.ErrMembershipNotFound)
}

func TestOrganizationService_DeleteMember(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	err := deps.service.DeleteMember(
		context.Background(),
		organizationID,
		member.ID,
		orgdomain.RoleOwner,
	)
	require.NoError(t, err)

	members, err := deps.service.ListMembers(context.Background(), organizationID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.Equal(t, orgdomain.RoleOwner, members[0].Role)
}

func TestOrganizationService_DeleteMember_LastOwner(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	owner := seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)

	err := deps.service.DeleteMember(
		context.Background(),
		organizationID,
		owner.ID,
		orgdomain.RoleOwner,
	)

	require.ErrorIs(t, err, orgdomain.ErrLastOwner)
}

func TestOrganizationService_DeleteMember_SecondOwner(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)
	otherOwner := seedMembership(t, deps, organizationID, "other@example.com", orgdomain.RoleOwner)

	err := deps.service.DeleteMember(
		context.Background(),
		organizationID,
		otherOwner.ID,
		orgdomain.RoleAdmin,
	)
	require.NoError(t, err)
}

func TestOrganizationService_DeleteMember_Forbidden(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	member := seedMembership(t, deps, organizationID, "ada@example.com", orgdomain.RoleMember)

	err := deps.service.DeleteMember(
		context.Background(),
		organizationID,
		member.ID,
		orgdomain.RoleMember,
	)

	require.ErrorIs(t, err, orgdomain.ErrForbidden)
}

func TestOrganizationService_DeleteMember_NotFound(t *testing.T) {
	db, cleanup := setupOrganizationTestDatabase(t)
	defer cleanup()

	deps := setupOrganizationService(t, db)
	organizationID := seedOrganization(t, db)
	_ = seedMembership(t, deps, organizationID, "owner@example.com", orgdomain.RoleOwner)

	err := deps.service.DeleteMember(
		context.Background(),
		organizationID,
		uuid.New(),
		orgdomain.RoleOwner,
	)

	require.ErrorIs(t, err, orgdomain.ErrMembershipNotFound)
}
