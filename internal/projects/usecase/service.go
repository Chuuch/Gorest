package usecase

import (
	"context"
	"fmt"
	"time"

	clientrepository "github.com/chuuch/gorest/internal/client/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID,
		clientID uuid.UUID,
	) ([]*projectdomain.Project, error)
	Create(
		ctx context.Context,
		organizationID,
		clientID uuid.UUID,
		actorRole orgdomain.Role,
		req projectdomain.CreateProjectRequest,
	) (*projectdomain.Project, error)
	Update(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
		actorRole orgdomain.Role,
		req projectdomain.UpdateProjectRequest,
	) (*projectdomain.Project, error)
	Delete(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
		actorRole orgdomain.Role,
	) error
}

type service struct {
	projects projectrepository.ProjectRepository
	clients  clientrepository.ClientRepository
}

func NewService(
	projects projectrepository.ProjectRepository,
	clients clientrepository.ClientRepository,
) Service {
	return &service{
		projects: projects,
		clients:  clients,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*projectdomain.Project, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	projects, err := s.projects.ListByClientID(ctx, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	req projectdomain.CreateProjectRequest,
) (*projectdomain.Project, error) {
	if !actorRole.CanManageMembers() {
		return nil, projectdomain.ErrForbidden
	}

	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	project := &projectdomain.Project{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ClientID:       clientID,
		Name:           req.Name,
		Notes:          req.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.projects.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
	actorRole orgdomain.Role,
	req projectdomain.UpdateProjectRequest,
) (*projectdomain.Project, error) {
	if !actorRole.CanManageMembers() {
		return nil, projectdomain.ErrForbidden
	}

	project, err := s.projects.GetByID(ctx, projectID, organizationID)
	if err != nil {
		return nil, err
	}

	project.Name = req.Name
	project.Notes = req.Notes
	project.UpdatedAt = time.Now().UTC()

	if err := s.projects.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return projectdomain.ErrForbidden
	}

	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return err
	}

	return s.projects.Delete(ctx, projectID, organizationID)
}
