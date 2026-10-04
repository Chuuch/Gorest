package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Organization struct {
	ID                 uuid.UUID
	Name               string
	LegalName          string
	RegistrationNumber string
	VATID              string
	AddressLine1       string
	AddressLine2       string
	City               string
	PostalCode         string
	Country            string
	DefaultVATRateBPS  int
	BankIBAN           string
	BankBIC            string
	BankName           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Membership struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           Role
	CreatedAt      time.Time
}

func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

func (o Organization) InvoiceLegalName() string {
	if strings.TrimSpace(o.LegalName) != "" {
		return strings.TrimSpace(o.LegalName)
	}
	return o.Name
}
