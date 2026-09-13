package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orghandler "github.com/chuuch/gorest/internal/organization/handler"
	orgusecase "github.com/chuuch/gorest/internal/organization/usecase"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testMember() *orgusecase.Member {
	return &orgusecase.Member{
		UserID:    uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Email:     "ada@example.com",
		Role:      orgdomain.RoleMember,
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func withSession(req *http.Request, organizationID uuid.UUID, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc   func(uuid.UUID) ([]orgusecase.Member, error)
	createFunc func(uuid.UUID, orgdomain.Role, orgdomain.CreateMemberRequest) (*orgusecase.Member, error)
}

func (m *mockService) ListMembers(
	_ context.Context,
	organizationID uuid.UUID,
) ([]orgusecase.Member, error) {
	return m.listFunc(organizationID)
}

func (m *mockService) CreateMember(
	_ context.Context,
	organizationID uuid.UUID,
	actorRole orgdomain.Role,
	req orgdomain.CreateMemberRequest,
) (*orgusecase.Member, error) {
	return m.createFunc(organizationID, actorRole, req)
}

func TestHandler_ListMembers(t *testing.T) {
	organizationID := testOrganizationID()
	member := testMember()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID) ([]orgusecase.Member, error) {
			require.Equal(t, organizationID, gotOrganizationID)

			return []orgusecase.Member{*member}, nil
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/members",
		nil,
	)
	req = withSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()

	handler.ListMembers(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response []orgdomain.MemberResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Len(t, response, 1)
	require.Equal(t, member.UserID, response[0].UserID)
	require.Equal(t, member.Email, response[0].Email)
	require.Equal(t, orgdomain.RoleMember, response[0].Role)
	require.Equal(t, member.CreatedAt, response[0].CreatedAt)
}

func TestHandler_ListMembers_Empty(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID) ([]orgusecase.Member, error) {
			return []orgusecase.Member{}, nil
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/members",
		nil,
	)
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()

	handler.ListMembers(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "[]", rec.Body.String())
}

func TestHandler_ListMembers_Unauthorized(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID) ([]orgusecase.Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/members",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ListMembers(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_ListMembers_InternalError(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID) ([]orgusecase.Member, error) {
			return nil, errors.New("database failure")
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/members",
		nil,
	)
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()

	handler.ListMembers(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_CreateMember(t *testing.T) {
	organizationID := testOrganizationID()
	member := testMember()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			actorRole orgdomain.Role,
			req orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.Equal(t, "ada@example.com", req.Email)
			require.Equal(t, "password123", req.Password)
			require.Equal(t, "member", req.Role)

			return member, nil
		},
	}

	handler := orghandler.NewHandler(service)

	body := `{
        "email": "ada@example.com",
        "password": "password123",
        "role": "member"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(body),
	)
	req = withSession(req, organizationID, "owner")

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response orgdomain.MemberResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Equal(t, member.UserID, response.UserID)
	require.Equal(t, member.Email, response.Email)
	require.Equal(t, orgdomain.RoleMember, response.Role)
}

func TestHandler_CreateMember_Forbidden(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			return nil, orgdomain.ErrForbidden
		},
	}

	handler := orghandler.NewHandler(service)

	body := `{
        "email": "ada@example.com",
        "password": "password123",
        "role": "member"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(body),
	)
	req = withSession(req, testOrganizationID(), "member")

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_CreateMember_AlreadyExists(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			return nil, orgdomain.ErrMemberAlreadyExists
		},
	}

	handler := orghandler.NewHandler(service)

	body := `{
        "email": "ada@example.com",
        "password": "password123",
        "role": "admin"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(body),
	)
	req = withSession(req, testOrganizationID(), "admin")

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_CreateMember_CannotCreateOwner(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := orghandler.NewHandler(service)

	body := `{
        "email": "ada@example.com",
        "password": "password123",
        "role": "owner"
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(body),
	)
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateMember_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(`invalid-json`),
	)
	req = withSession(req, testOrganizationID(), "owner")

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CreateMember_Unauthorized(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			orgdomain.Role,
			orgdomain.CreateMemberRequest,
		) (*orgusecase.Member, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := orghandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/members",
		bytes.NewBufferString(`{}`),
	)

	rec := httptest.NewRecorder()

	handler.CreateMember(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
