package usecase_test

import (
	"context"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	reportspostgres "github.com/chuuch/gorest/internal/reports/postgres"
	reportsusecase "github.com/chuuch/gorest/internal/reports/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupReportTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
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
			display_name TEXT NOT NULL DEFAULT '',
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
			estimate_run_id UUID,
			estimated_hours NUMERIC,
			target_end_date DATE,
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
			updated_at TIMESTAMPTZ NOT NULL
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
			updated_at TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return db, cleanup
}

func seedUser(t *testing.T, db *pgxpool.Pool, email, displayName string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`,
		userID,
		email,
		displayName,
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

func seedClient(t *testing.T, db *pgxpool.Pool, organizationID uuid.UUID, name string) uuid.UUID {
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
			INSERT INTO projects (id, organization_id, client_id, name, notes, created_at, updated_at)
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
			INSERT INTO tasks (id, organization_id, project_id, title, notes, status, created_at, updated_at)
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

func seedTimeEntry(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, taskID, userID uuid.UUID,
	minutes int,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO time_entries (
				id,
				organization_id,
				task_id,
				user_id,
				minutes,
				notes,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		uuid.New(),
		organizationID,
		taskID,
		userID,
		minutes,
		"",
		createdAt,
		createdAt,
	)
	require.NoError(t, err)
}

func TestTimeReport_OwnerSeesAll(t *testing.T) {
	db, cleanup := setupReportTestDatabase(t)
	defer cleanup()

	service := reportsusecase.NewService(reportspostgres.NewRepository(db))
	adaID := seedUser(t, db, "ada@example.com", "Ada")
	linusID := seedUser(t, db, "linus@example.com", "")
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Ghost")
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "Nope")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")
	otherTaskID := seedTask(t, db, otherOrganizationID, otherProjectID, "Skip")

	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	seedTimeEntry(t, db, organizationID, taskID, adaID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))
	seedTimeEntry(t, db, organizationID, taskID, linusID, 30, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	seedTimeEntry(t, db, organizationID, taskID, adaID, 15, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	seedTimeEntry(t, db, otherOrganizationID, otherTaskID, adaID, 40, time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC))

	report, err := service.Time(context.Background(), organizationID, adaID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)
	require.Equal(t, 120, report.TotalMinutes)
	require.Len(t, report.ByClient, 1)
	require.Equal(t, "Northwind", report.ByClient[0].ClientName)
	require.Equal(t, 120, report.ByClient[0].Minutes)
	require.Len(t, report.ByProject, 1)
	require.Equal(t, "Portal", report.ByProject[0].ProjectName)
	require.Equal(t, "Northwind", report.ByProject[0].ClientName)
	require.Equal(t, 120, report.ByProject[0].Minutes)
	require.Len(t, report.ByMember, 2)
	require.Equal(t, "ada@example.com", report.ByMember[0].Email)
	require.Equal(t, "Ada", report.ByMember[0].DisplayName)
	require.Equal(t, 90, report.ByMember[0].Minutes)
	require.Equal(t, "linus@example.com", report.ByMember[1].Email)
	require.Equal(t, 30, report.ByMember[1].Minutes)
}

func TestTimeReport_MemberSeesOwn(t *testing.T) {
	db, cleanup := setupReportTestDatabase(t)
	defer cleanup()

	service := reportsusecase.NewService(reportspostgres.NewRepository(db))
	adaID := seedUser(t, db, "ada@example.com", "Ada")
	linusID := seedUser(t, db, "linus@example.com", "")
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Portal")
	taskID := seedTask(t, db, organizationID, projectID, "Draw")

	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	seedTimeEntry(t, db, organizationID, taskID, adaID, 90, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))
	seedTimeEntry(t, db, organizationID, taskID, linusID, 30, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))

	report, err := service.Time(context.Background(), organizationID, linusID, orgdomain.RoleMember, from, to)
	require.NoError(t, err)
	require.Equal(t, 30, report.TotalMinutes)
	require.Len(t, report.ByMember, 1)
	require.Equal(t, linusID, report.ByMember[0].UserID)
	require.Equal(t, 30, report.ByMember[0].Minutes)
}

func TestTimeReport_Empty(t *testing.T) {
	db, cleanup := setupReportTestDatabase(t)
	defer cleanup()

	service := reportsusecase.NewService(reportspostgres.NewRepository(db))
	userID := seedUser(t, db, "ada@example.com", "Ada")
	organizationID := seedOrganization(t, db, "Acme")
	from := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	report, err := service.Time(context.Background(), organizationID, userID, orgdomain.RoleOwner, from, to)
	require.NoError(t, err)
	require.Equal(t, 0, report.TotalMinutes)
	require.Empty(t, report.ByClient)
	require.Empty(t, report.ByProject)
	require.Empty(t, report.ByMember)
}
