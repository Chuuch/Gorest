package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	clientdomain "github.com/chuuch/gorest/internal/clients/domain"
	clientrepository "github.com/chuuch/gorest/internal/clients/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/chuuch/gorest/internal/platform/search"
	"github.com/chuuch/gorest/internal/platform/taxid"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID uuid.UUID,
		limit int,
		cursor *pagination.Cursor,
		query string,
	) ([]*clientdomain.Client, *string, error)
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
	limit int,
	cursor *pagination.Cursor,
	query string,
) ([]*clientdomain.Client, *string, error) {
	clients, err := s.clients.ListByOrganizationID(
		ctx,
		organizationID,
		limit+1,
		cursor,
		search.Normalize(query),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list clients: %w", err)
	}

	page, next := pagination.NextCursor(clients, limit, func(client *clientdomain.Client) pagination.Cursor {
		return pagination.Cursor{
			CreatedAt: client.CreatedAt,
			ID:        client.ID,
		}
	})
	return page, next, nil
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
		LegalName:      strings.TrimSpace(req.LegalName),
		VATID:          taxid.NormalizeVATID(req.VATID),
		AddressLine1:   strings.TrimSpace(req.AddressLine1),
		AddressLine2:   strings.TrimSpace(req.AddressLine2),
		City:           strings.TrimSpace(req.City),
		PostalCode:     strings.TrimSpace(req.PostalCode),
		Country:        taxid.NormalizeCountry(req.Country),
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
	client.LegalName = strings.TrimSpace(req.LegalName)
	client.VATID = taxid.NormalizeVATID(req.VATID)
	client.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	client.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	client.City = strings.TrimSpace(req.City)
	client.PostalCode = strings.TrimSpace(req.PostalCode)
	client.Country = taxid.NormalizeCountry(req.Country)
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
