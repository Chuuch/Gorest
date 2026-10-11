package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chuuch/gorest/internal/auth/security"
	clientrepository "github.com/chuuch/gorest/internal/clients/repository"
	"github.com/chuuch/gorest/internal/estimates"
	"github.com/chuuch/gorest/internal/estimates/domain"
	"github.com/chuuch/gorest/internal/estimates/engine"
	estimaterepository "github.com/chuuch/gorest/internal/estimates/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/mailer"
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
	CreateIntake(
		ctx context.Context,
		organizationID, userID uuid.UUID,
		actorRole orgdomain.Role,
		req domain.CreateIntakeRequest,
	) (*domain.CreateIntakeResponse, error)
	GetIntake(ctx context.Context, organizationID, id uuid.UUID) (*domain.EstimateIntake, error)
	ListIntakes(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		status string,
	) ([]*domain.EstimateIntake, *string, error)
	GetPublicIntake(ctx context.Context, rawToken string) (*domain.PublicIntakeResponse, error)
	SubmitIntake(ctx context.Context, rawToken string, req domain.SubmitIntakeRequest) error
}

type service struct {
	estimates estimaterepository.EstimateRepository
	projects  projectrepository.ProjectRepository
	clients   clientrepository.ClientRepository
	tokens    security.TokenManager
	mailer    mailer.Mailer
	catalog   engine.Catalog
	intakeTTL time.Duration
	publicURL string
}

func NewService(
	estimates estimaterepository.EstimateRepository,
	projects projectrepository.ProjectRepository,
	clients clientrepository.ClientRepository,
	tokens security.TokenManager,
	mail mailer.Mailer,
	intakeTTL time.Duration,
	publicURL string,
) Service {
	return &service{
		estimates: estimates,
		projects:  projects,
		clients:   clients,
		tokens:    tokens,
		mailer:    mail,
		catalog:   engine.DefaultCatalog(),
		intakeTTL: intakeTTL,
		publicURL: strings.TrimRight(publicURL, "/"),
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

func (s *service) CreateIntake(
	ctx context.Context,
	organizationID, userID uuid.UUID,
	actorRole orgdomain.Role,
	req domain.CreateIntakeRequest,
) (*domain.CreateIntakeResponse, error) {
	if !actorRole.CanManageMembers() {
		return nil, domain.ErrForbidden
	}
	if req.Mode != engine.ModeProject && req.Mode != engine.ModeRetainer {
		return nil, domain.ErrInvalidInput
	}
	if req.Category != engine.CategoryMarketing && req.Mode != engine.ModeProject {
		return nil, fmt.Errorf("%w: only marketing supports retainer", domain.ErrInvalidInput)
	}

	raw, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate intake token: %w", err)
	}

	now := time.Now().UTC()
	intake := &domain.EstimateIntake{
		ID:              uuid.New(),
		OrganizationID:  organizationID,
		CreatedByUserID: userID,
		TokenHash:       s.tokens.HashRefreshToken(raw),
		Category:        req.Category,
		Mode:            req.Mode,
		Status:          domain.IntakeStatusPending,
		ExpiresAt:       now.Add(s.intakeTTL),
		CreatedAt:       now,
	}
	if err := s.estimates.CreateIntake(ctx, intake); err != nil {
		return nil, err
	}

	return &domain.CreateIntakeResponse{
		Intake: toIntakeResponse(intake),
		Token:  raw,
		URL:    s.publicURL + "/intake/" + raw,
	}, nil
}

func (s *service) GetIntake(ctx context.Context, organizationID, id uuid.UUID) (*domain.EstimateIntake, error) {
	intake, err := s.estimates.GetIntakeByID(ctx, id, organizationID)
	if err != nil {
		return nil, err
	}

	_ = s.expireIfNeeded(ctx, intake)
	return intake, nil
}

func (s *service) ListIntakes(
	ctx context.Context,
	organizationID uuid.UUID,
	limit int,
	cursor *pagination.Cursor,
	status string,
) ([]*domain.EstimateIntake, *string, error) {
	items, err := s.estimates.ListIntakes(ctx, organizationID, limit+1, cursor, status)
	if err != nil {
		return nil, nil, err
	}
	page, next := pagination.NextCursor(items, limit, func(intake *domain.EstimateIntake) pagination.Cursor {
		return pagination.Cursor{CreatedAt: intake.CreatedAt, ID: intake.ID}
	})
	return page, next, nil
}

func (s *service) GetPublicIntake(ctx context.Context, rawToken string) (*domain.PublicIntakeResponse, error) {
	intake, err := s.intakeByRawToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if err := s.ensureIntakeUsable(intake); err != nil {
		return nil, err
	}
	return &domain.PublicIntakeResponse{
		Category:  intake.Category,
		Mode:      intake.Mode,
		Status:    intake.Status,
		ExpiresAt: intake.ExpiresAt,
	}, nil
}

func (s *service) SubmitIntake(ctx context.Context, rawToken string, req domain.SubmitIntakeRequest) error {
	intake, err := s.intakeByRawToken(ctx, rawToken)
	if err != nil {
		return err
	}
	if err := s.ensureIntakeUsable(intake); err != nil {
		return err
	}
	if intake.Status != domain.IntakeStatusPending {
		if intake.Status == domain.IntakeStatusSubmitted || intake.Status == domain.IntakeStatusConsumed {
			return domain.ErrIntakeSubmitted
		}
		return domain.ErrIntakeNotPending
	}

	scope := req.Scope
	scope.ProjectName = strings.TrimSpace(req.ProjectName)
	scope.RespondentName = strings.TrimSpace(req.RespondentName)
	scope.CompanyName = strings.TrimSpace(req.CompanyName)
	scope.CustomerNotes = strings.TrimSpace(req.CustomerNotes)
	if req.PreferredDeadline != "" {
		d := req.PreferredDeadline
		scope.PreferredDeadline = &d
	}
	if err := validateIntakeScope(intake.Category, &scope); err != nil {
		return err
	}

	now := time.Now().UTC()
	email := strings.TrimSpace(strings.ToLower(req.RespondentEmail))
	name := scope.RespondentName
	company := scope.CompanyName
	project := scope.ProjectName

	intake.Status = domain.IntakeStatusSubmitted
	intake.RespondentEmail = &email
	intake.RespondentName = &name
	intake.CompanyName = &company
	intake.ProjectName = &project
	intake.Scope = &scope
	intake.SubmittedAt = &now

	if req.PreferredDeadline != "" {
		day, err := time.Parse("2006-01-02", req.PreferredDeadline)
		if err != nil {
			return fmt.Errorf("%w: preferred_deadline", domain.ErrInvalidIntakeScope)
		}
		intake.PreferredDeadline = &day
	}

	if err := s.estimates.UpdateIntake(ctx, intake); err != nil {
		return err
	}

	subject, text, html, err := estimates.ConfirmationMail(name, project)
	if err != nil {
		return fmt.Errorf("render confirmation mail: %w", err)
	}
	if err := s.mailer.Send(ctx, mailer.Message{
		To: email, Subject: subject, Text: text, HTML: html,
	}); err != nil {
		return fmt.Errorf("send confirmation mail: %w", err)
	}
	return nil
}

func (s *service) intakeByRawToken(ctx context.Context, rawToken string) (*domain.EstimateIntake, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, domain.ErrIntakeNotFound
	}
	intake, err := s.estimates.GetIntakeByTokenHash(ctx, s.tokens.HashRefreshToken(rawToken))
	if err != nil {
		return nil, err
	}

	_ = s.expireIfNeeded(ctx, intake)
	return intake, nil
}

