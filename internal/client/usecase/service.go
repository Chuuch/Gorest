package usecase

import (
	"context"
	"fmt"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	clientrepository "github.com/chuuch/gorest/internal/client/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, organizationID uuid.UUID) ([]*clientdomain.Client, error)
	Create(
		ctx context.Context,
		organizationID uuid.UUID,
		actorRole orgdomain.Role,
		req clientdomain.CreateClientRequest,
	) (*clientdomain.Client, error)
}

type service struct {
	clients clientrepository.ClientRepository
}

func NewService(clients clientrepository.ClientRepository) Service {
	return &service{
		clients: clients,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]*clientdomain.Client, error) {
	clients, err := s.clients.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}

	return clients, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	actorRole orgdomain.Role,
	req clientdomain.CreateClientRequest,
) (*clientdomain.Client, error) {
	if !actorRole.CanManageMembers() {
		return nil, clientdomain.ErrForbidden
	}

	now := time.Now().UTC()

	client := &clientdomain.Client{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           req.Name,
		Notes:          req.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.clients.Create(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}
