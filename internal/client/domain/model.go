package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Notes          string
	LegalName      string
	VATID          string
	AddressLine1   string
	AddressLine2   string
	City           string
	PostalCode     string
	Country        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (c Client) InvoiceLegalName() string {
	if strings.TrimSpace(c.LegalName) != "" {
		return strings.TrimSpace(c.LegalName)
	}
	return c.Name
}
