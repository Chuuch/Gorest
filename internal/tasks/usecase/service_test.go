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
			title TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT tasks_project_title_unique UNIQUE (project_id, title),
			CONSTRAINT tasks_status_check CHECK (status IN ('todo', 'in_progress', 'done'))
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
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

func setupTaskService(db *pgxpool.Pool) taskusecase.Service {
	return taskusecase.NewService(
		taskpostgres.NewRepository(db),
		projectpostgres.NewRepository(db),
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
