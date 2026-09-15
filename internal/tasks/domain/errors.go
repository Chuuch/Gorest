package domain

import "errors"

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskTitleExists = errors.New("task title already exists")
	ErrForbidden = errors.New("forbidden")
)
