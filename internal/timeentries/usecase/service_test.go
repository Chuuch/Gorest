package usecase_test

import (
	"context"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskpostgres "github.com/chuuch/gorest/internal/tasks/postgres"
	timeentrydomain "github.com/chuuch/gorest/internal/timeentries/domain"
	timeentrypostgres "github.com/chuuch/gorest/internal/timeentries/postgres"
	timeentryusecase "github.com/chuuch/gorest/internal/timeentries/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTimeEntryTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			created_by UUID REFERENCES users(id) ON DELETE RESTRICT,
			assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT tasks_project_title_unique UNIQUE (project_id, title),
			CONSTRAINT tasks_status_check CHECK (status IN ('todo', 'in_progress', 'done'))
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE time_entries (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			minutes INTEGER NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT time_entries_minutes_check CHECK (minutes >= 1 AND minutes <= 1440)
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
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		taskID,
		organizationID,
		projectID,
		title,
		"",
		"todo",
		now,
		now,
	)
	require.NoError(t, err)

	return taskID
}

func setupTimeEntryService(db *pgxpool.Pool) timeentryusecase.Service {
	return timeentryusecase.NewService(
		timeentrypostgres.NewRepository(db),
		taskpostgres.NewRepository(db),
	)
}

func TestTimeEntryService_CreateAndList(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")
	otherTaskID := seedTask(t, db, otherOrganizationID, otherProjectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{
			Minutes: 90,
			Notes:   "OAuth",
		},
	)

	require.NoError(t, err)
	require.Equal(t, 90, entry.Minutes)
	require.Equal(t, taskID, entry.TaskID)
	require.Equal(t, userID, entry.UserID)

	own, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Len(t, own, 1)

	_, err = service.List(context.Background(), otherOrganizationID, taskID)
	require.ErrorIs(t, err, taskdomain.ErrTaskNotFound)

	other, err := service.List(context.Background(), otherOrganizationID, otherTaskID)
	require.NoError(t, err)
	require.Empty(t, other)
}

func TestTimeEntryService_Create_MemberCanAdd(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 30},
	)

	require.NoError(t, err)
	require.Equal(t, 30, entry.Minutes)
}

func TestTimeEntryService_Create_MultipleEntries(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	_, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 30},
	)
	require.NoError(t, err)

	_, err = service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 45},
	)
	require.NoError(t, err)

	entries, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestTimeEntryService_Create_TaskNotFound(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Create(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 30},
	)

	require.ErrorIs(t, err, taskdomain.ErrTaskNotFound)
}

func TestTimeEntryService_Update(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90, Notes: "OAuth"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		entry.ID,
		userID,
		orgdomain.RoleMember,
		timeentrydomain.UpdateTimeEntryRequest{Minutes: 45, Notes: "SSO"},
	)
	require.NoError(t, err)
	require.Equal(t, 45, updated.Minutes)
	require.Equal(t, "SSO", updated.Notes)

	listed, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Equal(t, 45, listed[0].Minutes)
}

func TestTimeEntryService_Update_AdminCanEditOthers(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	ownerID := seedUser(t, db, "ada@example.com")
	adminID := seedUser(t, db, "linus@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		ownerID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90, Notes: "OAuth"},
	)
	require.NoError(t, err)

	updated, err := service.Update(
		context.Background(),
		organizationID,
		entry.ID,
		adminID,
		orgdomain.RoleAdmin,
		timeentrydomain.UpdateTimeEntryRequest{Minutes: 60, Notes: "Review"},
	)
	require.NoError(t, err)
	require.Equal(t, 60, updated.Minutes)
}

func TestTimeEntryService_Update_MemberForbiddenOnOthers(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	ownerID := seedUser(t, db, "ada@example.com")
	memberID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		ownerID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90},
	)
	require.NoError(t, err)

	_, err = service.Update(
		context.Background(),
		organizationID,
		entry.ID,
		memberID,
		orgdomain.RoleMember,
		timeentrydomain.UpdateTimeEntryRequest{Minutes: 15},
	)
	require.ErrorIs(t, err, timeentrydomain.ErrForbidden)
}

func TestTimeEntryService_Update_NotFound(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")

	_, err := service.Update(
		context.Background(),
		organizationID,
		uuid.New(),
		userID,
		orgdomain.RoleOwner,
		timeentrydomain.UpdateTimeEntryRequest{Minutes: 30},
	)
	require.ErrorIs(t, err, timeentrydomain.ErrTimeEntryNotFound)
}

func TestTimeEntryService_Delete(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		entry.ID,
		userID,
		orgdomain.RoleMember,
	)
	require.NoError(t, err)

	listed, err := service.List(context.Background(), organizationID, taskID)
	require.NoError(t, err)
	require.Empty(t, listed)
}

func TestTimeEntryService_Delete_MemberForbiddenOnOthers(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	ownerID := seedUser(t, db, "ada@example.com")
	memberID := seedUser(t, db, "mike@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		ownerID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		organizationID,
		entry.ID,
		memberID,
		orgdomain.RoleMember,
	)
	require.ErrorIs(t, err, timeentrydomain.ErrForbidden)
}

func TestTimeEntryService_Delete_WrongOrg(t *testing.T) {
	db, cleanup := setupTimeEntryTestDatabase(t)
	defer cleanup()

	service := setupTimeEntryService(db)
	userID := seedUser(t, db, "ada@example.com")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	taskID := seedTask(t, db, organizationID, projectID, "Fix login")

	entry, err := service.Create(
		context.Background(),
		organizationID,
		taskID,
		userID,
		timeentrydomain.CreateTimeEntryRequest{Minutes: 90},
	)
	require.NoError(t, err)

	err = service.Delete(
		context.Background(),
		otherOrganizationID,
		entry.ID,
		userID,
		orgdomain.RoleOwner,
	)
	require.ErrorIs(t, err, timeentrydomain.ErrTimeEntryNotFound)
}
