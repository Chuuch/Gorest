package usecase_test

import (
	"context"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	ticketcommentdomain "github.com/chuuch/gorest/internal/ticketcomments/domain"
	ticketcommentpostgres "github.com/chuuch/gorest/internal/ticketcomments/postgres"
	ticketcommentusecase "github.com/chuuch/gorest/internal/ticketcomments/usecase"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTicketCommentTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			email TEXT NOT NULL,
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
		CREATE TABLE clients (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE tickets (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE ticket_comments (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT ticket_comments_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func seedUser(t *testing.T, db *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		userID,
		email,
		"hash",
		now,
		now,
	)
	require.NoError(t, err)

	return userID
}

func seedOrganization(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()

	organizationID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO organizations (id, name, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
		`,
		organizationID,
		name,
		now,
		now,
	)
	require.NoError(t, err)

	return organizationID
}

func seedClient(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID uuid.UUID,
	name string,
) uuid.UUID {
	t.Helper()

	clientID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO clients (id, organization_id, name, notes, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		clientID,
		organizationID,
		name,
		"",
		now,
		now,
	)
	require.NoError(t, err)

	return clientID
}

func seedTicket(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID, userID uuid.UUID,
	title string,
) uuid.UUID {
	t.Helper()

	ticketID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO tickets (
				id, organization_id, client_id, user_id, kind, status, title, body, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
		ticketID,
		organizationID,
		clientID,
		userID,
		"bug",
		"open",
		title,
		"Clicking Sign in does nothing on mobile.",
		now,
		now,
	)
	require.NoError(t, err)

	return ticketID
}

func setupTicketCommentService(db *pgxpool.Pool) ticketcommentusecase.Service {
	return ticketcommentusecase.NewService(
		ticketcommentpostgres.NewRepository(db),
		ticketpostgres.NewRepository(db),
	)
}

func TestTicketCommentService_CreateAndList(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "pat@example.com")
	staffID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")
	otherTicketID := seedTicket(t, db, organizationID, otherClientID, userID, "Other login issue")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		staffID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)
	require.Equal(t, "Can you try another browser?", comment.Body)
	require.Equal(t, ticketID, comment.TicketID)
	require.Equal(t, staffID, comment.UserID)

	own, err := service.List(context.Background(), organizationID, ticketID, clientID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	staff, err := service.List(context.Background(), organizationID, ticketID, uuid.Nil)
	require.NoError(t, err)
	require.Len(t, staff, 1)

	_, err = service.List(context.Background(), organizationID, ticketID, otherClientID)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)

	_, err = service.List(context.Background(), otherOrganizationID, ticketID, uuid.Nil)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)

	other, err := service.List(context.Background(), organizationID, otherTicketID, otherClientID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestTicketCommentService_Create_PortalOnOwnTicket(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		clientID,
		ticketcommentdomain.CreateCommentRequest{Body: "Still broken on Safari"},
	)
	require.NoError(t, err)
	require.Equal(t, "Still broken on Safari", comment.Body)
	require.Equal(t, userID, comment.UserID)
}

func TestTicketCommentService_Create_OtherClientNotFound(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	_, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		otherClientID,
		ticketcommentdomain.CreateCommentRequest{Body: "Nope"},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}

func TestTicketCommentService_Create_TicketNotFound(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Nope"},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}

func TestTicketCommentService_Update(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		comment.ID,
		userID,
		uuid.Nil,
		orgdomain.RoleMember,
		ticketcommentdomain.UpdateCommentRequest{Body: "Try Safari first."},
	)
	require.NoError(t, err)
	require.Equal(t, "Try Safari first.", updated.Body)

	listed, err := service.List(context.Background(), organizationID, ticketID, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, "Try Safari first.", listed[0].Body)
}

func TestTicketCommentService_Update_AdminCanEditOthers(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	authorID := seedUser(t, db, "mike@example.com")
	adminID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, authorID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		authorID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		comment.ID,
		adminID,
		uuid.Nil,
		orgdomain.RoleAdmin,
		ticketcommentdomain.UpdateCommentRequest{Body: "Edited by admin."},
	)
	require.NoError(t, err)
	require.Equal(t, "Edited by admin.", updated.Body)
}

func TestTicketCommentService_Update_MemberForbiddenOnOthers(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	authorID := seedUser(t, db, "ada@example.com")
	memberID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, authorID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		authorID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		comment.ID,
		memberID,
		uuid.Nil,
		orgdomain.RoleMember,
		ticketcommentdomain.UpdateCommentRequest{Body: "Nope"},
	)
	require.ErrorIs(t, err, ticketcommentdomain.ErrForbidden)
}

func TestTicketCommentService_Update_PortalOtherClient(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		clientID,
		ticketcommentdomain.CreateCommentRequest{Body: "Still broken on Safari"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		comment.ID,
		userID,
		otherClientID,
		orgdomain.Role("client"),
		ticketcommentdomain.UpdateCommentRequest{Body: "Nope"},
	)
	require.ErrorIs(t, err, ticketcommentdomain.ErrCommentNotFound)
}

func TestTicketCommentService_Delete(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		comment.ID,
		userID,
		uuid.Nil,
		orgdomain.RoleAdmin,
	)
	require.NoError(t, err)

	listed, err := service.List(context.Background(), organizationID, ticketID, uuid.Nil)
	require.NoError(t, err)
	require.Empty(t, listed)
}

func TestTicketCommentService_Delete_MemberForbiddenOnOthers(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	authorID := seedUser(t, db, "ada@example.com")
	memberID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, authorID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		authorID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		comment.ID,
		memberID,
		uuid.Nil,
		orgdomain.RoleMember,
	)
	require.ErrorIs(t, err, ticketcommentdomain.ErrForbidden)
}

func TestTicketCommentService_Delete_WrongOrg(t *testing.T) {
	db, cleanup := setupTicketCommentTestDatabase(t)
	defer cleanup()

	service := setupTicketCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		ticketID,
		userID,
		uuid.Nil,
		ticketcommentdomain.CreateCommentRequest{Body: "Can you try another browser?"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		otherOrganizationID,
		comment.ID,
		userID,
		uuid.Nil,
		orgdomain.RoleOwner,
	)
	require.ErrorIs(t, err, ticketcommentdomain.ErrCommentNotFound)
}
