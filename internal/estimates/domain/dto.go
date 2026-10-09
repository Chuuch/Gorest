package domain

import (
	"time"

	"github.com/chuuch/gorest/internal/estimates/engine"
	"github.com/google/uuid"
)

type PreviewRequest struct {
	Input engine.Input `json:"input"`
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
