package domain

import (
	"encoding/json"
	"time"

	"github.com/chuuch/gorest/internal/estimates/engine"
	"github.com/google/uuid"
)

const (
	IntakeStatusPending   = "pending"
	IntakeStatusSubmitted = "submitted"
	IntakeStatusConsumed  = "consumed"
	IntakeStatusExpired   = "expired"
)

type IntakeScope struct {
	ProjectName       string                    `json:"project_name,omitempty"`
	RespondentName    string                    `json:"respondent_name,omitempty"`
	CompanyName       string                    `json:"company_name,omitempty"`
	CustomerNotes     string                    `json:"customer_notes,omitempty"`
	PreferredDeadline *string                   `json:"preferred_deadline,omitempty"`
	Website           *engine.WebsiteInput      `json:"website,omitempty"`
	Design            *engine.DesignInput       `json:"design,omitempty"`
	Branding          *engine.BrandingInput     `json:"branding,omitempty"`
	CustomWeb         *engine.CustomWebInput    `json:"custom_web,omitempty"`
	CustomMobile      *engine.CustomMobileInput `json:"custom_mobile,omitempty"`
	Marketing         *engine.MarketingInput    `json:"marketing,omitempty"`
}

type EstimateIntake struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	CreatedByUserID   uuid.UUID
	TokenHash         string
	Category          string
	Mode              string
	Status            string
	ExpiresAt         time.Time
	RespondentEmail   *string
	RespondentName    *string
	CompanyName       *string
	ProjectName       *string
	PreferredDeadline *time.Time
	ClientID          *uuid.UUID
	Scope             *IntakeScope
	SubmittedAt       *time.Time
	CreatedAt         time.Time
}

type EstimateRun struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	CreatedByUserID       uuid.UUID
	IntakeID              *uuid.UUID
	ClientID              *uuid.UUID
	Category              string
	Mode                  string
	CatalogVersionID      uuid.UUID
	Currency              string
	Input                 engine.Input
	Result                engine.Result
	EstimatedHours        *float64
	EstimatedTimelineDays *int
	HoursPerMonth         *float64
	MinimumPriceCents     int
	RecommendedPriceCents int
	RiskLevel             string
	CreatedAt             time.Time
}

type EstimatedRunRow struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	CreatedByUserID       uuid.UUID
	IntakeID              *uuid.UUID
	ClientID              *uuid.UUID
	Category              string
	Mode                  string
	CatalogVersionID      uuid.UUID
	Currency              string
	InputJSON             json.RawMessage
	ResultJSON            json.RawMessage
	EstimatedHours        *float64
	EstimatedTimelineDays *int
	HoursPerMonth         *float64
	MinimumPriceCents     int
	RecommendedPriceCents int
	RiskLevel             string
	CreatedAt             time.Time
}
