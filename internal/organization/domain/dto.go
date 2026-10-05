package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	LegalName          string    `json:"legal_name"`
	RegistrationNumber string    `json:"registration_number"`
	VATID              string    `json:"vat_id"`
	AddressLine1       string    `json:"address_line1"`
	AddressLine2       string    `json:"address_line2"`
	City               string    `json:"city"`
	PostalCode         string    `json:"postal_code"`
	Country            string    `json:"country"`
	DefaultVATRateBPS  int       `json:"default_vat_rate_bps"`
	BankIBAN           string    `json:"bank_iban"`
	BankBIC            string    `json:"bank_bic"`
	BankName           string    `json:"bank_name"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateOrganizationRequest struct {
	Name               string `json:"name" validate:"required,min=4,max=100"`
	LegalName          string `json:"legal_name" validate:"max=200"`
	RegistrationNumber string `json:"registration_number" validate:"max=64"`
	VATID              string `json:"vat_id" validate:"max=32"`
	AddressLine1       string `json:"address_line1" validate:"max=200"`
	AddressLine2       string `json:"address_line2" validate:"max=200"`
	City               string `json:"city" validate:"max=100"`
	PostalCode         string `json:"postal_code" validate:"max=32"`
	Country            string `json:"country" validate:"omitempty,len=2"`
	DefaultVATRateBPS  int    `json:"default_vat_rate_bps" validate:"min=0,max=10000"`
	BankIBAN           string `json:"bank_iban" validate:"max=64"`
	BankBIC            string `json:"bank_bic" validate:"max=32"`
	BankName           string `json:"bank_name" validate:"max=100"`
}

type MemberResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member"`
}

type UpdateMemberRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

func ToResponse(org *Organization) OrganizationResponse {
	rate := org.DefaultVATRateBPS
	if rate == 0 {
		rate = 2000
	}
	return OrganizationResponse{
		ID:                 org.ID,
		Name:               org.Name,
		LegalName:          org.LegalName,
		RegistrationNumber: org.RegistrationNumber,
		VATID:              org.VATID,
		AddressLine1:       org.AddressLine1,
		AddressLine2:       org.AddressLine2,
		City:               org.City,
		PostalCode:         org.PostalCode,
		Country:            org.Country,
		DefaultVATRateBPS:  rate,
		BankIBAN:           org.BankIBAN,
		BankBIC:            org.BankBIC,
		BankName:           org.BankName,
		CreatedAt:          org.CreatedAt,
		UpdatedAt:          org.UpdatedAt,
	}
}
