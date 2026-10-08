package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateFileRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"content_type" validate:"required,min=3,max=100"`
	Size        int    `json:"size" validate:"required,min=1,max=10485760"`
}

type FileResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	TicketID       uuid.UUID `json:"ticket_id"`
	UploadedBy     uuid.UUID `json:"uploaded_by"`
	Filename       string    `json:"filename"`
	ContentType    string    `json:"content_type"`
	Size           int       `json:"size"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	UploadURL      string    `json:"upload_url,omitempty"`
	DownloadURL    string    `json:"download_url,omitempty"`
}
