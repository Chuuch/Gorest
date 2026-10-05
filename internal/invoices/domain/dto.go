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
	ID                       uuid.UUID          `json:"id"`
	OrganizationID           uuid.UUID          `json:"organization_id"`
	ClientID                 uuid.UUID          `json:"client_id"`
	Number                   string             `json:"number"`
	Status                   string             `json:"status"`
	Currency                 string             `json:"currency"`
	RateCents                int                `json:"rate_cents"`
	OrganizationName         string             `json:"organization_name"`
	ClientName               string             `json:"client_name"`
	SellerLegalName          string             `json:"seller_legal_name"`
	SellerRegistrationNumber string             `json:"seller_registration_number"`
	SellerVATID              string             `json:"seller_vat_id"`
	SellerAddressLine1       string             `json:"seller_address_line1"`
	SellerAddressLine2       string             `json:"seller_address_line2"`
	SellerCity               string             `json:"seller_city"`
	SellerPostalCode         string             `json:"seller_postal_code"`
	SellerCountry            string             `json:"seller_country"`
	BuyerLegalName           string             `json:"buyer_legal_name"`
	BuyerVATID               string             `json:"buyer_vat_id"`
	BuyerAddressLine1        string             `json:"buyer_address_line1"`
	BuyerAddressLine2        string             `json:"buyer_address_line2"`
	BuyerCity                string             `json:"buyer_city"`
	BuyerPostalCode          string             `json:"buyer_postal_code"`
	BuyerCountry             string             `json:"buyer_country"`
	VATRegime                string             `json:"vat_regime"`
	VATRateBPS               int                `json:"vat_rate_bps"`
	SubtotalCents            int                `json:"subtotal_cents"`
	VATCents                 int                `json:"vat_cents"`
	BankIBAN                 string             `json:"bank_iban"`
	BankBIC                  string             `json:"bank_bic"`
	BankName                 string             `json:"bank_name"`
	PeriodFrom               time.Time          `json:"period_from"`
	PeriodTo                 time.Time          `json:"period_to"`
	IssuedAt                 time.Time          `json:"issued_at"`
	DueAt                    time.Time          `json:"due_at"`
	SentAt                   *time.Time         `json:"sent_at"`
	PaidAt                   *time.Time         `json:"paid_at"`
	TotalMinutes             int                `json:"total_minutes"`
	TotalCents               int                `json:"total_cents"`
	CreatedAt                time.Time          `json:"created_at"`
	UpdatedAt                time.Time          `json:"updated_at"`
	Lines                    []LineItemResponse `json:"lines"`
}

type CreateInvoiceRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type UpdateInvoiceRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}
