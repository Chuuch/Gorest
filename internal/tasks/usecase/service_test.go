package usecase_test

import (
	"context"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectpostgres "github.com/chuuch/gorest/internal/projects/postgres"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	taskusecase "github.com/chuuch/gorest/internal/tasks/usecase"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketpostgres "github.com/chuuch/gorest/internal/tickets/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTaskTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
		CREATE TABLE tickets (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE tasks (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			ticket_id UUID REFERENCES tickets(id) ON DELETE SET NULL,
			title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			completed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT tasks_project_title_unique UNIQUE (project_id, title),
			CONSTRAINT tasks_status_check CHECK (status IN ('todo', 'in_progress', 'done'))
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE UNIQUE INDEX idx_tasks_project_ticket_unique
		ON tasks (project_id, ticket_id)
		WHERE ticket_id IS NOT NULL
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

func setupTaskService(db *pgxpool.Pool) taskusecase.Service {
	return taskusecase.NewService(
		taskpostgres.NewRepository(db),
		projectpostgres.NewRepository(db),
		ticketpostgres.NewRepository(db),
	)
}

func TestTaskService_CreateAndList(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "Website")

	task, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Notes:  "OAuth",
			Status: "todo",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Fix login", task.Title)
	require.Equal(t, taskdomain.StatusTodo, task.Status)
	require.Equal(t, projectID, task.ProjectID)

	own, err := service.List(context.Background(), organizationID, projectID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	_, err = service.List(context.Background(), otherOrganizationID, projectID)
	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)

	other, err := service.List(context.Background(), otherOrganizationID, otherProjectID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestTaskService_Create_AdminCanAdd(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	task, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleAdmin,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "in_progress",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Fix login", task.Title)
	require.Equal(t, taskdomain.StatusInProgress, task.Status)
}

func TestTaskService_Create_Forbidden(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleMember,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)

	require.ErrorIs(t, err, taskdomain.ErrForbidden)
}

func TestTaskService_Create_TitleExists(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)
	require.NoError(t, err)

	_, err = service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "done",
		},
	)

	require.ErrorIs(t, err, taskdomain.ErrTaskTitleExists)
}

func TestTaskService_Create_SameTitleDifferentProjects(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, organizationID, clientID, "App")

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)
	require.NoError(t, err)

	task, err := service.Create(
		context.Background(),
		organizationID,
		otherProjectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)

	require.NoError(t, err)
	require.Equal(t, otherProjectID, task.ProjectID)
}

func TestTaskService_Create_ProjectNotFound(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)

	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
}

func TestTaskService_Update_MemberCanComplete(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	created, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "todo",
		},
	)
	require.NoError(t, err)
	require.Nil(t, created.CompletedAt)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		taskdomain.UpdateTaskRequest{Status: "done"},
	)

	require.NoError(t, err)
	require.Equal(t, taskdomain.StatusDone, updated.Status)
	require.NotNil(t, updated.CompletedAt)
}

func TestTaskService_Update_ReopenClearsCompletedAt(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	created, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "done",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, created.CompletedAt)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		taskdomain.UpdateTaskRequest{Status: "in_progress"},
	)

	require.NoError(t, err)
	require.Equal(t, taskdomain.StatusInProgress, updated.Status)
	require.Nil(t, updated.CompletedAt)
}

func TestTaskService_Update_DoneTwiceKeepsCompletedAt(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	created, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Fix login",
			Status: "done",
		},
	)
	require.NoError(t, err)

	own, err := service.List(context.Background(), organizationID, projectID)
	require.NoError(t, err)
	require.NotNil(t, own[0].CompletedAt)
	firstCompletedAt := *own[0].CompletedAt

	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		taskdomain.UpdateTaskRequest{Status: "done"},
	)

	require.NoError(t, err)
	require.NotNil(t, updated.CompletedAt)
	require.True(t, updated.CompletedAt.Equal(firstCompletedAt))
}

func TestTaskService_Update_NotFound(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Update(
		context.Background(),
		organizationID,
		uuid.New(),
		taskdomain.UpdateTaskRequest{Status: "done"},
	)

	require.ErrorIs(t, err, taskdomain.ErrTaskNotFound)
}

func TestTaskService_Convert(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, organizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, organizationID, clientID, "App")
	otherClientProjectID := seedProject(t, db, organizationID, otherClientID, "Portal")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	task, err := service.Convert(
		context.Background(),
		organizationID,
		ticketID,
		orgdomain.RoleOwner,
		taskdomain.ConvertTicketRequest{ProjectID: projectID},
	)
	require.NoError(t, err)
	require.Equal(t, "Login button broken", task.Title)
	require.Equal(t, "Clicking Sign in does nothing on mobile.", task.Notes)
	require.Equal(t, taskdomain.StatusTodo, task.Status)
	require.Equal(t, projectID, task.ProjectID)
	require.NotNil(t, task.TicketID)
	require.Equal(t, ticketID, *task.TicketID)

	_, err = service.Convert(
		context.Background(),
		organizationID,
		ticketID,
		orgdomain.RoleOwner,
		taskdomain.ConvertTicketRequest{ProjectID: projectID},
	)
	require.ErrorIs(t, err, taskdomain.ErrTicketAlreadyConverted)

	second, err := service.Convert(
		context.Background(),
		organizationID,
		ticketID,
		orgdomain.RoleAdmin,
		taskdomain.ConvertTicketRequest{ProjectID: otherProjectID},
	)
	require.NoError(t, err)
	require.Equal(t, otherProjectID, second.ProjectID)

	_, err = service.Convert(
		context.Background(),
		organizationID,
		ticketID,
		orgdomain.RoleOwner,
		taskdomain.ConvertTicketRequest{ProjectID: otherClientProjectID},
	)
	require.ErrorIs(t, err, projectdomain.ErrProjectNotFound)
}

func TestTaskService_Convert_Forbidden(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	userID := seedUser(t, db, "pat@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	ticketID := seedTicket(t, db, organizationID, clientID, userID, "Login button broken")

	_, err := service.Convert(
		context.Background(),
		organizationID,
		ticketID,
		orgdomain.RoleMember,
		taskdomain.ConvertTicketRequest{ProjectID: projectID},
	)
	require.ErrorIs(t, err, taskdomain.ErrForbidden)
}

func TestTaskService_Convert_TicketNotFound(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")

	_, err := service.Convert(
		context.Background(),
		organizationID,
		uuid.New(),
		orgdomain.RoleOwner,
		taskdomain.ConvertTicketRequest{ProjectID: projectID},
	)
	require.ErrorIs(t, err, ticketdomain.ErrTicketNotFound)
}
