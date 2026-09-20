package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/requestcontext"
	ticketfiledomain "github.com/chuuch/gorest/internal/ticketfiles/domain"
	ticketfilehandler "github.com/chuuch/gorest/internal/ticketfiles/handler"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func testClientID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testTicketID() uuid.UUID {
	return uuid.MustParse("99999999-9999-9999-9999-999999999999")
}

func testFileView() *ticketfiledomain.FileView {
	return &ticketfiledomain.FileView{
		File: &ticketfiledomain.File{
			ID:             uuid.MustParse("88888888-8888-8888-8888-888888888888"),
			OrganizationID: testOrganizationID(),
			TicketID:       testTicketID(),
			UploadedBy:     testUserID(),
			Filename:       "bug.png",
			ContentType:    "image/png",
			Size:           2048,
			CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		UploadURL:   "http://minio/put",
		DownloadURL: "http://minio/get",
	}
}

func withStaffSession(req *http.Request, organizationID, userID uuid.UUID) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	ctx = requestcontext.WithRole(ctx, "member")
	return req.WithContext(ctx)
}

func withPortalSession(
	req *http.Request,
	organizationID, userID, clientID uuid.UUID,
) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	ctx = requestcontext.WithClientID(ctx, clientID)
	ctx = requestcontext.WithRole(ctx, "client")
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID, uuid.UUID, uuid.UUID) ([]*ticketfiledomain.FileView, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, ticketfiledomain.CreateFileRequest) (*ticketfiledomain.FileView, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID, ticketID, portalClientID uuid.UUID,
) ([]*ticketfiledomain.FileView, error) {
	return m.listFunc(organizationID, ticketID, portalClientID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
	req ticketfiledomain.CreateFileRequest,
) (*ticketfiledomain.FileView, error) {
	return m.createFunc(organizationID, ticketID, userID, portalClientID, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	ticketID := testTicketID()
	view := testFileView()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotTicketID, portalClientID uuid.UUID) ([]*ticketfiledomain.FileView, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, uuid.Nil, portalClientID)
			return []*ticketfiledomain.FileView{view}, nil
		},
	}

	handler := ticketfilehandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+ticketID.String()+"/files", nil)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, organizationID, testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []ticketfiledomain.FileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "bug.png", response[0].Filename)
	require.Equal(t, ticketID, response[0].TicketID)
	require.Equal(t, "http://minio/get", response[0].DownloadURL)
}

func TestHandler_ListPortal(t *testing.T) {
	organizationID := testOrganizationID()
	ticketID := testTicketID()
	clientID := testClientID()
	view := testFileView()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotTicketID, portalClientID uuid.UUID) ([]*ticketfiledomain.FileView, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, clientID, portalClientID)
			return []*ticketfiledomain.FileView{view}, nil
		},
	}

	handler := ticketfilehandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client-auth/tickets/"+ticketID.String()+"/files", nil)
	req.SetPathValue("id", ticketID.String())
	req = withPortalSession(req, organizationID, testUserID(), clientID)

	rec := httptest.NewRecorder()
	handler.ListPortal(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_List_TicketNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID, uuid.UUID) ([]*ticketfiledomain.FileView, error) {
			return nil, ticketdomain.ErrTicketNotFound
		},
	}

	handler := ticketfilehandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+ticketID.String()+"/files", nil)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_CreatePortal(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	ticketID := testTicketID()
	clientID := testClientID()
	view := testFileView()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotTicketID, gotUserID, portalClientID uuid.UUID,
			req ticketfiledomain.CreateFileRequest,
		) (*ticketfiledomain.FileView, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, clientID, portalClientID)
			require.Equal(t, "bug.png", req.Filename)
			return view, nil
		},
	}

	handler := ticketfilehandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/tickets/"+ticketID.String()+"/files",
		bytes.NewBufferString(`{"filename":"bug.png","content_type":"image/png","size":2048}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withPortalSession(req, organizationID, userID, clientID)

	rec := httptest.NewRecorder()
	handler.CreatePortal(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response ticketfiledomain.FileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "http://minio/put", response.UploadURL)
}

func TestHandler_Create_InvalidTicketID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			ticketfiledomain.CreateFileRequest,
		) (*ticketfiledomain.FileView, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := ticketfilehandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/not-a-uuid/files",
		bytes.NewBufferString(`{"filename":"bug.png","content_type":"image/png","size":2048}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_UnsupportedType(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			ticketfiledomain.CreateFileRequest,
		) (*ticketfiledomain.FileView, error) {
			return nil, ticketfiledomain.ErrUnsupportedContentType
		},
	}

	handler := ticketfilehandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/files",
		bytes.NewBufferString(`{"filename":"virus.exe","content_type":"application/octet-stream","size":2048}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
