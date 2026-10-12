package domain

import "errors"

var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrTaskTitleExists        = errors.New("task title already exists")
	ErrTaskVersionMismatch    = errors.New("task version mismatch")
	ErrInvalidTaskDates       = errors.New("invalid task dates")
	ErrPhaseNotFound          = errors.New("phase not found")
	ErrPhaseWrongProject      = errors.New("phase does not belong to project")
	ErrTicketAlreadyConverted = errors.New("ticket already converted")
	ErrAssigneeNotMember      = errors.New("assignee is not a member")
	ErrForbidden              = errors.New("forbidden")
)
