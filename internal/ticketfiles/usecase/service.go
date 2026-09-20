package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chuuch/gorest/internal/storage"
	ticketfiledomain "github.com/chuuch/gorest/internal/ticketfiles/domain"
	ticketfilerepository "github.com/chuuch/gorest/internal/ticketfiles/repository"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	ticketrepository "github.com/chuuch/gorest/internal/tickets/repository"
	"github.com/google/uuid"
)

type Service interface {
	List(
		ctx context.Context,
		organizationID, ticketID, portalClientID uuid.UUID,
	) ([]*ticketfiledomain.FileView, error)
	Create(
		ctx context.Context,
		organizationID, ticketID, userID, portalClientID uuid.UUID,
		req ticketfiledomain.CreateFileRequest,
	) (*ticketfiledomain.FileView, error)
}

type service struct {
	files   ticketfilerepository.TicketFileRepository
	tickets ticketrepository.TicketRepository
	store   storage.ObjectStore
}

func NewService(
	files ticketfilerepository.TicketFileRepository,
	tickets ticketrepository.TicketRepository,
	store storage.ObjectStore,
) Service {
	return &service{
		files:   files,
		tickets: tickets,
		store:   store,
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
) ([]*ticketfiledomain.FileView, error) {
	if _, err := s.visibleTicket(ctx, organizationID, ticketID, portalClientID); err != nil {
		return nil, err
	}

	files, err := s.files.ListByTicketID(ctx, organizationID, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list ticket files: %w", err)
	}

	views := make([]*ticketfiledomain.FileView, 0, len(files))

	for _, file := range files {
		downloadURL, err := s.store.PresignGet(ctx, file.ObjectKey, file.Filename)
		if err != nil {
			return nil, err
		}

		views = append(views, &ticketfiledomain.FileView{
			File:        file,
			DownloadURL: downloadURL,
		})
	}

	return views, nil
}

func (s *service) Create(
	ctx context.Context,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
	req ticketfiledomain.CreateFileRequest,
) (*ticketfiledomain.FileView, error) {
	if strings.Contains(req.Filename, "/") ||
		strings.Contains(req.Filename, "\\") ||
		strings.Contains(req.Filename, "..") {
		return nil, ticketfiledomain.ErrInvalidFilename
	}

	if !ticketfiledomain.AllowedContentType(req.ContentType) {
		return nil, ticketfiledomain.ErrUnsupportedContentType
	}

	if _, err := s.visibleTicket(ctx, organizationID, ticketID, portalClientID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fileID := uuid.New()

	file := &ticketfiledomain.File{
		ID:             fileID,
		OrganizationID: organizationID,
		TicketID:       ticketID,
		UploadedBy:     userID,
		ObjectKey:      ticketfiledomain.ObjectKey(organizationID, ticketID, fileID),
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

	return &ticketfiledomain.FileView{
		File:      file,
		UploadURL: uploadURL,
	}, nil
}
