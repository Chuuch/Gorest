package domain

import (
	"encoding/json"
	"time"

	"github.com/chuuch/gorest/internal/estimates/engine"
	"github.com/google/uuid"
)

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
