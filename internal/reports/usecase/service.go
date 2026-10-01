package usecase

import (
	"context"
	"fmt"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	reportsdomain "github.com/chuuch/gorest/internal/reports/domain"
	reportsrepository "github.com/chuuch/gorest/internal/reports/repository"
	"github.com/google/uuid"
)

type Service interface {
	Time(
		ctx context.Context,
		organizationID, actorUserID uuid.UUID,
		actorRole orgdomain.Role,
		from, to time.Time,
	) (*reportsdomain.TimeReport, error)
}

type service struct {
	reports reportsrepository.ReportRepository
}

func NewService(reports reportsrepository.ReportRepository) Service {
	return &service{reports: reports}
}

func (s *service) Time(
	ctx context.Context,
	organizationID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
	from, to time.Time,
) (*reportsdomain.TimeReport, error) {
	var userID *uuid.UUID
	if !actorRole.CanManageMembers() {
		userID = &actorUserID
	}

	report, err := s.reports.TimeReport(ctx, organizationID, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("time report: %w", err)
	}

	return report, nil
}