func (s *service) ensureIntakeUsable(intake *domain.EstimateIntake) error {
	if intake.Status == domain.IntakeStatusExpired || time.Now().UTC().After(intake.ExpiresAt) {
		return domain.ErrIntakeExpired
	}
	return nil
}

func (s *service) expireIfNeeded(ctx context.Context, intake *domain.EstimateIntake) error {
	if intake.Status == domain.IntakeStatusPending && time.Now().UTC().After(intake.ExpiresAt) {
		intake.Status = domain.IntakeStatusExpired
		return s.estimates.UpdateIntake(ctx, intake)
	}
	return nil
}

func validateIntakeScope(category string, scope *domain.IntakeScope) error {
	if scope == nil {
		return domain.ErrInvalidIntakeScope
	}
	switch category {
	case engine.CategoryWebsite:
		if scope.Website == nil {
			return fmt.Errorf("%w: website scope required", domain.ErrInvalidIntakeScope)
		}
	case engine.CategoryDesign:
		if scope.Design == nil {
			return fmt.Errorf("%w: design scope required", domain.ErrInvalidIntakeScope)
		}
	case engine.CategoryBranding:
		if scope.Branding == nil {
			return fmt.Errorf("%w: branding scope required", domain.ErrInvalidIntakeScope)
		}
	case engine.CategoryCustomWeb:
		if scope.CustomWeb == nil {
			return fmt.Errorf("%w: custom_web scopre required", domain.ErrInvalidIntakeScope)
		}
	case engine.CategoryCustomMobile:
		if scope.CustomMobile == nil {
			return fmt.Errorf("%w: custom_mobile scope required", domain.ErrInvalidIntakeScope)
		}
	case engine.CategoryMarketing:
		if scope.Marketing == nil {
			return fmt.Errorf("%w: marketing scope required", domain.ErrInvalidIntakeScope)
		}
	default:
		return domain.ErrInvalidIntakeScope
	}
	return nil
}

func toIntakeResponse(intake *domain.EstimateIntake) domain.IntakeResponse {
	out := domain.IntakeResponse{
		ID:              intake.ID,
		OrganizationID:  intake.OrganizationID,
		CreatedByUserID: intake.OrganizationID,
		Category:        intake.Category,
		Mode:            intake.Mode,
		Status:          intake.Status,
		ExpiresAt:       intake.ExpiresAt,
		RespondentEmail: intake.RespondentEmail,
		RespondentName:  intake.RespondentName,
		CompanyName:     intake.CompanyName,
		ProjectName:     intake.ProjectName,
		ClientID:        intake.ClientID,
		Scope:           intake.Scope,
		SubmittedAt:     intake.SubmittedAt,
		CreatedAt:       intake.CreatedAt,
	}
	if intake.PreferredDeadline != nil {
		d := intake.PreferredDeadline.Format("2006-01-02")
		out.PreferredDeadline = &d
	}
	return out
}
