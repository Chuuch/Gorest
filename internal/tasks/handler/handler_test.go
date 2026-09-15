package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testProjectID() uuid.UUID {
	return uuid.MustParse("55555555-5555-5555-5555-555555555555")
}

func testTask() *taskdomain.Task {
	return &taskdomain.Task{
		ID:             uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		OrganizationID: testOrganizationID(),
		ProjectID:      testProjectID(),
		Title:          "Fix login",
		Notes:          "OAuth",
		Status:         taskdomain.StatusTodo,
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func withSession(req *http.Request, organizationID uuid.UUID, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID, uuid.UUID) ([]*taskdomain.Task, error)
	createFunc func(uuid.UUID, uuid.UUID, orgdomain.Role, taskdomain.CreateTaskRequest) (*taskdomain.Task, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
	projectID uuid.UUID,
) ([]*taskdomain.Task, error) {
	return m.listFunc(organizationID, projectID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID uuid.UUID,
	projectID uuid.UUID,
	actorRole orgdomain.Role,
	req taskdomain.CreateTaskRequest,
) (*taskdomain.Task, error) {
	return m.createFunc(organizationID, projectID, actorRole, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	projectID := testProjectID()
	task := testTask()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID, gotProjectID uuid.UUID) ([]*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, projectID, gotProjectID)
			return []*taskdomain.Task{task}, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/tasks",
		nil,
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []taskdomain.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Fix login", response[0].Title)
	require.Equal(t, taskdomain.StatusTodo, response[0].Status)
}

func TestHandler_List_ProjectNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*taskdomain.Task, error) {
			return nil, projectdomain.ErrProjectNotFound
		},
	}

	handler := taskhandler.NewHandler(service)
	projectID := testProjectID()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/tasks",
		nil,
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	projectID := testProjectID()
	task := testTask()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			gotProjectID uuid.UUID,
			actorRole orgdomain.Role,
			req taskdomain.CreateTaskRequest,
		) (*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, projectID, gotProjectID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "Fix login", req.Title)
			require.Equal(t, "todo", req.Status)
			return task, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/tasks",
		bytes.NewBufferString(`{"title":"Fix login","notes":"OAuth","status":"todo"}`),
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_Forbidden(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.CreateTaskRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrForbidden
		},
	}

	handler := taskhandler.NewHandler(service)
	projectID := testProjectID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/tasks",
		bytes.NewBufferString(`{"title":"Fix login","status":"todo"}`),
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Create_InvalidProjectID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.CreateTaskRequest,
		) (*taskdomain.Task, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/not-a-uuid/tasks",
		bytes.NewBufferString(`{"title":"Fix login","status":"todo"}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_InvalidStatus(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.CreateTaskRequest,
		) (*taskdomain.Task, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := taskhandler.NewHandler(service)
	projectID := testProjectID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/tasks",
		bytes.NewBufferString(`{"title":"Fix login","status":"blocked"}`),
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
