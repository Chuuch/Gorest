package usecase

import (
	"context"
	"fmt"
	"time"

	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	taskdomain "github.com/chuuch/gorest/internal/tasks/domain"
	taskrepository "github.com/chuuch/gorest/internal/tasks/repository"
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
}

type service struct {
	tasks    taskrepository.TaskRepository
	projects projectrepository.ProjectRepository
}

func NewService(
	tasks taskrepository.TaskRepository,
	projects projectrepository.ProjectRepository,
) Service {
	return &service{
		tasks:    tasks,
		projects: projects,
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
		CompletedAt: 		completedAtFor(status, nil, now),
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

	if err := s.tasks.Update(ctx, task); err != nil {
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
