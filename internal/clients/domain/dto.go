package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateClientRequest struct {
	Name         string `json:"name" validate:"required,min=4,max=100"`
	Notes        string `json:"notes" validate:"max=2000"`
	LegalName    string `json:"legal_name" validate:"max=200"`
	VATID        string `json:"vat_id" validate:"max=32"`
	AddressLine1 string `json:"address_line1" validate:"max=200"`
	AddressLine2 string `json:"address_line2" validate:"max=200"`
	City         string `json:"city" validate:"max=100"`
	PostalCode   string `json:"postal_code" validate:"max=32"`
	Country      string `json:"country" validate:"omitempty,len=2"`
}

type UpdateClientRequest struct {
	Name         string `json:"name" validate:"required,min=4,max=100"`
	Notes        string `json:"notes" validate:"max=2000"`
	LegalName    string `json:"legal_name" validate:"max=200"`
	VATID        string `json:"vat_id" validate:"max=32"`
	AddressLine1 string `json:"address_line1" validate:"max=200"`
	AddressLine2 string `json:"address_line2" validate:"max=200"`
	City         string `json:"city" validate:"max=100"`
	PostalCode   string `json:"postal_code" validate:"max=32"`
	Country      string `json:"country" validate:"omitempty,len=2"`
}

type ClientResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Notes          string    `json:"notes"`
	LegalName      string    `json:"legal_name"`
	VATID          string    `json:"vat_id"`
	AddressLine1   string    `json:"address_line1"`
	AddressLine2   string    `json:"address_line2"`
	City           string    `json:"city"`
	PostalCode     string    `json:"postal_code"`
	Country        string    `json:"country"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func ToResponse(client *Client) ClientResponse {
	return ClientResponse{
		ID:             client.ID,
		OrganizationID: client.OrganizationID,
		Name:           client.Name,
		Notes:          client.Notes,
		LegalName:      client.LegalName,
		VATID:          client.VATID,
		AddressLine1:   client.AddressLine1,
		AddressLine2:   client.AddressLine2,
		City:           client.City,
		PostalCode:     client.PostalCode,
		Country:        client.Country,
		CreatedAt:      client.CreatedAt,
		UpdatedAt:      client.UpdatedAt,
	}
}
