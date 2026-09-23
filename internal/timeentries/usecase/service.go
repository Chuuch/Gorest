package usecase

import (
	"context"
	"fmt"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	taskrepository "github.com/chuuch/gorest/internal/tasks/repository"
	timeentrydomain "github.com/chuuch/gorest/internal/timeentries/domain"
	timeentryrepository "github.com/chuuch/gorest/internal/timeentries/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, organizationID, taskID uuid.UUID) ([]*timeentrydomain.TimeEntry, error)
	Create(
		ctx context.Context,
		organizationID, taskID, userID uuid.UUID,
		req timeentrydomain.CreateTimeEntryRequest,
	) (*timeentrydomain.TimeEntry, error)
	Update(
		ctx context.Context,
		organizationID, entryID, actorUserID uuid.UUID,
		actorRole orgdomain.Role,
		req timeentrydomain.UpdateTimeEntryRequest,
	) (*timeentrydomain.TimeEntry, error)
	Delete(
		ctx context.Context,
		organizationID, entryID, actorUserID uuid.UUID,
		actorRole orgdomain.Role,
	) error
}

type service struct {
	entries timeentryrepository.TimeEntryRepository
	tasks   taskrepository.TaskRepository
}

func NewService(
	entries timeentryrepository.TimeEntryRepository,
	tasks taskrepository.TaskRepository,
) Service {
	return &service{
		entries: entries,
		tasks:   tasks,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, taskID uuid.UUID,
) ([]*timeentrydomain.TimeEntry, error) {
	if _, err := s.tasks.GetByID(ctx, taskID, organizationID); err != nil {
		return nil, err
	}

	entries, err := s.entries.ListByTaskID(ctx, organizationID, taskID)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}
	return entries, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, taskID, userID uuid.UUID,
	req timeentrydomain.CreateTimeEntryRequest,
) (*timeentrydomain.TimeEntry, error) {
	if _, err := s.tasks.GetByID(ctx, taskID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	entry := &timeentrydomain.TimeEntry{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		TaskID:         taskID,
		UserID:         userID,
		Minutes:        req.Minutes,
		Notes:          req.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.entries.Create(ctx, entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, entryID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
	req timeentrydomain.UpdateTimeEntryRequest,
) (*timeentrydomain.TimeEntry, error) {
	entry, err := s.entries.GetByID(ctx, entryID, organizationID)
	if err != nil {
		return nil, err
	}

	if !canMutateTimeEntry(actorRole, actorUserID, entry.UserID) {
		return nil, timeentrydomain.ErrForbidden
	}

	entry.Minutes = req.Minutes
	entry.Notes = req.Notes
	entry.UpdatedAt = time.Now().UTC()

	if err := s.entries.Update(ctx, entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, entryID, actorUserID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	entry, err := s.entries.GetByID(ctx, entryID, organizationID)
	if err != nil {
		return err
	}

	if !canMutateTimeEntry(actorRole, actorUserID, entry.UserID) {
		return timeentrydomain.ErrForbidden
	}

	return s.entries.Delete(ctx, entryID, organizationID)
}

func canMutateTimeEntry(actorRole orgdomain.Role, actorUserID, ownerUserID uuid.UUID) bool {
	if actorRole.CanManageMembers() {
		return true
	}

	return actorUserID == ownerUserID
}
