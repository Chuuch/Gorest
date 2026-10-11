package domain

import (
	"time"

	"github.com/chuuch/gorest/internal/estimates/engine"
	"github.com/google/uuid"
)

type CreateIntakeRequest struct {
	Category string `json:"category" validate:"required,oneof=website design branding custom_web custom_mobile marketing"`
	Mode     string `json:"mode" validate:"required,oneof=project retainer"`
}

type CreateIntakeResponse struct {
	Intake IntakeResponse `json:"intake"`
	Token  string         `json:"token"`
	URL    string         `json:"url"`
}

type IntakeResponse struct {
	ID                uuid.UUID    `json:"id"`
	OrganizationID    uuid.UUID    `json:"organization_id"`
	CreatedByUserID   uuid.UUID    `json:"created_by_user_id"`
	Category          string       `json:"category"`
	Mode              string       `json:"mode"`
	Status            string       `json:"status"`
	ExpiresAt         time.Time    `json:"expires_at"`
	RespondentEmail   *string      `json:"respondent_email,omitempty"`
	RespondentName    *string      `json:"respondent_name,omitempty"`
	CompanyName       *string      `json:"company_name,omitempty"`
	ProjectName       *string      `json:"project_name,omitempty"`
	PreferredDeadline *string      `json:"preferred_deadline,omitempty"`
	ClientID          *uuid.UUID   `json:"client_id,omitempty"`
	Scope             *IntakeScope `json:"scope,omitempty"`
	SubmittedAt       *time.Time   `json:"submitted_at,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
}

type PublicIntakeResponse struct {
	Category  string    `json:"category"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SubmitIntakeRequest struct {
	RespondentEmail   string      `json:"respondent_email" validate:"required,email"`
	RespondentName    string      `json:"respondent_name" validate:"required,min=1,max=200"`
	CompanyName       string      `json:"company_name" validate:"max=200"`
	ProjectName       string      `json:"project_name" validate:"required,min=1,max=200"`
	CustomerNotes     string      `json:"customer_notes" validate:"max=4000"`
	PreferredDeadline string      `json:"preferred_deadline" validate:"omitempty,datetime=2006-01-02"`
	Scope             IntakeScope `json:"scope"`
}

type PreviewRequest struct {
	Input    engine.Input `json:"input"`
	IntakeID *uuid.UUID   `json:"intake_id"`
}

type CreateEstimateRequest struct {
	ClientID *uuid.UUID   `json:"client_id"`
	Input    engine.Input `json:"input"`
}

type CreateProjectFromEstimateRequest struct {
	ClientID uuid.UUID `json:"client_id" validate:"required"`
	Name     string    `json:"name" validate:"required,min=4,max=100"`
	Notes    string    `json:"notes" validate:"max=2000"`
}

type CatalogResponse struct {
	ID                  uuid.UUID `json:"id"`
	Version             string    `json:"version"`
	Currency            string    `json:"currency"`
	FloorCentsPerHour   int       `json:"floor_cents_per_hour"`
	TargetCentsPerHour  int       `json:"target_cents_per_hour"`
	TargetMultiplierBPS int       `json:"target_multiplier_bps"`
}

type EstimateResponse struct {
	ID                    uuid.UUID     `json:"id"`
	OrganizationID        uuid.UUID     `json:"organization_id"`
	CreatedByUserID       uuid.UUID     `json:"created_by_user_id"`
	ClientID              *uuid.UUID    `json:"client_id,omitempty"`
	Category              string        `json:"category"`
	Mode                  string        `json:"mode"`
	CatalogVersionID      uuid.UUID     `json:"catalog_version_id"`
	Currency              string        `json:"currency"`
	Input                 engine.Input  `json:"input"`
	Result                engine.Result `json:"result"`
	EstimatedHours        *float64      `json:"estimated_hours,omitempty"`
	EstimatedTimelineDays *int          `json:"estimated_timeline_days,omitempty"`
	HoursPerMonth         *float64      `json:"hours_per_month,omitempty"`
	MinimumPriceCents     int           `json:"minimum_price_cents"`
	RecommendedPriceCents int           `json:"recommended_price_cents"`
	RiskLevel             string        `json:"risk_level"`
	CreatedAt             time.Time     `json:"created_at"`
}

type PreviewResponse struct {
	Catalog CatalogResponse `json:"catalog"`
	Result  engine.Result   `json:"result"`
}
