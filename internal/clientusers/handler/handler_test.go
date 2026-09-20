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
	clientuserdomain "github.com/chuuch/gorest/internal/clientusers/domain"
	clientuserhandler "github.com/chuuch/gorest/internal/clientusers/handler"
	clientuserusecase "github.com/chuuch/gorest/internal/clientusers/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testClientID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testMember() clientuserusecase.Member {
	return clientuserusecase.Member{
		UserID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Email:          "pat@northwind.test",
		OrganizationID: testOrganizationID(),
		ClientID:       testClientID(),
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func withStaffSession(req *http.Request, organizationID uuid.UUID, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID, uuid.UUID) ([]clientuserusecase.Member, error)
	createFunc func(uuid.UUID, uuid.UUID, orgdomain.Role, clientuserdomain.CreateClientUserRequest) (*clientuserusecase.Member, error)
	loginFunc  func(clientuserdomain.LoginRequest) (*clientuserusecase.AuthResult, error)
}

func (m *mockService) List(_ context.Context, organizationID, clientID uuid.UUID) ([]clientuserusecase.Member, error) {
	return m.listFunc(organizationID, clientID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	req clientuserdomain.CreateClientUserRequest,
) (*clientuserusecase.Member, error) {
	return m.createFunc(organizationID, clientID, actorRole, req)
}

func (m *mockService) Login(_ context.Context, req clientuserdomain.LoginRequest) (*clientuserusecase.AuthResult, error) {
	return m.loginFunc(req)
}

func (m *mockService) Refresh(context.Context, string) (*clientuserusecase.AuthResult, error) {
	panic("unused")
}

func (m *mockService) Logout(context.Context, string) error {
	panic("unused")
}

func (m *mockService) Me(context.Context, uuid.UUID) (*clientuserusecase.AuthResult, error) {
	panic("unused")
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	member := testMember()

	service := &mockService{
		listFunc: func(gotOrganizationID, gotClientID uuid.UUID) ([]clientuserusecase.Member, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			return []clientuserusecase.Member{member}, nil
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+clientID.String()+"/users", nil)
	req.SetPathValue("id", clientID.String())
	req = withStaffSession(req, organizationID, "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []clientuserdomain.ClientUserResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "pat@northwind.test", response[0].Email)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	clientID := testClientID()
	member := testMember()

	service := &mockService{
		createFunc: func(
			gotOrganizationID, gotClientID uuid.UUID,
			actorRole orgdomain.Role,
			req clientuserdomain.CreateClientUserRequest,
		) (*clientuserusecase.Member, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, clientID, gotClientID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "pat@northwind.test", req.Email)
			return &member, nil
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients/"+clientID.String()+"/users",
		bytes.NewBufferString(`{"email":"pat@northwind.test","password":"password123"}`),
	)
	req.SetPathValue("id", clientID.String())
	req = withStaffSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			clientuserdomain.CreateClientUserRequest,
		) (*clientuserusecase.Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)
	clientID := testClientID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/clients/"+clientID.String()+"/users",
		bytes.NewBufferString(`{"email":"pat","password":"short"}`),
	)
	req.SetPathValue("id", clientID.String())
	req = withStaffSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Login(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	clientID := testClientID()
	organizationID := testOrganizationID()

	service := &mockService{
		loginFunc: func(req clientuserdomain.LoginRequest) (*clientuserusecase.AuthResult, error) {
			require.Equal(t, "pat@northwind.test", req.Email)
			return &clientuserusecase.AuthResult{
				AccessToken:  "token",
				RefreshToken: "refresh",
				User: &userdomain.User{
					ID:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
					Email:     req.Email,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Organization: &orgdomain.Organization{
					ID:        organizationID,
					Name:      "Acme",
					CreatedAt: now,
					UpdatedAt: now,
				},
				Client: &clientdomain.Client{
					ID:             clientID,
					OrganizationID: organizationID,
					Name:           "Northwind",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Role: clientuserdomain.RoleClient,
			}, nil
		},
	}

	handler := clientuserhandler.NewHandler(service, time.Hour, false)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/client-auth/login",
		bytes.NewBufferString(`{"email":"pat@northwind.test","password":"password123"}`),
	)

	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response clientuserdomain.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "client", response.Role)
	require.Equal(t, clientID, response.Client.ID)
	require.Equal(t, "refresh", rec.Result().Cookies()[0].Value)
}
