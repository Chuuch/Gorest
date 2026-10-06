package handler_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	activitydomain "github.com/chuuch/gorest/internal/activity/domain"
	activityhandler "github.com/chuuch/gorest/internal/activity/handler"
	"github.com/chuuch/gorest/internal/pagination"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-4222-8222-222222222222")
}

type mockService struct {
	listFn func(uuid.UUID, int, *pagination.Cursor) ([]*activitydomain.Event, *string, error)
}

func (m *mockService) Record(context.Context, activitydomain.Event) error {
	return nil
}

func (m *mockService) List(
	_ context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
) ([]*activitydomain.Event, *string, error) {
	return m.listFn(organizationID, limit, cursor)
}

func TestHandler_List(t *testing.T) {
	organizationID := testOrganizationID()
	event := &activitydomain.Event{
		ID:               uuid.MustParse("33333333-3333-4333-8333-333333333333"),
		OrganizationID:   organizationID,
		ActorID:          uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		ActorEmail:       "ada@example.com",
		ActorDisplayName: "Ada",
		Action:           activitydomain.ActionCreated,
		EntityType:       activitydomain.EntityTask,
		EntityID:         uuid.MustParse("44444444-4444-8444-8444-444444444444"),
		Summary:          "Draw wireframes",
		CreatedAt:        time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	service := &mockService{
		listFn: func(gotOrganizationID uuid.UUID, limit int, cursor *pagination.Cursor) ([]*activitydomain.Event, *string, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, 50, limit)
			require.Nil(t, cursor)
			return []*activitydomain.Event{event}, nil, nil
		},
	}

	handler := activityhandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activity", nil)
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), organizationID))

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response pagination.Page[activitydomain.EventResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Items, 1)
	require.Nil(t, response.NextCursor)
	require.Equal(t, event.ID, response.Items[0].ID)
	require.Equal(t, "ada@example.com", response.Items[0].ActorEmail)
	require.Equal(t, "Ada", response.Items[0].ActorDisplayName)
	require.Equal(t, "Draw wireframes", response.Items[0].Summary)
}

func TestHandler_List_Empty(t *testing.T) {
	service := &mockService{
		listFn: func(uuid.UUID, int, *pagination.Cursor) ([]*activitydomain.Event, *string, error) {
			return []*activitydomain.Event{}, nil, nil
		},
	}

	handler := activityhandler.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activity", nil)
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), testOrganizationID()))

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response pagination.Page[activitydomain.EventResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Empty(t, response.Items)
	require.Nil(t, response.NextCursor)
}

func TestHandler_List_Unauthorized(t *testing.T) {
	handler := activityhandler.NewHandler(&mockService{
		listFn: func(uuid.UUID, int, *pagination.Cursor) ([]*activitydomain.Event, *string, error) {
			t.Fatal("service should not be called")
			return nil, nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activity", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_List_InvalidLimit(t *testing.T) {
	handler := activityhandler.NewHandler(&mockService{
		listFn: func(uuid.UUID, int, *pagination.Cursor) ([]*activitydomain.Event, *string, error) {
			t.Fatal("service should not be called")
			return nil, nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activity?limit=nope", nil)
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), testOrganizationID()))

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_List_InvalidCursor(t *testing.T) {
	handler := activityhandler.NewHandler(&mockService{
		listFn: func(uuid.UUID, int, *pagination.Cursor) ([]*activitydomain.Event, *string, error) {
			t.Fatal("service should not be called")
			return nil, nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activity?cursor=not-a-cursor", nil)
	req = req.WithContext(requestcontext.WithOrganizationID(req.Context(), testOrganizationID()))

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
