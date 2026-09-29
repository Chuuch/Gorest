package domain

import "errors"

var (
	ErrTimeEntryNotFound = errors.New("time entry not found")
	ErrForbidden         = errors.New("forbidden")
)
