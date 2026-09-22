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
	"github.com/chuuch/gorest/internal/requestcontext"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	tickethandler "github.com/chuuch/gorest/internal/tickets/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testClientID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func testTicket() *ticketdomain.Ticket {
	return &ticketdomain.Ticket{
		ID:             uuid.MustParse("99999999-9999-9999-9999-999999999999"),
		OrganizationID: testOrganizationID(),
		ClientID:       testClientID(),
		UserID:         testUserID(),
		Kind:           ticketdomain.KindBug,
		Status:         ticketdomain.StatusOpen,
		Title:          "Login button broken",
		Body:           "Clicking Sign in does nothing on mobile.",
		Version:        1,
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func withStaffSession(req *http.Request, organizationID uuid.UUID, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, testUserID())
	ctx = requestcontext.WithRole(ctx, role)
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
	listFunc   func(uuid.UUID, uuid.UUID) ([]*ticketdomain.Ticket, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, ticketdomain.CreateTicketRequest) (*ticketdomain.Ticket, error)
	updateFunc func(uuid.UUID, uuid.UUID, ticketdomain.UpdateTicketRequest) (*ticketdomain.Ticket, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID, clientID uuid.UUID,
) ([]*ticketdomain.Ticket, error) {
	return m.listFunc(organizationID, clientID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, clientID, userID uuid.UUID,
	req ticketdomain.CreateTicketRequest,
) (*ticketdomain.Ticket, error) {
	return m.createFunc(organizationID, clientID, userID, req)
}

func (m *mockService) Update(
	_ context.Context,
	organizationID, ticketID uuid.UUID,
	req ticketdomain.UpdateTicketRequest,
) (*ticketdomain.Ticket, error) {
	return m.updateFunc(organizationID, ticketID, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	ticket := testTicket()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotClientID uuid.UUID) ([]*ticketdomain.Ticket, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			return []*ticketdomain.Ticket{ticket}, nil
		},
	}

	handler := tickethandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+clientID.String()+"/tickets", nil)
	req.SetPathValue("id", clientID.String())
	req = withStaffSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []ticketdomain.TicketResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, ticketdomain.KindBug, response[0].Kind)
	require.Equal(t, "Login button broken", response[0].Title)
	require.Equal(t, 1, response[0].Version)
}

func TestHandler_List_ClientNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*ticketdomain.Ticket, error) {
			return nil, clientdomain.ErrClientNotFound
		},
	}

	handler := tickethandler.NewHandler(service)
	clientID := testClientID()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+clientID.String()+"/tickets", nil)
	req.SetPathValue("id", clientID.String())
	req = withStaffSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ListPortal(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	userID := testUserID()
	ticket := testTicket()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotClientID uuid.UUID) ([]*ticketdomain.Ticket, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			return []*ticketdomain.Ticket{ticket}, nil
		},
	}

	handler := tickethandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client-auth/tickets", nil)
	req = withPortalSession(req, organizationID, userID, clientID)

	rec := httptest.NewRecorder()
	handler.ListPortal(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []ticketdomain.TicketResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, ticketdomain.StatusOpen, response[0].Status)
	require.Equal(t, 1, response[0].Version)
}

func TestHandler_CreatePortal(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	userID := testUserID()
	ticket := testTicket()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotClientID, gotUserID uuid.UUID,
			req ticketdomain.CreateTicketRequest,
		) (*ticketdomain.Ticket, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, "bug", req.Kind)
			require.Equal(t, "Login button broken", req.Title)
			return ticket, nil
		},
	}

	handler := tickethandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/tickets",
		bytes.NewBufferString(`{"kind":"bug","title":"Login button broken","body":"Clicking Sign in does nothing on mobile."}`),
	)
	req = withPortalSession(req, organizationID, userID, clientID)

	rec := httptest.NewRecorder()
	handler.CreatePortal(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_CreatePortal_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			ticketdomain.CreateTicketRequest,
		) (*ticketdomain.Ticket, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := tickethandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/tickets",
		bytes.NewBufferString(`{"kind":"bug","title":"Hey","body":""}`),
	)
	req = withPortalSession(req, testOrganizationID(), testUserID(), testClientID())

	rec := httptest.NewRecorder()
	handler.CreatePortal(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Update(t *testing.T) {
	organizationID := testOrganizationID()
	ticket := testTicket()
	ticket.Status = ticketdomain.StatusInProgress
	ticket.Version = 2

	service := &mockService{
		updateFunc: func(
			gotOrganizationID, gotTicketID uuid.UUID,
			req ticketdomain.UpdateTicketRequest,
		) (*ticketdomain.Ticket, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, ticket.ID, gotTicketID)
			require.Equal(t, "in_progress", req.Status)
			require.Equal(t, 1, req.Version)
			return ticket, nil
		},
	}

	handler := tickethandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+ticket.ID.String(),
		bytes.NewBufferString(`{"status":"in_progress","version":1}`),
	)
	req.SetPathValue("id", ticket.ID.String())
	req = withStaffSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response ticketdomain.TicketResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, ticketdomain.StatusInProgress, response.Status)
	require.Equal(t, 2, response.Version)
}

func TestHandler_Update_NotFound(t *testing.T) {
	service := &mockService{
		updateFunc: func(uuid.UUID, uuid.UUID, ticketdomain.UpdateTicketRequest) (*ticketdomain.Ticket, error) {
			return nil, ticketdomain.ErrTicketNotFound
		},
	}

	handler := tickethandler.NewHandler(service)
	ticketID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+ticketID.String(),
		bytes.NewBufferString(`{"status":"closed","version":1}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), "admin")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Update_VersionMismatch(t *testing.T) {
	service := &mockService{
		updateFunc: func(uuid.UUID, uuid.UUID, ticketdomain.UpdateTicketRequest) (*ticketdomain.Ticket, error) {
			return nil, ticketdomain.ErrTicketVersionMismatch
		},
	}

	handler := tickethandler.NewHandler(service)
	ticketID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+ticketID.String(),
		bytes.NewBufferString(`{"status":"closed","version":1}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), "admin")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Update_MissingVersion(t *testing.T) {
	service := &mockService{
		updateFunc: func(uuid.UUID, uuid.UUID, ticketdomain.UpdateTicketRequest) (*ticketdomain.Ticket, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := tickethandler.NewHandler(service)
	ticketID := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+ticketID.String(),
		bytes.NewBufferString(`{"status":"closed"}`),
	)
	req.SetPathValue("id", ticketID.String())
	req = withStaffSession(req, testOrganizationID(), "admin")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
