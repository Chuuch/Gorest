package usecase

import (
	"context"
	"fmt"
	"time"

	commentdomain "github.com/chuuch/gorest/internal/comments/domain"
	commentrepository "github.com/chuuch/gorest/internal/comments/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	taskrepository "github.com/chuuch/gorest/internal/tasks/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context, organizationID, taskID uuid.UUID) ([]*commentdomain.Comment, error)
	Create(
		ctx context.Context,
		organizationID, tsakID, userID uuid.UUID,
		req commentdomain.CreateCommentRequest,
	) (*commentdomain.Comment, error)
	Update(
		ctx context.Context,
		organizationID, commentID, actorID uuid.UUID,
		req commentdomain.UpdateCommentRequest,
	) (*commentdomain.Comment, error)
	Delete(
		ctx context.Context,
		organizationID, commentID, actorID uuid.UUID,
		actorRole orgdomain.Role,
	) error
}

type service struct {
	comments commentrepository.CommentRepository
	tasks    taskrepository.TaskRepository
}

func NewService(
	comments commentrepository.CommentRepository,
	tasks taskrepository.TaskRepository,
) Service {
	return &service{
		comments: comments,
		tasks:    tasks,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, taskID uuid.UUID,
) ([]*commentdomain.Comment, error) {
	if _, err := s.tasks.GetByID(ctx, taskID, organizationID); err != nil {
		return nil, err
	}

	comments, err := s.comments.ListByTaskID(ctx, organizationID, taskID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	return comments, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, taskID, userID uuid.UUID,
	req commentdomain.CreateCommentRequest,
) (*commentdomain.Comment, error) {
	if _, err := s.tasks.GetByID(ctx, taskID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	comment := &commentdomain.Comment{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		TaskID:         taskID,
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

func (s *service) Update(
	ctx context.Context,
	organizationID, commentID, actorID uuid.UUID,
	req commentdomain.UpdateCommentRequest,
) (*commentdomain.Comment, error) {
	comment, err := s.comments.GetByID(ctx, commentID, organizationID)
	if err != nil {
		return nil, err
	}

	if comment.UserID != actorID {
		return nil, commentdomain.ErrForbidden
	}

	comment.Body = req.Body
	comment.UpdatedAt = time.Now().UTC()

	if err := s.comments.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *service) Delete(
	ctx context.Context,
	organizationID, commentID, actorID uuid.UUID,
	actorRole orgdomain.Role,
) error {
	comment, err := s.comments.GetByID(ctx, commentID, organizationID)
	if err != nil {
		return err
	}

	if comment.UserID != actorID && !actorRole.CanManageMembers() {
		return commentdomain.ErrForbidden
	}

	return s.comments.Delete(ctx, commentID, organizationID)
}
