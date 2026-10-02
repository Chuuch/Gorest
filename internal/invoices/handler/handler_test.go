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
	invoicedomain "github.com/chuuch/gorest/internal/invoices/domain"
	invoicehandler "github.com/chuuch/gorest/internal/invoices/handler"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOrganizationID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func testClientID() uuid.UUID {
	return uuid.MustParse("44444444-4444-4444-4444-444444444444")
}

func testInvoiceID() uuid.UUID {
	return uuid.MustParse("55555555-5555-5555-5555-555555555555")
}

func testInvoice() *invoicedomain.Invoice {
	return &invoicedomain.Invoice{
		ID:               testInvoiceID(),
		OrganizationID:   testOrganizationID(),
		ClientID:         testClientID(),
		Number:           "INV-2026-0001",
		Status:           invoicedomain.StatusDraft,
		Currency:         invoicedomain.CurrencyEUR,
		RateCents:        3000,
		OrganizationName: "Acme",
		ClientName:       "Northwind",
		PeriodFrom:       time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		PeriodTo:         time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		IssuedAt:         time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		DueAt:            time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
		TotalMinutes:     90,
		TotalCents:       4500,
		CreatedAt:        time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		Lines: []invoicedomain.LineItem{
			{
				ID:          uuid.MustParse("66666666-6666-6666-6666-666666666666"),
				ProjectName: "Portal",
				TaskTitle:   "Draw",
				Minutes:     90,
				AmountCents: 4500,
				Position:    1,
			},
		},
	}
}

func withStaffSession(req *http.Request, role string) *http.Request {
	ctx := requestcontext.WithOrganizationID(req.Context(), testOrganizationID())
	ctx = requestcontext.WithUserID(ctx, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	ctx = requestcontext.WithRole(ctx, role)
	return req.WithContext(ctx)
}

type mockService struct {
	listFunc     func(uuid.UUID, uuid.UUID) ([]*invoicedomain.Invoice, error)
	getFunc      func(uuid.UUID, uuid.UUID) (*invoicedomain.Invoice, error)
	createFunc   func(uuid.UUID, uuid.UUID, orgdomain.Role, time.Time, time.Time) (*invoicedomain.Invoice, error)
	updateFunc   func(uuid.UUID, uuid.UUID, orgdomain.Role, time.Time, time.Time) (*invoicedomain.Invoice, error)
	deleteFunc   func(uuid.UUID, uuid.UUID, orgdomain.Role) error
	sendFunc     func(uuid.UUID, uuid.UUID, orgdomain.Role) (*invoicedomain.Invoice, error)
	markPaidFunc func(uuid.UUID, uuid.UUID, orgdomain.Role) (*invoicedomain.Invoice, error)
}

func (m *mockService) List(_ context.Context, organizationID, clientID uuid.UUID) ([]*invoicedomain.Invoice, error) {
	return m.listFunc(organizationID, clientID)
}

func (m *mockService) Get(_ context.Context, organizationID, invoiceID uuid.UUID) (*invoicedomain.Invoice, error) {
	return m.getFunc(organizationID, invoiceID)
}

func (m *mockService) Create(
	_ context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*invoicedomain.Invoice, error) {
	return m.createFunc(organizationID, clientID, actorRole, from, to)
}

func (m *mockService) Update(
	_ context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*invoicedomain.Invoice, error) {
	return m.updateFunc(organizationID, invoiceID, actorRole, from, to)
}

func (m *mockService) Delete(
	_ context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	return m.deleteFunc(organizationID, invoiceID, actorRole)
}

func (m *mockService) Send(
	_ context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) (*invoicedomain.Invoice, error) {
	return m.sendFunc(organizationID, invoiceID, actorRole)
}

func (m *mockService) MarkPaid(
	_ context.Context,
	organizationID, invoiceID uuid.UUID,
	actorRole orgdomain.Role,
) (*invoicedomain.Invoice, error) {
	return m.markPaidFunc(organizationID, invoiceID, actorRole)
}

func TestHandler_List(t *testing.T) {
	invoice := testInvoice()
	service := &mockService{
		listFunc: func(organizationID, clientID uuid.UUID) ([]*invoicedomain.Invoice, error) {
			require.Equal(t, testOrganizationID(), organizationID)
			require.Equal(t, testClientID(), clientID)
			return []*invoicedomain.Invoice{invoice}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+testClientID().String()+"/invoices", nil)
	req.SetPathValue("id", testClientID().String())
	req = withStaffSession(req, "member")
	rec := httptest.NewRecorder()
	invoicehandler.NewHandler(service).List(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var response []invoicedomain.InvoiceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "INV-2026-0001", response[0].Number)
	require.Equal(t, 4500, response[0].TotalCents)
	require.Equal(t, "Portal", response[0].Lines[0].ProjectName)
}

func TestHandler_List_ClientNotFound(t *testing.T) {
	service := &mockService{
		listFunc: func(uuid.UUID, uuid.UUID) ([]*invoicedomain.Invoice, error) {
			return nil, clientdomain.ErrClientNotFound
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+testClientID().String()+"/invoices", nil)
	req.SetPathValue("id", testClientID().String())
	req = withStaffSession(req, "owner")
	rec := httptest.NewRecorder()
	invoicehandler.NewHandler(service).List(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Create(t *testing.T) {
	invoice := testInvoice()
	service := &mockService{
		createFunc: func(
			organizationID, clientID uuid.UUID,
			actorRole orgdomain.Role,
			from, to time.Time,
		) (*invoicedomain.Invoice, error) {
			require.Equal(t, testOrganizationID(), organizationID)
			require.Equal(t, testClientID(), clientID)
			require.Equal(t, orgdomain.RoleOwner, actorRole)
			require.True(t, from.Equal(time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)))
			require.True(t, to.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)))
			return invoice, nil
		},
	}

	body := []byte(`{"from":"2026-09-28T00:00:00.000Z","to":"2026-10-05T00:00:00.000Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+testClientID().String()+"/invoices", bytes.NewReader(body))
	req.SetPathValue("id", testClientID().String())
	req = withStaffSession(req, "owner")
	rec := httptest.NewRecorder()
	invoicehandler.NewHandler(service).Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_Create_NoLineItems(t *testing.T) {
	service := &mockService{
		createFunc: func(uuid.UUID, uuid.UUID, orgdomain.Role, time.Time, time.Time) (*invoicedomain.Invoice, error) {
			return nil, invoicedomain.ErrNoLineItems
		},
	}

	body := []byte(`{"from":"2026-09-28T00:00:00.000Z","to":"2026-10-05T00:00:00.000Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+testClientID().String()+"/invoices", bytes.NewReader(body))
	req.SetPathValue("id", testClientID().String())
	req = withStaffSession(req, "owner")
	rec := httptest.NewRecorder()
	invoicehandler.NewHandler(service).Create(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
