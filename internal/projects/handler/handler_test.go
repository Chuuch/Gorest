package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projecthandler "github.com/chuuch/gorest/internal/projects/handler"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testClientID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testProject() *projectdomain.Project {
	return &projectdomain.Project{
		ID:             uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		OrganizationID: testOrganizationID(),
		ClientID:       testClientID(),
		Name:           "Website",
		Notes:          "Launch",
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
	listFunc   func(uuid.UUID, uuid.UUID) ([]*projectdomain.Project, error)
	createFunc func(uuid.UUID, uuid.UUID, orgdomain.Role, projectdomain.CreateProjectRequest) (*projectdomain.Project, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
	clientID uuid.UUID,
) ([]*projectdomain.Project, error) {
	return m.listFunc(organizationID, clientID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID uuid.UUID,
	clientID uuid.UUID,
	actorRole orgdomain.Role,
	req projectdomain.CreateProjectRequest,
) (*projectdomain.Project, error) {
	return m.createFunc(organizationID, clientID, actorRole, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	project := testProject()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID, gotClientID uuid.UUID) ([]*projectdomain.Project, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			return []*projectdomain.Project{project}, nil
		},
	}

	handler := projecthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/clients/"+clientID.String()+"/projects",
		nil,
	)
	req.SetPathValue("id", clientID.String())
	req = withSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []projectdomain.ProjectResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Website", response[0].Name)
}

func TestHandler_List_ClientNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*projectdomain.Project, error) {
			return nil, clientdomain.ErrClientNotFound
		},
	}

	handler := projecthandler.NewHandler(service)
	clientID := testClientID()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/clients/"+clientID.String()+"/projects",
		nil,
	)
	req.SetPathValue("id", clientID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	project := testProject()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			gotClientID uuid.UUID,
			actorRole orgdomain.Role,
			req projectdomain.CreateProjectRequest,
		) (*projectdomain.Project, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "Website", req.Name)
			return project, nil
		},
	}

	handler := projecthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients/"+clientID.String()+"/projects",
		bytes.NewBufferString(`{"name":"Website","notes":"Launch"}`),
	)
	req.SetPathValue("id", clientID.String())
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
			projectdomain.CreateProjectRequest,
		) (*projectdomain.Project, error) {
			return nil, projectdomain.ErrForbidden
		},
	}

	handler := projecthandler.NewHandler(service)
	clientID := testClientID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients/"+clientID.String()+"/projects",
		bytes.NewBufferString(`{"name":"Website"}`),
	)
	req.SetPathValue("id", clientID.String())
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Create_InvalidClientID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			projectdomain.CreateProjectRequest,
		) (*projectdomain.Project, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := projecthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients/not-a-uuid/projects",
		bytes.NewBufferString(`{"name":"Website"}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
