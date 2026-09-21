package domain

import "errors"

var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrTaskTitleExists        = errors.New("task title already exists")
	ErrTicketAlreadyConverted = errors.New("ticket already converted")
	ErrForbidden              = errors.New("forbidden")
)
