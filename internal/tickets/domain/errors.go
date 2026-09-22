package domain

import "errors"

var (
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketVersionMismatch = errors.New("ticket version mismatch")
	ErrForbidden             = errors.New("forbidden")
)
