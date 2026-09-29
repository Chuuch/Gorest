package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskhandler "github.com/chuuch/gorest/internal/tasks/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func (m *mockService) Inbox(
	_ context.Context,
	organizationID, userID uuid.UUID,
) ([]*taskdomain.Task, error) {
	return nil, nil
}

type inboxService struct {
	mockService
	fn func(uuid.UUID, uuid.UUID) ([]*taskdomain.Task, error)
}

func (s *inboxService) Inbox(
	_ context.Context,
	organizationID, userID uuid.UUID,
) ([]*taskdomain.Task, error) {
	return s.fn(organizationID, userID)
}

func withActor(req *http.Request, organizationID, userID uuid.UUID, role string) *http.Request {
	req = withSession(req, organizationID, role)
	return req.WithContext(requestcontext.WithUserID(req.Context(), userID))
}

func TestHandler_Inbox(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	task := testTask()

	service := &inboxService{
		fn: func(gotOrganizationID, gotUserID uuid.UUID) ([]*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, userID, gotUserID)
			return []*taskdomain.Task{task}, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbox/tasks", nil)
	req = withActor(req, organizationID, userID, "member")

	rec := httptest.NewRecorder()
	handler.Inbox(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []taskdomain.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Fix login", response[0].Title)
}

func TestHandler_Inbox_Unauthorized(t *testing.T) {
	handler := taskhandler.NewHandler(&mockService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbox/tasks", nil)
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Inbox(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Update_TitleAndAssignee(t *testing.T) {
	organizationID := testOrganizationID()
	assigneeID := testUserID()
	task := testTask()
	task.Title = "Ship site"
	task.AssigneeID = &assigneeID
	task.Version = 2

	service := &mockService{
		updateFunc: func(
			gotOrganizationID uuid.UUID,
			gotTaskID uuid.UUID,
			req taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, task.ID, gotTaskID)
			require.Equal(t, "Ship site", req.Title)
			require.True(t, req.AssigneeID.Set)
			require.Equal(t, &assigneeID, req.AssigneeID.Value)
			return task, nil
		},
	}

	handler := taskhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+task.ID.String(),
		bytes.NewBufferString(
			`{"title":"Ship site","status":"todo","assignee_id":"`+assigneeID.String()+`","version":1}`,
		),
	)
	req.SetPathValue("id", task.ID.String())
	req = withSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response taskdomain.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "Ship site", response.Title)
	require.Equal(t, &assigneeID, response.AssigneeID)
}

func TestHandler_Update_AssigneeNotMember(t *testing.T) {
	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			taskdomain.UpdateTaskRequest,
		) (*taskdomain.Task, error) {
			return nil, taskdomain.ErrAssigneeNotMember
		},
	}

	handler := taskhandler.NewHandler(service)
	taskID := testTask().ID

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tasks/"+taskID.String(),
		bytes.NewBufferString(`{"status":"todo","assignee_id":"`+testUserID().String()+`","version":1}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
