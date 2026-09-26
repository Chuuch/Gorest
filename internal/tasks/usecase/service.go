package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	orgrepository "github.com/chuuch/gorest/internal/organization/repository"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	"github.com/chuuch/gorest/internal/requestcontext"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskrepository "github.com/chuuch/gorest/internal/tasks/repository"
	ticketrepository "github.com/chuuch/gorest/internal/tickets/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
	) ([]*taskdomain.Task, error)
	Inbox(
		ctx context.Context,
		organizationID, userID uuid.UUID,
	) ([]*taskdomain.Task, error)
	Create(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
		actorRole orgdomain.Role,
		req taskdomain.CreateTaskRequest,
	) (*taskdomain.Task, error)
	Update(
		ctx context.Context,
		organizationID, taskID uuid.UUID,
		req taskdomain.UpdateTaskRequest,
	) (*taskdomain.Task, error)
	Delete(
		ctx context.Context,
		organizationID, taskID uuid.UUID,
		actorRole orgdomain.Role,
	) error
	Convert(
		ctx context.Context,
		organizationID, ticketID uuid.UUID,
		actorRole orgdomain.Role,
		req taskdomain.ConvertTicketRequest,
	) (*taskdomain.Task, error)
}

type service struct {
	tasks       taskrepository.TaskRepository
	projects    projectrepository.ProjectRepository
	tickets     ticketrepository.TicketRepository
	memberships orgrepository.MembershipRepository
}

func NewService(
	tasks taskrepository.TaskRepository,
	projects projectrepository.ProjectRepository,
	tickets ticketrepository.TicketRepository,
	memberships orgrepository.MembershipRepository,
) Service {
	return &service{
		tasks:       tasks,
		projects:    projects,
		tickets:     tickets,
		memberships: memberships,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*taskdomain.Task, error) {
	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	tasks, err := s.tasks.ListByProjectID(ctx, organizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func (s *service) Inbox(
	ctx context.Context,
	organizationID, userID uuid.UUID,
) ([]*taskdomain.Task, error) {
	tasks, err := s.tasks.ListInbox(ctx, organizationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	return tasks, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
	_ orgdomain.Role,
	req taskdomain.CreateTaskRequest,
) (*taskdomain.Task, error) {
	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	assigneeID, err := s.resolveAssignee(ctx, organizationID, req.AssigneeID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	status := taskdomain.Status(req.Status)
	createdBy := actorUserID(ctx)

	task := &taskdomain.Task{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Title:          req.Title,
		Notes:          req.Notes,
		Status:         status,
		CompletedAt:    completedAtFor(status, nil, now),
		CreatedBy:      createdBy,
		AssigneeID:     assigneeID,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, taskID uuid.UUID,
	req taskdomain.UpdateTaskRequest,
) (*taskdomain.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID, organizationID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	status := taskdomain.Status(req.Status)

	if req.Title != "" {
		task.Title = req.Title
	}

	if req.Notes != nil {
		task.Notes = *req.Notes
	}

	if req.AssigneeID.Set {
		assigneeID, err := s.resolveAssignee(ctx, organizationID, req.AssigneeID.Value)
		if err != nil {
			return nil, err
		}
		task.AssigneeID = assigneeID
	}

	task.Status = status
	task.CompletedAt = completedAtFor(status, task.CompletedAt, now)
	task.UpdatedAt = now
	task.Version = req.Version

	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, taskID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	if !actorRole.CanManageMembers() {
		return taskdomain.ErrForbidden
	}

	if _, err := s.tasks.GetByID(ctx, taskID, organizationID); err != nil {
		return err
	}

	return s.tasks.Delete(ctx, taskID, organizationID)
}

func (s *service) Convert(
	ctx context.Context,
	organizationID, ticketID uuid.UUID,
	_ orgdomain.Role,
	req taskdomain.ConvertTicketRequest,
) (*taskdomain.Task, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID, organizationID)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(ctx, req.ProjectID, organizationID)
	if err != nil {
		return nil, err
	}

	if project.ClientID != ticket.ClientID {
		return nil, projectdomain.ErrProjectNotFound
	}

	existing, err := s.tasks.ListByProjectID(ctx, organizationID, project.ID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	for _, task := range existing {
		if task.TicketID != nil && *task.TicketID == ticket.ID {
			return nil, taskdomain.ErrTicketAlreadyConverted
		}
	}

	now := time.Now().UTC()
	ticketIDCopy := ticket.ID
	createdBy := actorUserID(ctx)

	task := &taskdomain.Task{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		TicketID:       &ticketIDCopy,
		Title:          ticket.Title,
		Notes:          ticket.Body,
		Status:         taskdomain.StatusTodo,
		CreatedBy:      createdBy,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *service) resolveAssignee(
	ctx context.Context,
	organizationID uuid.UUID,
	assigneeID *uuid.UUID,
) (*uuid.UUID, error) {
	if assigneeID == nil {
		return nil, nil
	}

	membership, err := s.memberships.GetByUserID(ctx, *assigneeID)
	if err != nil {
		if errors.Is(err, orgdomain.ErrMembershipNotFound) {
			return nil, taskdomain.ErrAssigneeNotMember
		}
		return nil, fmt.Errorf("get assignee membership: %w", err)
	}

	if membership.OrganizationID != organizationID {
		return nil, taskdomain.ErrAssigneeNotMember
	}

	return assigneeID, nil
}

func actorUserID(ctx context.Context) *uuid.UUID {
	userID, ok := requestcontext.UserID(ctx)
	if !ok {
		return nil
	}
	return &userID
}

func completedAtFor(status taskdomain.Status, current *time.Time, now time.Time) *time.Time {
	if status != taskdomain.StatusDone {
		return nil
	}

	if current != nil {
		return current
	}

	completedAt := now
	return &completedAt
}
