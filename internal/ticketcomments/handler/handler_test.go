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
	ticketcommentdomain "github.com/chuuch/gorest/internal/ticketcomments/domain"
	ticketcommenthandler "github.com/chuuch/gorest/internal/ticketcomments/handler"
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

func testComment() *ticketcommentdomain.Comment {
	return &ticketcommentdomain.Comment{
		ID:             uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		OrganizationID: testOrganizationID(),
		TicketID:       testTicketID(),
		UserID:         testUserID(),
		Body:           "Can you try another browser?",
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
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
	listFunc   func(uuid.UUID, uuid.UUID, uuid.UUID) ([]*ticketcommentdomain.Comment, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, ticketcommentdomain.CreateCommentRequest) (*ticketcommentdomain.Comment, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID, ticketID, portalClientID uuid.UUID,
) ([]*ticketcommentdomain.Comment, error) {
	return m.listFunc(organizationID, ticketID, portalClientID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
	req ticketcommentdomain.CreateCommentRequest,
) (*ticketcommentdomain.Comment, error) {
	return m.createFunc(organizationID, ticketID, userID, portalClientID, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	ticketID := testTicketID()
	comment := testComment()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotTicketID, portalClientID uuid.UUID) ([]*ticketcommentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, uuid.Nil, portalClientID)
			return []*ticketcommentdomain.Comment{comment}, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+ticketID.String()+"/comments", nil)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, organizationID, testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []ticketcommentdomain.CommentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Can you try another browser?", response[0].Body)
	require.Equal(t, ticketID, response[0].TicketID)
}

func TestHandler_ListPortal(t *testing.T) {
	organizationID := testOrganizationID()
	ticketID := testTicketID()
	clientID := testClientID()
	comment := testComment()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotTicketID, portalClientID uuid.UUID) ([]*ticketcommentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, clientID, portalClientID)
			return []*ticketcommentdomain.Comment{comment}, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client-auth/tickets/"+ticketID.String()+"/comments", nil)
	req.SetPathValue("id", ticketID.String())
	req = withPortalSession(req, organizationID, testUserID(), clientID)

	rec := httptest.NewRecorder()
	handler.ListPortal(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_List_TicketNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID, uuid.UUID) ([]*ticketcommentdomain.Comment, error) {
			return nil, ticketdomain.ErrTicketNotFound
		},
	}

	handler := ticketcommenthandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+ticketID.String()+"/comments", nil)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	ticketID := testTicketID()
	comment := testComment()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotTicketID, gotUserID, portalClientID uuid.UUID,
			req ticketcommentdomain.CreateCommentRequest,
		) (*ticketcommentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, uuid.Nil, portalClientID)
			require.Equal(t, "Can you try another browser?", req.Body)
			return comment, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/comments",
		bytes.NewBufferString(`{"body":"Can you try another browser?"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, organizationID, userID)

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_CreatePortal(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	ticketID := testTicketID()
	clientID := testClientID()
	comment := testComment()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotTicketID, gotUserID, portalClientID uuid.UUID,
			req ticketcommentdomain.CreateCommentRequest,
		) (*ticketcommentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticketID, gotTicketID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, clientID, portalClientID)
			require.Equal(t, "Still broken on Safari", req.Body)
			return comment, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/tickets/"+ticketID.String()+"/comments",
		bytes.NewBufferString(`{"body":"Still broken on Safari"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withPortalSession(req, organizationID, userID, clientID)

	rec := httptest.NewRecorder()
	handler.CreatePortal(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			ticketcommentdomain.CreateCommentRequest,
		) (*ticketcommentdomain.Comment, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)
	ticketID := testTicketID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID.String()+"/comments",
		bytes.NewBufferString(`{"body":""}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_InvalidTicketID(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			ticketcommentdomain.CreateCommentRequest,
		) (*ticketcommentdomain.Comment, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := ticketcommenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/not-a-uuid/comments",
		bytes.NewBufferString(`{"body":"Nope"}`),
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withStaffSession(req, testOrganizationID(), testUserID())

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
