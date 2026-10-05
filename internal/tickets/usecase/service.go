package usecase

import (
	"context"
	"fmt"
	"time"

	clientrepository "github.com/chuuch/gorest/internal/client/repository"
	"github.com/chuuch/gorest/internal/notifications"
	notificationdomain "github.com/chuuch/gorest/internal/notifications/domain"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketrepository "github.com/chuuch/gorest/internal/tickets/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, clientID uuid.UUID,
	) ([]*ticketdomain.Ticket, error)
	Create(
		ctx context.Context,
		organizationID, clientID, userID uuid.UUID,
		req ticketdomain.CreateTicketRequest,
	) (*ticketdomain.Ticket, error)
	Update(
		ctx context.Context,
		organizationID, ticketID uuid.UUID,
		req ticketdomain.UpdateTicketRequest,
	) (*ticketdomain.Ticket, error)
	Delete(
		ctx context.Context,
		organizationID, ticketID uuid.UUID,
		actorRole orgdomain.Role,
	) error
}

type service struct {
	tickets ticketrepository.TicketRepository
	clients clientrepository.ClientRepository
	notify  notifications.Publisher
}

func NewService(
	tickets ticketrepository.TicketRepository,
	clients clientrepository.ClientRepository,
) Service {
	return &service{
		tickets: tickets,
		clients: clients,
		notify:  notifications.Nop{},
	}
}

func EnableNotifications(s Service, pub notifications.Publisher) Service {
	impl, ok := s.(*service)
	if !ok || pub == nil {
		return s
	}
	impl.notify = pub
	return impl
}

func (s *service) List(
	ctx context.Context,
	organizationID, clientID uuid.UUID,
) ([]*ticketdomain.Ticket, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	tickets, err := s.tickets.ListByClientID(ctx, organizationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return tickets, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, clientID, userID uuid.UUID,
	req ticketdomain.CreateTicketRequest,
) (*ticketdomain.Ticket, error) {
	if _, err := s.clients.GetByID(ctx, clientID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	ticket := &ticketdomain.Ticket{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ClientID:       clientID,
		UserID:         userID,
		Kind:           ticketdomain.Kind(req.Kind),
		Status:         ticketdomain.StatusOpen,
		Title:          req.Title,
		Body:           req.Body,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.tickets.Create(ctx, ticket); err != nil {
		return nil, err
	}

	notifications.NotifyStaff(
		ctx,
		s.notify,
		organizationID,
		notificationdomain.KindTicketOpened,
		notificationdomain.EntityTicket,
		ticket.Title,
		ticket.ID,
	)

	return ticket, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, ticketID uuid.UUID,
	req ticketdomain.UpdateTicketRequest,
) (*ticketdomain.Ticket, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID, organizationID)
	if err != nil {
		return nil, err
	}

	previous := ticket.Status
	ticket.Status = ticketdomain.Status(req.Status)
	ticket.UpdatedAt = time.Now().UTC()
	ticket.Version = req.Version

	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}

	if previous != ticket.Status {
		notifications.NotifyUser(
			ctx,
			s.notify,
			organizationID,
			ticket.UserID,
			notificationdomain.KindTicketStatusChanged,
			notificationdomain.EntityTicket,
			ticket.Title,
			ticket.ID,
		)
	}

	return ticket, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, ticketID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return ticketdomain.ErrForbidden
	}

	if _, err := s.tickets.GetByID(ctx, ticketID, organizationID); err != nil {
		return err
	}

	return s.tickets.Delete(ctx, ticketID, organizationID)
}
