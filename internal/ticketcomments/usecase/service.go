package usecase

import (
	"context"
	"fmt"
	"time"

	ticketcommentdomain "github.com/chuuch/gorest/internal/ticketcomments/domain"
	ticketcommentrepository "github.com/chuuch/gorest/internal/ticketcomments/repository"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketrepository "github.com/chuuch/gorest/internal/tickets/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, ticketID, portalClientID uuid.UUID,
	) ([]*ticketcommentdomain.Comment, error)
	Create(
		ctx context.Context,
		organizationID, ticketID, userID, portalClientID uuid.UUID,
		req ticketcommentdomain.CreateCommentRequest,
	) (*ticketcommentdomain.Comment, error)
}

type service struct {
	comments ticketcommentrepository.TicketCommentRepository
	tickets  ticketrepository.TicketRepository
}

func NewService(
	comments ticketcommentrepository.TicketCommentRepository,
	tickets ticketrepository.TicketRepository,
) Service {
	return &service{
		comments: comments,
		tickets:  tickets,
	}
}

func (s *service) visibleTicket(
	ctx context.Context,
	organizationID, ticketID, portalClientID uuid.UUID,
) (*ticketdomain.Ticket, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID, organizationID)
	if err != nil {
		return nil, err
	}

	if portalClientID != uuid.Nil && ticket.ClientID != portalClientID {
		return nil, ticketdomain.ErrTicketNotFound
	}

	return ticket, nil
}

func (s *service) List(
	ctx context.Context,
	organizationID, ticketID, portalClientID uuid.UUID,
) ([]*ticketcommentdomain.Comment, error) {
	if _, err := s.visibleTicket(ctx, organizationID, ticketID, portalClientID); err != nil {
		return nil, err
	}

	comments, err := s.comments.ListByTicketID(ctx, organizationID, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list ticket comments: %w", err)
	}
	return comments, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
	req ticketcommentdomain.CreateCommentRequest,
) (*ticketcommentdomain.Comment, error) {
	if _, err := s.visibleTicket(ctx, organizationID, ticketID, portalClientID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	comment := &ticketcommentdomain.Comment{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		TicketID:       ticketID,
		UserID:         userID,
		Body:           req.Body,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.comments.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}
