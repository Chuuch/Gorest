package usecase

import (
	"context"
	"fmt"
	"time"

	clientrepository "github.com/chuuch/gorest/internal/clients/repository"
	"github.com/chuuch/gorest/internal/estimates/domain"
	"github.com/chuuch/gorest/internal/estimates/engine"
	estimaterepository "github.com/chuuch/gorest/internal/estimates/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	"github.com/google/uuid"
)

type Service interface {
	Catalog() domain.CatalogResponse
	Preview(input engine.Input) (domain.PreviewResponse, error)
	Create(
		ctx context.Context,
		organizationID, userID uuid.UUID,
		actorRole orgdomain.Role,
		req domain.CreateEstimateRequest,
	) (*domain.EstimateRun, error)
	Get(ctx context.Context, organizationID, id uuid.UUID) (*domain.EstimateRun, error)
	List(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		clientID *uuid.UUID,
	) ([]*domain.EstimateRun, *string, error)
	CreateProject(
		ctx context.Context,
		organizationID, estimateID, userID uuid.UUID,
		actorRole orgdomain.Role,
		req domain.CreateProjectFromEstimateRequest,
	) (*projectdomain.Project, error)
}

type service struct {
	estimates estimaterepository.EstimateRepository
	projects  projectrepository.ProjectRepository
	clients   clientrepository.ClientRepository
	catalog   engine.Catalog
}

func NewService(
	estimates estimaterepository.EstimateRepository,
	projects projectrepository.ProjectRepository,
	clients clientrepository.ClientRepository,
) Service {
	return &service{
		estimates: estimates,
		projects:  projects,
		clients:   clients,
		catalog:   engine.DefaultCatalog(),
	}
}

func (s *service) Catalog() domain.CatalogResponse {
	c := s.catalog
	return domain.CatalogResponse{
		ID:                  c.ID,
		Version:             c.Version,
		Currency:            c.Currency,
		FloorCentsPerHour:   c.FloorCentsPerHour,
		TargetCentsPerHour:  c.TargetCentsPerHour,
		TargetMultiplierBPS: c.TargetMultiplierBPS,
	}
}

func (s *service) Preview(input engine.Input) (domain.PreviewResponse, error) {
	result, err := engine.Estimate(input, s.catalog)
	if err != nil {
		return domain.PreviewResponse{}, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	return domain.PreviewResponse{
		Catalog: s.Catalog(),
		Result:  result,
	}, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, userID uuid.UUID,
	actorRole orgdomain.Role,
	req domain.CreateEstimateRequest,
) (*domain.EstimateRun, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}
	if req.ClientID != nil {
		if _, err := s.clients.GetByID(ctx, *req.ClientID, organizationID); err != nil {
			return nil, err
		}
	}

	result, err := engine.Estimate(req.Input, s.catalog)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}

	now := time.Now().UTC()
	run := &domain.EstimateRun{
		ID:                    uuid.New(),
		OrganizationID:        organizationID,
		CreatedByUserID:       userID,
		ClientID:              req.ClientID,
		Category:              req.Input.Category,
		Mode:                  req.Input.Mode,
		CatalogVersionID:      s.catalog.ID,
		Currency:              s.catalog.Currency,
		Input:                 req.Input,
		Result:                result,
		EstimatedHours:        result.EstimatedHours,
		EstimatedTimelineDays: result.EstimatedTimelineDays,
		HoursPerMonth:         result.HoursPerMonth,
		MinimumPriceCents:     result.MinimumPriceCents,
		RecommendedPriceCents: result.RecommendedPriceCents,
		RiskLevel:             result.RiskLevel,
		CreatedAt:             now,
	}

	if err := s.estimates.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *service) Get(ctx context.Context, organizationID, id uuid.UUID) (*domain.EstimateRun, error) {
	return s.estimates.GetByID(ctx, id, organizationID)
}

func (s *service) List(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	clientID *uuid.UUID,
) ([]*domain.EstimateRun, *string, error) {
	runs, err := s.estimates.List(ctx, organizationID, limit+1, cursor, clientID)
	if err != nil {
		return nil, nil, err
	}
	page, next := pagination.NextCursor(runs, limit, func(run *domain.EstimateRun) pagination.Cursor {
		return pagination.Cursor{CreatedAt: run.CreatedAt, ID: run.ID}
	})
	return page, next, nil
}

func (s *service) CreateProject(
	ctx context.Context,
	organizationID, estimateID, userID uuid.UUID,
	actorRole orgdomain.Role,
	req domain.CreateProjectFromEstimateRequest,
) (*projectdomain.Project, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	run, err := s.estimates.GetByID(ctx, estimateID, organizationID)
	if err != nil {
		return nil, err
	}
	if _, err := s.clients.GetByID(ctx, req.ClientID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	notes := req.Notes
	if notes == "" {
		notes = fmt.Sprintf(
			"From estimate %s (%s/%s). Recommended: %d cents %s. Risk: %s.",
			run.ID, run.Category, run.Mode, run.RecommendedPriceCents, run.Currency, run.RiskLevel,
		)
	}

	project := &projectdomain.Project{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ClientID:       req.ClientID,
		Name:           req.Name,
		Notes:          notes,
		EstimateRunID:  &run.ID,
		EstimatedHours: run.EstimatedHours,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.projects.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}
