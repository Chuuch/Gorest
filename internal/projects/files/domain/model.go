package domain

import (
	"time"

	"github.com/google/uuid"
)

const MaxSizeBytes = 10 * 1024 * 1024

type File struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProjectID      uuid.UUID
	UploadedBy     uuid.UUID
	ObjectKey      string
	Filename       string
	ContentType    string
	Size           int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FileView struct {
	File        *File
	UploadURL   string
	DownloadURL string
}

func AllowedContentType(contentType string) bool {
	switch contentType {
	case "application/pdf",
		"image/png",
		"image/jpeg",
		"image/webp",
		"text/plain",
		"application/zip":
		return true
	default:
		return false
	}
}

func ObjectKey(organizationID, projectID, fileID uuid.UUID) string {
	return organizationID.String() + "/" + projectID.String() + "/" + fileID.String()
}
