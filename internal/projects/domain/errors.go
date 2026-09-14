package domain

import "errors"

var (
	ErrProjectNotFound   = errors.New("project not found")
	ErrProjectNameExists = errors.New("project name already exists")
	ErrForbidden         = errors.New("forbidden")
)
