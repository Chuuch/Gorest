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
	clienthandler "github.com/chuuch/gorest/internal/client/handler"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testClient() *clientdomain.Client {
	return &clientdomain.Client{
		ID:             uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		OrganizationID: testOrganizationID(),
		Name:           "Northwind",
		Notes:          "Retail",
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
	listFunc   func(uuid.UUID) ([]*clientdomain.Client, error)
	createFunc func(uuid.UUID, orgdomain.Role, clientdomain.CreateClientRequest) (*clientdomain.Client, error)
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
) ([]*clientdomain.Client, error) {
	return m.listFunc(organizationID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID uuid.UUID,
	actorRole orgdomain.Role,
	req clientdomain.CreateClientRequest,
) (*clientdomain.Client, error) {
	return m.createFunc(organizationID, actorRole, req)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	client := testClient()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID) ([]*clientdomain.Client, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			return []*clientdomain.Client{client}, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	req = withSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []clientdomain.ClientResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, client.Name, response[0].Name)
	require.Equal(t, client.Notes, response[0].Notes)
}

func TestHandler_List_Empty(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID) ([]*clientdomain.Client, error) {
			return []*clientdomain.Client{}, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "[]", rec.Body.String())
}

func TestHandler_List_Unauthorized(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID) ([]*clientdomain.Client, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	client := testClient()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			actorRole orgdomain.Role,
			req clientdomain.CreateClientRequest,
		) (*clientdomain.Client, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "Northwind", req.Name)
			require.Equal(t, "Retail", req.Notes)
			return client, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	body := `{
        "name": "Northwind",
        "notes": "Retail"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients",
		bytes.NewBufferString(body),
	)
	req = withSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var response clientdomain.ClientResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "Northwind", response.Name)
}

func TestHandler_Create_Forbidden(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			clientdomain.CreateClientRequest,
		) (*clientdomain.Client, error) {
			return nil, clientdomain.ErrForbidden
		},
	}

	handler := clienthandler.NewHandler(service)

	body := `{"name":"Northwind"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients",
		bytes.NewBufferString(body),
	)
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Create_NameExists(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			clientdomain.CreateClientRequest,
		) (*clientdomain.Client, error) {
			return nil, clientdomain.ErrClientNameExists
		},
	}

	handler := clienthandler.NewHandler(service)

	body := `{"name":"Northwind"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients",
		bytes.NewBufferString(body),
	)
	req = withSession(req, testOrganizationID(), "admin")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			clientdomain.CreateClientRequest,
		) (*clientdomain.Client, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients",
		bytes.NewBufferString(`invalid-json`),
	)
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Create_Unauthorized(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			clientdomain.CreateClientRequest,
		) (*clientdomain.Client, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := clienthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients",
		bytes.NewBufferString(`{}`),
	)

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
