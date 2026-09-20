package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
	ticketfiledomain "github.com/chuuch/gorest/internal/ticketfiles/domain"
	"github.com/chuuch/gorest/internal/ticketfiles/usecase"
	ticketdomain "github.com/chuuch/gorest/internal/tickets/domain"
	"github.com/chuuch/gorest/internal/validation"
	"github.com/google/uuid"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeList(w, r, organizationID, ticketID, uuid.Nil)
}

func (h *Handler) ListPortal(w http.ResponseWriter, r *http.Request) {
	organizationID, _, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeList(w, r, organizationID, ticketID, clientID)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, ok := h.staffSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeCreate(w, r, organizationID, ticketID, userID, uuid.Nil)
}

func (h *Handler) CreatePortal(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, clientID, ok := h.portalSession(w, r)
	if !ok {
		return
	}

	ticketID, ok := h.ticketID(w, r)
	if !ok {
		return
	}

	h.writeCreate(w, r, organizationID, ticketID, userID, clientID)
}

func (h *Handler) writeList(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, ticketID, portalClientID uuid.UUID,
) {
	views, err := h.service.List(r.Context(), organizationID, ticketID, portalClientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]ticketfiledomain.FileResponse, 0, len(views))
	for _, view := range views {
		responses = append(responses, toResponse(view))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) writeCreate(
	w http.ResponseWriter,
	r *http.Request,
	organizationID, ticketID, userID, portalClientID uuid.UUID,
) {
	var req ticketfiledomain.CreateFileRequest

	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	if err := validation.Struct(req); err != nil {
		api.WriteValidationError(w, validation.Errors(err))
		return
	}

	view, err := h.service.Create(
		r.Context(),
		organizationID,
		ticketID,
		userID,
		portalClientID,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(view))
}

func (h *Handler) ticketID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	ticketID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_ticket_id",
			"invalid ticket id",
		)
		return uuid.Nil, false
	}
	return ticketID, true
}

func (h *Handler) staffSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, false
	}
	return organizationID, userID, true
}

func (h *Handler) portalSession(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	organizationID, userID, ok := h.staffSession(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	clientID, ok := requestcontext.ClientID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	return organizationID, userID, clientID, true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ticketfiledomain.ErrUnsupportedContentType):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"unsupported_content_type",
			"unsupported content type",
		)

	case errors.Is(err, ticketfiledomain.ErrInvalidFilename):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_filename",
			"invalid filename",
		)

	case errors.Is(err, ticketdomain.ErrTicketNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"ticket_not_found",
			"ticket not found",
		)

	case errors.Is(err, ticketfiledomain.ErrFileNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"file_not_found",
			"file not found",
		)

	default:
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
		)
	}
}

func toResponse(view *ticketfiledomain.FileView) ticketfiledomain.FileResponse {
	return ticketfiledomain.FileResponse{
		ID:             view.File.ID,
		OrganizationID: view.File.OrganizationID,
		TicketID:       view.File.TicketID,
		UploadedBy:     view.File.UploadedBy,
		Filename:       view.File.Filename,
		ContentType:    view.File.ContentType,
		Size:           view.File.Size,
		CreatedAt:      view.File.CreatedAt,
		UpdatedAt:      view.File.UpdatedAt,
		UploadURL:      view.UploadURL,
		DownloadURL:    view.DownloadURL,
	}
}
