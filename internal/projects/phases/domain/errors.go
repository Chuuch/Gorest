package domain

import "errors"

var (
	ErrPhaseNotFound        = errors.New("phase not found")
	ErrPhaseNameExists      = errors.New("phase name already exists")
	ErrPhaseSortExists      = errors.New("phase sort order already exists")
	ErrPhaseVersionMismatch = errors.New("phase version mismatch")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidDates         = errors.New("invalid phase dates")
)
