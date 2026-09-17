package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	filedomain "github.com/chuuch/gorest/internal/files/domain"
	filehandler "github.com/chuuch/gorest/internal/files/handler"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func testProjectID() uuid.UUID {
	return uuid.MustParse("55555555-5555-5555-5555-555555555555")
}

func testFileView() *filedomain.FileView {
	return &filedomain.FileView{
		File: &filedomain.File{
			ID:             uuid.MustParse("88888888-8888-8888-8888-888888888888"),
			OrganizationID: testOrganizationID(),
			ProjectID:      testProjectID(),
			UploadedBy:     testUserID(),
			Filename:       "spec.pdf",
			ContentType:    "application/pdf",
			Size:           2048,
			CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		UploadURL:   "http://minio/put",
		DownloadURL: "http://minio/get",
	}
}

func withSession(
	req *http.Request,
	organizationID, userID uuid.UUID,
	role string,
) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID, uuid.UUID) ([]*filedomain.FileView, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, orgdomain.Role, filedomain.CreateFileRequest) (*filedomain.FileView, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID, projectID uuid.UUID,
) ([]*filedomain.FileView, error) {
	return m.listFunc(organizationID, projectID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, projectID, userID uuid.UUID,
	actorRole orgdomain.Role,
	req filedomain.CreateFileRequest,
) (*filedomain.FileView, error) {
	return m.createFunc(organizationID, projectID, userID, actorRole, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	projectID := testProjectID()
	view := testFileView()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotProjectID uuid.UUID) ([]*filedomain.FileView, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, projectID, gotProjectID)
			return []*filedomain.FileView{view}, nil
		},
	}

	handler := filehandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/files",
		nil,
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, organizationID, testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []filedomain.FileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "spec.pdf", response[0].Filename)
	require.Equal(t, "http://minio/get", response[0].DownloadURL)
}

func TestHandler_List_ProjectNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*filedomain.FileView, error) {
			return nil, projectdomain.ErrProjectNotFound
		},
	}

	handler := filehandler.NewHandler(service)
	projectID := testProjectID()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/files",
		nil,
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, testOrganizationID(), testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	projectID := testProjectID()
	view := testFileView()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotProjectID, gotUserID uuid.UUID,
			actorRole orgdomain.Role,
			req filedomain.CreateFileRequest,
		) (*filedomain.FileView, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, projectID, gotProjectID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "spec.pdf", req.Filename)
			return view, nil
		},
	}

	handler := filehandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/files",
		bytes.NewBufferString(
			`{"filename":"spec.pdf","content_type":"application/pdf","size":2048}`,
		),
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, organizationID, userID, "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response filedomain.FileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "http://minio/put", response.UploadURL)
}

func TestHandler_Create_Forbidden(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			filedomain.CreateFileRequest,
		) (*filedomain.FileView, error) {
			return nil, filedomain.ErrForbidden
		},
	}

	handler := filehandler.NewHandler(service)
	projectID := testProjectID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/files",
		bytes.NewBufferString(
			`{"filename":"spec.pdf","content_type":"application/pdf","size":2048}`,
		),
	)
	req.SetPathValue("id", projectID.String())
	req = withSession(req, testOrganizationID(), testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Create_InvalidProjectID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			filedomain.CreateFileRequest,
		) (*filedomain.FileView, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := filehandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/not-a-uuid/files",
		bytes.NewBufferString(
			`{"filename":"spec.pdf","content_type":"application/pdf","size":2048}`,
		),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), testUserID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
