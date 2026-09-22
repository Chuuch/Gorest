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
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testProjectID() uuid.UUID {
	return uuid.MustParse("55555555-5555-5555-5555-555555555555")
}

func testTicketID() uuid.UUID {
	return uuid.MustParse("99999999-9999-9999-9999-999999999999")
}

func testTask() *taskdomain.Task {
	return &taskdomain.Task{
		ID:             uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		OrganizationID: testOrganizationID(),
		ProjectID:      testProjectID(),
		Title:          "Fix login",
		Notes:          "OAuth",
		Status:         taskdomain.StatusTodo,
		Version:        1,
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
	listFunc    func(uuid.UUID, uuid.UUID) ([]*taskdomain.Task, error)
	createFunc  func(uuid.UUID, uuid.UUID, orgdomain.Role, taskdomain.CreateTaskRequest) (*taskdomain.Task, error)
	updateFunc  func(uuid.UUID, uuid.UUID, taskdomain.UpdateTaskRequest) (*taskdomain.Task, error)
	convertFunc func(uuid.UUID, uuid.UUID, orgdomain.Role, taskdomain.ConvertTicketRequest) (*taskdomain.Task, error)
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

func (m *mockService) Update(
	_ context.Context,
	organizationID uuid.UUID,
	taskID uuid.UUID,
	req taskdomain.UpdateTaskRequest,
) (*taskdomain.Task, error) {
	return m.updateFunc(organizationID, taskID, req)
}

func (m *mockService) Convert(
	_ context.Context,
	organizationID uuid.UUID,
	ticketID uuid.UUID,
	actorRole orgdomain.Role,
	req taskdomain.ConvertTicketRequest,
) (*taskdomain.Task, error) {
	return m.convertFunc(organizationID, ticketID, actorRole, req)
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
	require.Equal(t, 1, response[0].Version)
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

func TestHandler_Update(t *testing.T) {
	organizationID := testOrganizationID()
	task := testTask()
	task.Status = taskdomain.StatusDone
	task.Version = 2

	service := &mockService{
		updateFunc: func(
			gotOrganizationID uuid.UUID,
			gotTaskID uuid.UUID,
			req taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, task.ID, gotTaskID)
			require.Equal(t, "done", req.Status)
			require.Equal(t, 1, req.Version)
			return task, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+task.ID.String(),
		bytes.NewBufferString(`{"status":"done","version":1}`),
	)
	req.SetPathValue("id", task.ID.String())
	req = withSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response taskdomain.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, 2, response.Version)
}

func TestHandler_Update_NotFound(t *testing.T) {
	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrTaskNotFound
		},
	}

	handler := taskhandler.NewHandler(service)
	taskID := testTask().ID

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+taskID.String(),
		bytes.NewBufferString(`{"status":"done","version":1}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Update_VersionMismatch(t *testing.T) {
	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrTaskVersionMismatch
		},
	}

	handler := taskhandler.NewHandler(service)
	taskID := testTask().ID

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+taskID.String(),
		bytes.NewBufferString(`{"status":"done","version":1}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Update_MissingVersion(t *testing.T) {
	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := taskhandler.NewHandler(service)
	taskID := testTask().ID

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+taskID.String(),
		bytes.NewBufferString(`{"status":"done"}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Update_InvalidStatus(t *testing.T) {
	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := taskhandler.NewHandler(service)
	taskID := testTask().ID

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+taskID.String(),
		bytes.NewBufferString(`{"status":"blocked","version":1}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Convert(t *testing.T) {
	organizationID := testOrganizationID()
	projectID := testProjectID()
	ticketID := testTicketID()
	task := testTask()
	task.TicketID = &ticketID
	task.Notes = "Clicking Sign in does nothing on mobile."

	service := &mockService{
		convertFunc: func(
			gotOrganizationID uuid.UUID,
			gotTicketID uuid.UUID,
			actorRole orgdomain.Role,
			req taskdomain.ConvertTicketRequest,
		) (*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, projectID, req.ProjectID)
			return task, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/convert",
		bytes.NewBufferString(`{"project_id":"`+projectID.String()+`"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response taskdomain.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, ticketID, *response.TicketID)
	require.Equal(t, "Fix login", response.Title)
	require.Equal(t, 1, response.Version)
}

func TestHandler_Convert_Forbidden(t *testing.T) {
	service := &mockService{
		convertFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.ConvertTicketRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrForbidden
		},
	}

	handler := taskhandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/convert",
		bytes.NewBufferString(`{"project_id":"`+testProjectID().String()+`"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Convert_InvalidTicketID(t *testing.T) {
	service := &mockService{
		convertFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.ConvertTicketRequest,
		) (*taskdomain.Task, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/not-a-uuid/convert",
		bytes.NewBufferString(`{"project_id":"`+testProjectID().String()+`"}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Convert_AlreadyConverted(t *testing.T) {
	service := &mockService{
		convertFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.ConvertTicketRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrTicketAlreadyConverted
		},
	}

	handler := taskhandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/convert",
		bytes.NewBufferString(`{"project_id":"`+testProjectID().String()+`"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Convert_TicketNotFound(t *testing.T) {
	service := &mockService{
		convertFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			taskdomain.ConvertTicketRequest,
		) (*taskdomain.Task, error) {
			return nil, ticketdomain.ErrTicketNotFound
		},
	}

	handler := taskhandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/convert",
		bytes.NewBufferString(`{"project_id":"`+testProjectID().String()+`"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
