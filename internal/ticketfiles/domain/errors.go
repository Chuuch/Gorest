package domain

import "errors"

var (
	ErrFileNotFound           = errors.New("file not found")
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrInvalidFilename        = errors.New("invalid filename")
)
