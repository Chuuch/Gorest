package usecase

import (
	"context"
	"fmt"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
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
	tasks    taskrepository.TaskRepository
	projects projectrepository.ProjectRepository
	tickets  ticketrepository.TicketRepository
}

func NewService(
	tasks taskrepository.TaskRepository,
	projects projectrepository.ProjectRepository,
	tickets ticketrepository.TicketRepository,
) Service {
	return &service{
		tasks:    tasks,
		projects: projects,
		tickets:  tickets,
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

func (s *service) Create(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
	actorRole orgdomain.Role,
	req taskdomain.CreateTaskRequest,
) (*taskdomain.Task, error) {
	if !actorRole.CanManageMembers() {
		return nil, taskdomain.ErrForbidden
	}

	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	status := taskdomain.Status(req.Status)

	task := &taskdomain.Task{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Title:          req.Title,
		Notes:          req.Notes,
		Status:         status,
		CompletedAt:    completedAtFor(status, nil, now),
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
	actorRole orgdomain.Role,
	req taskdomain.ConvertTicketRequest,
) (*taskdomain.Task, error) {
	if !actorRole.CanManageMembers() {
		return nil, taskdomain.ErrForbidden
	}

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

	task := &taskdomain.Task{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		TicketID:       &ticketIDCopy,
		Title:          ticket.Title,
		Notes:          ticket.Body,
		Status:         taskdomain.StatusTodo,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
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
