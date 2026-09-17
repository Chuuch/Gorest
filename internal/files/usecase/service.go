package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	filedomain "github.com/chuuch/gorest/internal/files/domain"
	filerepository "github.com/chuuch/gorest/internal/files/repository"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectrepository "github.com/chuuch/gorest/internal/projects/repository"
	"github.com/chuuch/gorest/internal/storage"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, projectID uuid.UUID,
	) ([]*filedomain.FileView, error)
	Create(
		ctx context.Context,
		organizationID, projectID, userID uuid.UUID,
		actorRole orgdomain.Role,
		req filedomain.CreateFileRequest,
	) (*filedomain.FileView, error)
}

type service struct {
	files    filerepository.FileRepository
	projects projectrepository.ProjectRepository
	store    storage.ObjectStore
}

func NewService(
	files filerepository.FileRepository,
	projects projectrepository.ProjectRepository,
	store storage.ObjectStore,
) Service {
	return &service{
		files:    files,
		projects: projects,
		store:    store,
	}
}

func (s *service) List(
	ctx context.Context,
	organizationID, projectID uuid.UUID,
) ([]*filedomain.FileView, error) {
	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	files, err := s.files.ListByProjectID(ctx, organizationID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	views := make([]*filedomain.FileView, 0, len(files))

	for _, file := range files {
		downloadURL, err := s.store.PresignGet(ctx, file.ObjectKey, file.Filename)
		if err != nil {
			return nil, err
		}

		views = append(views, &filedomain.FileView{
			File:        file,
			DownloadURL: downloadURL,
		})
	}

	return views, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, projectID, userID uuid.UUID,
	actorRole orgdomain.Role,
	req filedomain.CreateFileRequest,
) (*filedomain.FileView, error) {
	if !actorRole.CanManageMembers() {
		return nil, filedomain.ErrForbidden
	}

	if strings.Contains(req.Filename, "/") ||
		strings.Contains(req.Filename, "\\") ||
		strings.Contains(req.Filename, "..") {
		return nil, filedomain.ErrInvalidFilename
	}

	if !filedomain.AllowedContentType(req.ContentType) {
		return nil, filedomain.ErrUnsupportedContentType
	}

	if _, err := s.projects.GetByID(ctx, projectID, organizationID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fileID := uuid.New()

	file := &filedomain.File{
		ID:             fileID,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		UploadedBy:     userID,
		ObjectKey:      filedomain.ObjectKey(organizationID, projectID, fileID),
		Filename:       req.Filename,
		ContentType:    req.ContentType,
		Size:           req.Size,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.files.Create(ctx, file); err != nil {
		return nil, err
	}

	uploadURL, err := s.store.PresignPut(ctx, file.ObjectKey, file.ContentType)
	if err != nil {
		return nil, err
	}

	return &filedomain.FileView{
		File:      file,
		UploadURL: uploadURL,
	}, nil
}
