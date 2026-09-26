package usecase_test

import (
	"context"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func seedMember(
	t *testing.T,
	db *pgxpool.Pool,
	organizationID, userID uuid.UUID,
	role orgdomain.Role,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
			INSERT INTO memberships (id, organization_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		uuid.New(),
		organizationID,
		userID,
		role,
		time.Now().UTC(),
	)
	require.NoError(t, err)
}

func withActor(ctx context.Context, userID uuid.UUID) context.Context {
	return requestcontext.WithUserID(ctx, userID)
}

func TestTaskService_Create_SetsCreatedByAndAssignee(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	actorID := seedUser(t, db, "ada@example.com")
	assigneeID := seedUser(t, db, "ben@example.com")
	seedMember(t, db, organizationID, actorID, orgdomain.RoleOwner)
	seedMember(t, db, organizationID, assigneeID, orgdomain.RoleMember)

	task, err := service.Create(
		withActor(context.Background(), actorID),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:      "Fix login",
			Notes:      "OAuth",
			Status:     "todo",
			AssigneeID: &assigneeID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, &actorID, task.CreatedBy)
	require.Equal(t, &assigneeID, task.AssigneeID)
}

func TestTaskService_Create_AssigneeNotMember(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	outsiderID := seedUser(t, db, "pat@example.com")
	seedMember(t, db, otherOrganizationID, outsiderID, orgdomain.RoleMember)

	_, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:      "Fix login",
			Status:     "todo",
			AssigneeID: &outsiderID,
		},
	)

	require.ErrorIs(t, err, taskdomain.ErrAssigneeNotMember)
}

func TestTaskService_Update_TitleNotesAssignee(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	clientID := seedClient(t, db, organizationID, "Northwind")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	assigneeID := seedUser(t, db, "ben@example.com")
	seedMember(t, db, organizationID, assigneeID, orgdomain.RoleMember)

	created, err := service.Create(
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

	notes := "New notes"
	updated, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		taskdomain.UpdateTaskRequest{
			Title:      "Ship site",
			Notes:      &notes,
			Status:     "in_progress",
			AssigneeID: taskdomain.OptionalAssignee{Set: true, Value: &assigneeID},
			Version:    created.Version,
		},
	)

	require.NoError(t, err)
	require.Equal(t, "Ship site", updated.Title)
	require.Equal(t, "New notes", updated.Notes)
	require.Equal(t, &assigneeID, updated.AssigneeID)
	require.Equal(t, 2, updated.Version)

	cleared, err := service.Update(
		context.Background(),
		organizationID,
		created.ID,
		taskdomain.UpdateTaskRequest{
			Status:     "todo",
			AssigneeID: taskdomain.OptionalAssignee{Set: true, Value: nil},
			Version:    updated.Version,
		},
	)
	require.NoError(t, err)
	require.Nil(t, cleared.AssigneeID)
	require.Equal(t, "Ship site", cleared.Title)
}

func TestTaskService_Inbox(t *testing.T) {
	db, cleanup := setupTaskTestDatabase(t)
	defer cleanup()

	service := setupTaskService(db)
	organizationID := seedOrganization(t, db, "Acme")
	otherOrganizationID := seedOrganization(t, db, "Other")
	clientID := seedClient(t, db, organizationID, "Northwind")
	otherClientID := seedClient(t, db, otherOrganizationID, "Contoso")
	projectID := seedProject(t, db, organizationID, clientID, "Website")
	otherProjectID := seedProject(t, db, otherOrganizationID, otherClientID, "App")
	me := seedUser(t, db, "ada@example.com")
	other := seedUser(t, db, "ben@example.com")
	seedMember(t, db, organizationID, me, orgdomain.RoleMember)
	seedMember(t, db, organizationID, other, orgdomain.RoleMember)

	mine, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:      "Mine",
			Status:     "todo",
			AssigneeID: &me,
		},
	)
	require.NoError(t, err)

	open, err := service.Create(
		context.Background(),
		organizationID,
		projectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Open",
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
			Title:      "Theirs",
			Status:     "todo",
			AssigneeID: &other,
		},
	)
	require.NoError(t, err)

	_, err = service.Create(
		context.Background(),
		otherOrganizationID,
		otherProjectID,
		orgdomain.RoleOwner,
		taskdomain.CreateTaskRequest{
			Title:  "Other org",
			Status: "todo",
		},
	)
	require.NoError(t, err)

	inbox, err := service.Inbox(context.Background(), organizationID, me)
	require.NoError(t, err)
	require.Len(t, inbox, 2)
	require.Equal(t, mine.ID, inbox[0].ID)
	require.Equal(t, open.ID, inbox[1].ID)
}
