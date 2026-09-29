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
	Update(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		actorRole orgdomain.Role,
		req clientdomain.UpdateClientRequest,
	) (*clientdomain.Client, error)
	Delete(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
		actorRole orgdomain.Role,
	) error
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

func (s *service) Update(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
	req clientdomain.UpdateClientRequest,
) (*clientdomain.Client, error) {
	if !actorRole.CanManageMembers() {
		return nil, clientdomain.ErrForbidden
	}

	client, err := s.clients.GetByID(ctx, clientID, organizationID)
	if err != nil {
		return nil, err
	}

	client.Name = req.Name
	client.Notes = req.Notes
	client.UpdatedAt = time.Now().UTC()

	if err := s.clients.Update(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return clientdomain.ErrForbidden
	}

	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return err
	}

	return s.clients.Delete(ctx, clientID, organizationID)
}
