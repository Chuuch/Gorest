package handler

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	filedomain "github.com/chuuch/gorest/internal/files/domain"
	"github.com/chuuch/gorest/internal/files/usecase"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	projectdomain "github.com/chuuch/gorest/internal/projects/domain"
	"github.com/chuuch/gorest/internal/requestcontext"
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
	organizationID, _, _, ok := h.session(w, r)
	if !ok {
		return
	}

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	views, err := h.service.List(r.Context(), organizationID, projectID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	responses := make([]filedomain.FileResponse, 0, len(views))
	for _, view := range views {
		responses = append(responses, toResponse(view))
	}

	api.WriteJSON(w, http.StatusOK, responses)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, userID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	projectID, ok := h.projectID(w, r)
	if !ok {
		return
	}

	var req filedomain.CreateFileRequest

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
		projectID,
		userID,
		actorRole,
		req,
	)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, toResponse(view))
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, userID, actorRole, ok := h.session(w, r)
	if !ok {
		return
	}

	fileID, ok := h.fileID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		fileID,
		userID,
		actorRole,
	); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) projectID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	projectID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_project_id",
			"invalid project id",
		)
		return uuid.Nil, false
	}
	return projectID, true
}

func (h *Handler) fileID(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, bool) {
	fileID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_file_id",
			"invalid file id",
		)
		return uuid.Nil, false
	}
	return fileID, true
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, orgdomain.Role, bool) {
	organizationID, ok := requestcontext.OrganizationID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	userID, ok := requestcontext.UserID(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	role, ok := requestcontext.Role(r.Context())
	if !ok {
		api.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"unauthorized",
		)
		return uuid.Nil, uuid.Nil, "", false
	}

	return organizationID, userID, orgdomain.Role(role), true
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filedomain.ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")

	case errors.Is(err, filedomain.ErrUnsupportedContentType):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"unsupported_content_type",
			"unsupported content type",
		)

	case errors.Is(err, filedomain.ErrInvalidFilename):
		api.WriteError(
			w,
			http.StatusBadRequest,
			"invalid_filename",
			"invalid filename",
		)

	case errors.Is(err, projectdomain.ErrProjectNotFound):
		api.WriteError(
			w,
			http.StatusNotFound,
			"project_not_found",
			"project not found",
		)

	case errors.Is(err, filedomain.ErrFileNotFound):
		api.WriteError(w, http.StatusNotFound, "file_not_found", "file not found")

	default:
		api.WriteError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
		)
	}
}

func toResponse(view *filedomain.FileView) filedomain.FileResponse {
	return filedomain.FileResponse{
		ID:             view.File.ID,
		OrganizationID: view.File.OrganizationID,
		ProjectID:      view.File.ID,
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
