package domain

import "errors"

var (
	ErrClientNotFound   = errors.New("client not found")
	ErrClientNameExists = errors.New("client name already exists")
	ErrForbidden        = errors.New("forbidden")
)
