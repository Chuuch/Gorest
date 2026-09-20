package domain

import "errors"

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrForbidden      = errors.New("forbidden")
)
