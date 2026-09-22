package usecase_test

import (
	"context"
	"testing"
	"time"

	commentdomain "github.com/chuuch/gorest/internal/comments/domain"
	commentpostgres "github.com/chuuch/gorest/internal/comments/postgres"
	commentusecase "github.com/chuuch/gorest/internal/comments/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupCommentTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
		CREATE TABLE projects (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT projects_client_name_unique UNIQUE (client_id, name)
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE tasks (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ticket_id UUID,
			title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			completed_at TIMESTAMPTZ NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT tasks_project_title_unique UNIQUE (project_id, title),
			CONSTRAINT tasks_status_check CHECK (status IN ('todo', 'in_progress', 'done'))
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE comments (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT comments_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
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
			INSERT INTO clients (
				id,
				organization_id,
				name,
				notes,
				created_at,
				updated_at
			)
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

func seedProject(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, clientID uuid.UUID,
	name string,
) uuid.UUID {
	t.Helper()

	projectID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO projects (
				id,
				organization_id,
				client_id,
				name,
				notes,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		projectID,
		organizationID,
		clientID,
		name,
		"",
		now,
		now,
	)
	require.NoError(t, err)

	return projectID
}

func seedTask(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, projectID uuid.UUID,
	title string,
) uuid.UUID {
	t.Helper()

	taskID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO tasks (
				id,
				organization_id,
				project_id,
				title,
				notes,
				status,
				completed_at,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
		taskID,
		organizationID,
		projectID,
		title,
		"",
		"todo",
		nil,
		now,
		now,
	)
	require.NoError(t, err)

	return taskID
}

func setupCommentService(db *pgxpool.Pool) commentusecase.Service {
	return commentusecase.NewService(
		commentpostgres.NewRepository(db),
		taskpostgres.NewRepository(db),
	)
}

func TestCommentService_CreateAndList(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")
	otherTaskID := seedTask(t, db, otherOrganizationID, otherProjectID, "Fix login")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		commentdomain.CreateCommentRequest{Body: "Check the OAuth redirect"},
	)

	require.NoError(t, err)
	require.Equal(t, "Check the OAuth redirect", comment.Body)
	require.Equal(t, taskID, comment.TaskID)
	require.Equal(t, userID, comment.UserID)

	own, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	_, err = service.List(context.Background(), otherOrganizationID, taskID)
	require.ErrorIs(t, err, taskdomain.ErrTaskNotFound)

	other, err := service.List(context.Background(), otherOrganizationID, otherTaskID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestCommentService_Create_MemberCanAdd(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	comment, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		commentdomain.CreateCommentRequest{Body: "Looks good"},
	)

	require.NoError(t, err)
	require.Equal(t, "Looks good", comment.Body)
}

func TestCommentService_Create_TaskNotFound(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		commentdomain.CreateCommentRequest{Body: "Nope"},
	)

	require.ErrorIs(t, err, taskdomain.ErrTaskNotFound)
}

func TestCommentService_Update_Author(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		commentdomain.CreateCommentRequest{Body: "First draft"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		userID,
		commentdomain.UpdateCommentRequest{Body: "Fixed draft"},
	)

	require.NoError(t, err)
	require.Equal(t, "Fixed draft", updated.Body)
}

func TestCommentService_Update_OtherMemberForbidden(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	authorID := seedUser(t, db, "ada@example.com")
	otherID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		authorID,
		commentdomain.CreateCommentRequest{Body: "First draft"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		created.ID,
		otherID,
		commentdomain.UpdateCommentRequest{Body: "Hijacked"},
	)

	require.ErrorIs(t, err, commentdomain.ErrForbidden)
}

func TestCommentService_Update_WrongOrgNotFound(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		commentdomain.CreateCommentRequest{Body: "First draft"},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		otherOrganizationID,
		created.ID,
		userID,
		commentdomain.UpdateCommentRequest{Body: "Nope"},
	)

	require.ErrorIs(t, err, commentdomain.ErrCommentNotFound)
}

func TestCommentService_Delete_Author(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		commentdomain.CreateCommentRequest{Body: "Remove me"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		created.ID,
		userID,
		orgdomain.RoleMember,
	)
	require.NoError(t, err)

	comments, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Empty(t, comments)
}

func TestCommentService_Delete_AdminCanDeleteOther(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	authorID := seedUser(t, db, "ada@example.com")
	adminID := seedUser(t, db, "olivia@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		authorID,
		commentdomain.CreateCommentRequest{Body: "Remove me"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		created.ID,
		adminID,
		orgdomain.RoleAdmin,
	)
	require.NoError(t, err)
}

func TestCommentService_Delete_MemberCannotDeleteOther(t *testing.T) {
	db, cleanup := setupCommentTestDatabase(t)
	defer cleanup()

	service := setupCommentService(db)
	authorID := seedUser(t, db, "ada@example.com")
	otherID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	created, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		authorID,
		commentdomain.CreateCommentRequest{Body: "Leave me"},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		created.ID,
		otherID,
		orgdomain.RoleMember,
	)

	require.ErrorIs(t, err, commentdomain.ErrForbidden)
}
