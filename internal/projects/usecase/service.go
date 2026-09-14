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
