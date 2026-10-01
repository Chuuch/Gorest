package repository

import (
	"context"
	"time"

	"github.com/chuuch/gorest/internal/reports/domain"
	"github.com/google/uuid"
)

type ReportRepository interface {
	TimeReport(
		ctx context.Context,
		organizationID uuid.UUID,
		userID *uuid.UUID,
		from, to time.Time,
	) (*domain.TimeReport, error)
}
