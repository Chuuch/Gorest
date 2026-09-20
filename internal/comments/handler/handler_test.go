package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commentdomain "github.com/chuuch/gorest/internal/comments/domain"
	commenthandler "github.com/chuuch/gorest/internal/comments/handler"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
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

func testComment() *commentdomain.Comment {
	return &commentdomain.Comment{
		ID:             uuid.MustParse("88888888-8888-8888-8888-888888888888"),
		OrganizationID: testOrganizationID(),
		TaskID:         testTaskID(),
		UserID:         testUserID(),
		Body:           "Check the OAuth redirect",
		CreatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
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
	listFunc   func(uuid.UUID, uuid.UUID) ([]*commentdomain.Comment, error)
	createFunc func(uuid.UUID, uuid.UUID, uuid.UUID, commentdomain.CreateCommentRequest) (*commentdomain.Comment, error)
	updateFunc func(uuid.UUID, uuid.UUID, uuid.UUID, commentdomain.UpdateCommentRequest) (*commentdomain.Comment, error)
	deleteFunc func(uuid.UUID, uuid.UUID, uuid.UUID, orgdomain.Role) error
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
	taskID uuid.UUID,
) ([]*commentdomain.Comment, error) {
	return m.listFunc(organizationID, taskID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID uuid.UUID,
	taskID uuid.UUID,
	userID uuid.UUID,
	req commentdomain.CreateCommentRequest,
) (*commentdomain.Comment, error) {
	return m.createFunc(organizationID, taskID, userID, req)
}

func (m *mockService) Update(
	_ context.Context,
	organizationID uuid.UUID,
	commentID uuid.UUID,
	actorID uuid.UUID,
	req commentdomain.UpdateCommentRequest,
) (*commentdomain.Comment, error) {
	return m.updateFunc(organizationID, commentID, actorID, req)
}

func (m *mockService) Delete(
	_ context.Context,
	organizationID uuid.UUID,
	commentID uuid.UUID,
	actorID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	return m.deleteFunc(organizationID, commentID, actorID, actorRole)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	taskID := testTaskID()
	comment := testComment()

	service := &mockService{
		listFunc: func(gotOrganizationID uuid.UUID, gotTaskID uuid.UUID) ([]*commentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, taskID, gotTaskID)
			return []*commentdomain.Comment{comment}, nil
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/"+taskID.String()+"/comments",
		nil,
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, organizationID, testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response []commentdomain.CommentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Check the OAuth redirect", response[0].Body)
}

func TestHandler_List_TaskNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*commentdomain.Comment, error) {
			return nil, taskdomain.ErrTaskNotFound
		},
	}

	handler := commenthandler.NewHandler(service)
	taskID := testTaskID()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/"+taskID.String()+"/comments",
		nil,
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	taskID := testTaskID()
	comment := testComment()

	service := &mockService{
		createFunc: func(
			gotOrganizationID uuid.UUID,
			gotTaskID uuid.UUID,
			gotUserID uuid.UUID,
			req commentdomain.CreateCommentRequest,
		) (*commentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, taskID, gotTaskID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, "Check the OAuth redirect", req.Body)
			return comment, nil
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID.String()+"/comments",
		bytes.NewBufferString(`{"body":"Check the OAuth redirect"}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, organizationID, userID, "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_InvalidBody(t *testing.T) {
	service := &mockService{
		createFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			commentdomain.CreateCommentRequest,
		) (*commentdomain.Comment, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := commenthandler.NewHandler(service)
	taskID := testTaskID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID.String()+"/comments",
		bytes.NewBufferString(`{"body":""}`),
	)
	req.SetPathValue("id", taskID.String())
	req = withSession(req, testOrganizationID(), testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Update(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	comment := testComment()
	comment.Body = "Fixed draft"

	service := &mockService{
		updateFunc: func(
			gotOrganizationID uuid.UUID,
			gotCommentID uuid.UUID,
			gotActorID uuid.UUID,
			req commentdomain.UpdateCommentRequest,
		) (*commentdomain.Comment, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, comment.ID, gotCommentID)
			require.Equal(t, userID, gotActorID)
			require.Equal(t, "Fixed draft", req.Body)
			return comment, nil
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/comments/"+comment.ID.String(),
		bytes.NewBufferString(`{"body":"Fixed draft"}`),
	)
	req.SetPathValue("id", comment.ID.String())
	req = withSession(req, organizationID, userID, "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response commentdomain.CommentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "Fixed draft", response.Body)
}

func TestHandler_Update_Forbidden(t *testing.T) {
	comment := testComment()

	service := &mockService{
		updateFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			commentdomain.UpdateCommentRequest,
		) (*commentdomain.Comment, error) {
			return nil, commentdomain.ErrForbidden
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/comments/"+comment.ID.String(),
		bytes.NewBufferString(`{"body":"Hijacked"}`),
	)
	req.SetPathValue("id", comment.ID.String())
	req = withSession(req, testOrganizationID(), uuid.New(), "member")

	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_Delete(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	comment := testComment()

	service := &mockService{
		deleteFunc: func(
			gotOrganizationID uuid.UUID,
			gotCommentID uuid.UUID,
			gotActorID uuid.UUID,
			actorRole orgdomain.Role,
		) error {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, comment.ID, gotCommentID)
			require.Equal(t, userID, gotActorID)
			require.Equal(t, orgdomain.RoleMember, actorRole)
			return nil
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/comments/"+comment.ID.String(),
		nil,
	)
	req.SetPathValue("id", comment.ID.String())
	req = withSession(req, organizationID, userID, "member")

	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_Delete_InvalidCommentID(t *testing.T) {
	service := &mockService{
		deleteFunc: func(
			uuid.UUID,
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
		) error {
			t.Fatal("service should not be called")
			return nil
		},
	}

	handler := commenthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/comments/not-a-uuid",
		nil,
	)
	req.SetPathValue("id", "not-a-uuid")
	req = withSession(req, testOrganizationID(), testUserID(), "member")

	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
