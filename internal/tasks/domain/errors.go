package domain

import "errors"

var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrTaskTitleExists        = errors.New("task title already exists")
	ErrTaskVersionMismatch    = errors.New("task version mismatch")
	ErrTicketAlreadyConverted = errors.New("ticket already converted")
	ErrAssigneeNotMember      = errors.New("assignee is not a member")
	ErrForbidden              = errors.New("forbidden")
)
