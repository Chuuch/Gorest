package domain

import (
	"time"

	"github.com/google/uuid"
)

type LineItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectName string    `json:"project_name"`
	TaskTitle   string    `json:"task_title"`
	Minutes     int       `json:"minutes"`
	AmountCents int       `json:"amount_cents"`
	Position    int       `json:"position"`
}

type InvoiceResponse struct {
	ID               uuid.UUID          `json:"id"`
	OrganizationID   uuid.UUID          `json:"organization_id"`
	ClientID         uuid.UUID          `json:"client_id"`
	Number           string             `json:"number"`
	Status           string             `json:"status"`
	Currency         string             `json:"currency"`
	RateCents        int                `json:"rate_cents"`
	OrganizationName string             `json:"organization_name"`
	ClientName       string             `json:"client_name"`
	PeriodFrom       time.Time          `json:"period_from"`
	PeriodTo         time.Time          `json:"period_to"`
	IssuedAt         time.Time          `json:"issued_at"`
	DueAt            time.Time          `json:"due_at"`
	SentAt           *time.Time         `json:"sent_at"`
	PaidAt           *time.Time         `json:"paid_at"`
	TotalMinutes     int                `json:"total_minutes"`
	TotalCents       int                `json:"total_cents"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Lines            []LineItemResponse `json:"lines"`
}

type CreateInvoiceRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type UpdateInvoiceRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}
