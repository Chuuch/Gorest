package usecase

import (
	"context"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	phasedomain "github.com/chuuch/gorest/internal/projects/phases/domain"
	phaserepository "github.com/chuuch/gorest/internal/projects/phases/repository"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, organizationID, projectID uuid.UUID) ([]*phasedomain.Phase, error)
	ListPortal(
		ctx context.Context,
		organizationID, clientID, projectID uuid.UUID,
	) ([]*phasedomain.Phase, error)
	Create(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
		actorRole orgdomain.Role,
		req phasedomain.CreatePhaseRequest,
	) (*phasedomain.Phase, error)
	Update(
		ctx context.Context,
		organizationID, phaseID uuid.UUID,
		actorRole orgdomain.Role,
		req phasedomain.UpdatePhaseRequest,
	) (*phasedomain.Phase, error)
	EnsureDefaults(ctx context.Context, organizationID, projectID uuid.UUID) error
}

var defaultPhaseNames = []string{
	"Kickoff",
	"Discovery",
	"Design",
	"Development",
	"Content",
	"QA",
	"Launch",
	"Support",
}

type service struct {
	phases   phaserepository.PhaseRepository
	projects projectrepository.ProjectRepository
}

func NewService(
	phases phaserepository.PhaseRepository,
	projects projectrepository.ProjectRepository,
) Service {
	return &service{
		phases:   phases,
		projects: projects,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*phasedomain.Phase, error) {
	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}
	return s.phases.ListByProjectID(ctx, organizationID, projectID)
}

func (s *service) ListPortal(
	ctx context.Context,
	organizationID, clientID, projectID uuid.UUID,
) ([]*phasedomain.Phase, error) {
	project, err := s.projects.GetByID(ctx, projectID, organizationID)
	if err != nil {
		return nil, err
	}
	if project.ClientID != clientID {
		return nil, projectdomain.ErrProjectNotFound
	}
	return s.phases.ListByProjectID(ctx, organizationID, projectID)
}

func (s *service) EnsureDefaults(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) error {
	existing, err := s.phases.ListByProjectID(ctx, organizationID, projectID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	now := time.Now().UTC()
	for i, name := range defaultPhaseNames {
		phase := &phasedomain.Phase{
			ID:             uuid.New(),
			OrganizationID: organizationID,
			ProjectID:      projectID,
			Name:           name,
			SortOrder:      i,
			Status:         phasedomain.StatusNotStarted,
			Version:        1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.phases.Create(ctx, phase); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
	actorRole orgdomain.Role,
	req phasedomain.CreatePhaseRequest,
) (*phasedomain.Phase, error) {
	if !actorRole.CanManageMembers() {
		return nil, phasedomain.ErrForbidden
	}

	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	startsOn, dueOn, err := parseOptionalDates(req.StartsOn, req.DueOn)
	if err != nil {
		return nil, err
	}

	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	} else {
		sortOrder, err = s.phases.NextSortOrder(ctx, organizationID, projectID)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	phase := &phasedomain.Phase{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Name:           req.Name,
		SortOrder:      sortOrder,
		Status:         phasedomain.StatusNotStarted,
		StartsOn:       startsOn,
		DueOn:          dueOn,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.phases.Create(ctx, phase); err != nil {
		return nil, err
	}
	return phase, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, phaseID uuid.UUID,
	actorRole orgdomain.Role,
	req phasedomain.UpdatePhaseRequest,
) (*phasedomain.Phase, error) {
	if !actorRole.CanManageMembers() {
		return nil, phasedomain.ErrForbidden
	}

	phase, err := s.phases.GetByID(ctx, phaseID, organizationID)
	if err != nil {
		return nil, err
	}

	startsOn, dueOn, err := parseOptionalDates(req.StartsOn, req.DueOn)
	if err != nil {
		return nil, err
	}

	phase.Name = req.Name
	phase.SortOrder = req.SortOrder
	phase.Status = phasedomain.Status(req.Status)
	phase.StartsOn = startsOn
	phase.DueOn = dueOn
	phase.UpdatedAt = time.Now().UTC()
	phase.Version = req.Version

	if err := s.phases.Update(ctx, phase); err != nil {
		return nil, err
	}

	return phase, nil
}

func parseOptionalDates(startsOn, dueOn *string) (*time.Time, *time.Time, error) {
	var start, due *time.Time

	if startsOn != nil {
		day, err := time.Parse("2006-01-02", *startsOn)
		if err != nil {
			return nil, nil, phasedomain.ErrInvalidDates
		}
		start = &day
	}
	if dueOn != nil {
		day, err := time.Parse("2006-01-02", *dueOn)
		if err != nil {
			return nil, nil, phasedomain.ErrInvalidDates
		}
		due = &day
	}
	if start != nil && due != nil && start.After(*due) {
		return nil, nil, phasedomain.ErrInvalidDates
	}
	return start, due, nil
}
