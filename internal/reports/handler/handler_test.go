package handler_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/requestcontext"
	reportsdomain "github.com/chuuch/gorest/internal/reports/domain"
	reporthandler "github.com/chuuch/gorest/internal/reports/handler"
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
	return uuid.MustParse("33333333-3333-3333-3333-333333333333")
}

func testProjectID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testReport() *reportsdomain.TimeReport {
	return &reportsdomain.TimeReport{
		From:         time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		To:           time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		TotalMinutes: 90,
		ByClient: []reportsdomain.ClientRow{
			{ClientID: testClientID(), ClientName: "Northwind", Minutes: 90},
		},
		ByProject: []reportsdomain.ProjectRow{
			{
				ProjectID:   testProjectID(),
				ProjectName: "Portal",
				ClientID:    testClientID(),
				ClientName:  "Northwind",
				Minutes:     90,
			},
		},
		ByMember: []reportsdomain.MemberRow{
			{
				UserID:      testUserID(),
				Email:       "ada@example.com",
				DisplayName: "Ada",
				Minutes:     90,
			},
		},
	}
}

func withActor(req *http.Request, organizationID, userID uuid.UUID, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), organizationID)
	ctx = requestcontext.WithUserID(ctx, userID)
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	timeFunc func(uuid.UUID, uuid.UUID, orgdomain.Role, time.Time, time.Time) (*reportsdomain.TimeReport, error)
}

func (m *mockService) Time(
	_ context.Context,
	organizationID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*reportsdomain.TimeReport, error) {
	return m.timeFunc(organizationID, actorUserID, actorRole, from, to)
}

func TestHandler_Time(t *testing.T) {
	organizationID := testOrganizationID()
	userID := testUserID()
	report := testReport()
	from := report.From
	to := report.To

	service := &mockService{
		timeFunc: func(
			gotOrganizationID, gotUserID uuid.UUID,
			actorRole orgdomain.Role,
			gotFrom, gotTo time.Time,
		) (*reportsdomain.TimeReport, error) {
			require.Equal(t, organizationID, gotOrganizationID)
			require.Equal(t, userID, gotUserID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.True(t, from.Equal(gotFrom))
			require.True(t, to.Equal(gotTo))
			return report, nil
		},
	}

	handler := reporthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/reports/time?from=2026-09-28T00:00:00.000Z&to=2026-10-05T00:00:00.000Z",
		nil,
	)
	req = withActor(req, organizationID, userID, "owner")

	rec := httptest.NewRecorder()
	handler.Time(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response reportsdomain.TimeReportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, 90, response.TotalMinutes)
	require.Len(t, response.ByClient, 1)
	require.Equal(t, "Northwind", response.ByClient[0].ClientName)
	require.Equal(t, "Ada", response.ByMember[0].DisplayName)
}

func TestHandler_Time_InvalidRange(t *testing.T) {
	service := &mockService{
		timeFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			time.Time,
			time.Time,
		) (*reportsdomain.TimeReport, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := reporthandler.NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/reports/time?from=nope&to=2026-10-05T00:00:00Z",
		nil,
	)
	req = withActor(req, testOrganizationID(), testUserID(), "owner")

	rec := httptest.NewRecorder()
	handler.Time(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Time_Unauthorized(t *testing.T) {
	service := &mockService{
		timeFunc: func(
			uuid.UUID,
			uuid.UUID,
			orgdomain.Role,
			time.Time,
			time.Time,
		) (*reportsdomain.TimeReport, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}

	handler := reporthandler.NewHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/time", nil)
	rec := httptest.NewRecorder()
	handler.Time(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
