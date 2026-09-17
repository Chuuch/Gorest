package domain

import "errors"

var (
	ErrFileNotFound = errors.New("file not found")
	ErrForbidden = errors.New("forbidden")
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrInvalidFilename = errors.New("invalid filename")
)
