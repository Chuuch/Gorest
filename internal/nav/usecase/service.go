package usecase

import (
	"context"

	"github.com/chuuch/gorest/internal/nav/domain"
	"github.com/chuuch/gorest/internal/nav/postgres"
	"github.com/google/uuid"
)

type Service interface {
	Counts(
		ctx context.Context,
		organizationID, userID uuid.UUID,
		portal bool,
	) (domain.Counts, error)
}

type service struct {
	repo *postgres.Repository
}

func NewService(repo *postgres.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Counts(
	ctx context.Context,
	organizationID, userID uuid.UUID,
	portal bool,
) (domain.Counts, error) {
	unread, err := s.repo.CountUnreadNotifications(ctx, organizationID, userID)
	if err != nil {
		return domain.Counts{}, err
	}

	if portal {
		return domain.Counts{UnreadNotifications: unread}, nil
	}

	tasks, err := s.repo.CountInboxActive(ctx, organizationID, userID)
	if err != nil {
		return domain.Counts{}, err
	}

	tickets, err := s.repo.CountActiveTickets(ctx, organizationID)
	if err != nil {
		return domain.Counts{}, err
	}

	return domain.Counts{
		Tasks:               tasks,
		Tickets:             tickets,
		UnreadNotifications: unread,
	}, nil
}
