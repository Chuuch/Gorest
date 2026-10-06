package handler_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/notifications"
	"github.com/chuuch/gorest/internal/notifications/domain"
	notificationhandler "github.com/chuuch/gorest/internal/notifications/handler"
	"github.com/chuuch/gorest/internal/pagination"
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

func withSession(req *http.Request, organizationID, userID uuid.UUID) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc     func(uuid.UUID, uuid.UUID, int, *pagination.Cursor) ([]*domain.Notification, *string, error)
	markReadFunc func(uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (m *mockService) Publish(context.Context, notifications.Message) error {
	return nil
}

func (m *mockService) List(
	_ context.Context,
	organizationID, recipientID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
) ([]*domain.Notification, *string, error) {
	return m.listFunc(organizationID, recipientID, limit, cursor)
}

func (m *mockService) MarkRead(
	_ context.Context,
	organizationID, recipientID, id uuid.UUID,
) error {
	return m.markReadFunc(organizationID, recipientID, id)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	itemID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	service := &mockService{
		listFunc: func(gotOrg, gotUser uuid.UUID, limit int, cursor *pagination.Cursor) ([]*domain.Notification, *string, error) {
			require.Equal(t, organizationID, gotOrg)
			require.Equal(t, userID, gotUser)
			require.Equal(t, 50, limit)
			require.Nil(t, cursor)
			return []*domain.Notification{{
				ID:             itemID,
				OrganizationID: organizationID,
				RecipientID:    userID,
				ActorID:        uuid.MustParse("66666666-6666-6666-6666-666666666666"),
				ActorEmail:     "pat@example.com",
				Kind:           domain.KindTicketOpened,
				EntityType:     domain.EntityTicket,
				EntityID:       uuid.MustParse("77777777-7777-7777-7777-777777777777"),
				Summary:        "Login broken",
				CreatedAt:      time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
			}}, nil, nil
		},
	}

	handler := notificationhandler.NewHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req = withSession(req, organizationID, userID)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response pagination.Page[domain.NotificationResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Items, 1)
	require.Nil(t, response.NextCursor)
	require.Equal(t, domain.KindTicketOpened, response.Items[0].Kind)
	require.Equal(t, "Login broken", response.Items[0].Summary)
}

func TestHandler_List_InvalidCursor(t *testing.T) {
	handler := notificationhandler.NewHandler(&mockService{
		listFunc: func(uuid.UUID, uuid.UUID, int, *pagination.Cursor) ([]*domain.Notification, *string, error) {
			t.Fatal("service should not be called")
			return nil, nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?cursor=not-a-cursor", nil)
	req = withSession(req, testOrganizationID(), testUserID())
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_MarkRead_NotFound(t *testing.T) {
	service := &mockService{
		markReadFunc: func(uuid.UUID, uuid.UUID, uuid.UUID) error {
			return domain.ErrNotificationNotFound
		},
	}

	handler := notificationhandler.NewHandler(service)
	id := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+id.String()+"/read", nil)
	req.SetPathValue("id", id.String())
	req = withSession(req, testOrganizationID(), testUserID())
	rec := httptest.NewRecorder()
	handler.MarkRead(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
