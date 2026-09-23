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
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	timeentrydomain "github.com/chuuch/gorest/internal/timeentries/domain"
	timeentryhandler "github.com/chuuch/gorest/internal/timeentries/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func testTaskID() uuid.UUID {
	return uuid.MustParse("66666666-6666-6666-6666-666666666666")
}

func testTimeEntry() *timeentrydomain.TimeEntry {
	return &timeentrydomain.TimeEntry{
		ID:             uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		OrganizationID: testOrganizationID(),
		TaskID:         testTaskID(),
		UserID:         testUserID(),
		Minutes:        90,
		Notes:          "OAuth",
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func withSession(req *http.Request, organizationID, userID uuid.UUID) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID, uuid.UUID) ([]*timeentrydomain.TimeEntry, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, timeentrydomain.CreateTimeEntryRequest) (*timeentrydomain.TimeEntry, error)
	updateFunc func(uuid.UUID, uuid.UUID, uuid.UUID, orgdomain.Role, timeentrydomain.UpdateTimeEntryRequest) (*timeentrydomain.TimeEntry, error)
	deleteFunc func(uuid.UUID, uuid.UUID, uuid.UUID, orgdomain.Role) error
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
	taskID uuid.UUID,
) ([]*timeentrydomain.TimeEntry, error) {
	return m.listFunc(organizationID, taskID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID uuid.UUID,
	taskID uuid.UUID,
	userID uuid.UUID,
	req timeentrydomain.CreateTimeEntryRequest,
) (*timeentrydomain.TimeEntry, error) {
	return m.createFunc(organizationID, taskID, userID, req)
}

func (m *mockService) Update(
	_ context.Context,
	organizationID, entryID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
	req timeentrydomain.UpdateTimeEntryRequest,
) (*timeentrydomain.TimeEntry, error) {
	return m.updateFunc(organizationID, entryID, actorUserID, actorRole, req)
}

func (m *mockService) Delete(
	_ context.Context,
	organizationID, entryID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	return m.deleteFunc(organizationID, entryID, actorUserID, actorRole)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	taskID := testTaskID()
	entry := testTimeEntry()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID, gotTaskID uuid.UUID) ([]*timeentrydomain.TimeEntry, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, taskID, gotTaskID)
			return []*timeentrydomain.TimeEntry{entry}, nil
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/"+taskID.String()+"/time-entries",
		nil,
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, organizationID, testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []timeentrydomain.TimeEntryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, 90, response[0].Minutes)
}

func TestHandler_List_TaskNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*timeentrydomain.TimeEntry, error) {
			return nil, taskdomain.ErrTaskNotFound
		},
	}

	handler := timeentryhandler.NewHandler(service)
	taskID := testTaskID()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/"+taskID.String()+"/time-entries",
		nil,
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	taskID := testTaskID()
	entry := testTimeEntry()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			gotTaskID uuid.UUID,
			gotUserID uuid.UUID,
			req timeentrydomain.CreateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, taskID, gotTaskID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, 90, req.Minutes)
			require.Equal(t, "OAuth", req.Notes)
			return entry, nil
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID.String()+"/time-entries",
		bytes.NewBufferString(`{"minutes":90,"notes":"OAuth"}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, organizationID, userID)

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_InvalidTaskID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			timeentrydomain.CreateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/not-a-uuid/time-entries",
		bytes.NewBufferString(`{"minutes":90}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_InvalidMinutes(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			timeentrydomain.CreateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := timeentryhandler.NewHandler(service)
	taskID := testTaskID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID.String()+"/time-entries",
		bytes.NewBufferString(`{"minutes":0}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func withActor(req *http.Request, organizationID, userID uuid.UUID, role string) *http.Request {
	req = withSession(req, organizationID, userID)
	ctx := requestcontext.WithRole(req.Context(), role)
	return req.WithContext(ctx)
}

func TestHandler_Update(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	entry := testTimeEntry()

	service := &mockService{
		updateFunc: func(
			gotOrganizationID, gotEntryID, gotUserID uuid.UUID,
			actorRole orgdomain.Role,
			req timeentrydomain.UpdateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, entry.ID, gotEntryID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, orgdomain.RoleMember, actorRole)
			require.Equal(t, 45, req.Minutes)
			require.Equal(t, "SSO", req.Notes)
			updated := *entry
			updated.Minutes = req.Minutes
			updated.Notes = req.Notes
			return &updated, nil
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/time-entries/"+entry.ID.String(),
		bytes.NewBufferString(`{"minutes":45,"notes":"SSO"}`),
	)
	req.SetPathValue("id", entry.ID.String())
	req = withActor(req, organizationID, userID, "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response timeentrydomain.TimeEntryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, 45, response.Minutes)
}

func TestHandler_Update_Forbidden(t *testing.T) {
	entry := testTimeEntry()

	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			timeentrydomain.UpdateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			return nil, timeentrydomain.ErrForbidden
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/time-entries/"+entry.ID.String(),
		bytes.NewBufferString(`{"minutes":15}`),
	)
	req.SetPathValue("id", entry.ID.String())
	req = withActor(req, testOrganizationID(), uuid.New(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Update_NotFound(t *testing.T) {
	entryID := uuid.New()

	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			timeentrydomain.UpdateTimeEntryRequest,
		) (*timeentrydomain.TimeEntry, error) {
			return nil, timeentrydomain.ErrTimeEntryNotFound
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/time-entries/"+entryID.String(),
		bytes.NewBufferString(`{"minutes":15}`),
	)
	req.SetPathValue("id", entryID.String())
	req = withActor(req, testOrganizationID(), testUserID(), "owner")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Delete(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	entry := testTimeEntry()

	service := &mockService{
		deleteFunc: func(
			gotOrganizationID, gotEntryID, gotUserID uuid.UUID,
			actorRole orgdomain.Role,
		) error {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, entry.ID, gotEntryID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, orgdomain.RoleAdmin, actorRole)
			return nil
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/time-entries/"+entry.ID.String(),
		nil,
	)
	req.SetPathValue("id", entry.ID.String())
	req = withActor(req, organizationID, userID, "admin")

	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_Delete_Forbidden(t *testing.T) {
	entry := testTimeEntry()

	service := &mockService{
		deleteFunc: func(uuid.UUID, uuid.UUID, uuid.UUID, orgdomain.Role) error {
			return timeentrydomain.ErrForbidden
		},
	}

	handler := timeentryhandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/time-entries/"+entry.ID.String(),
		nil,
	)
	req.SetPathValue("id", entry.ID.String())
	req = withActor(req, testOrganizationID(), uuid.New(), "member")

	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}
