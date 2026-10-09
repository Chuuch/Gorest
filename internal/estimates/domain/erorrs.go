package domain

import "errors"

var (
	ErrForbidden        = errors.New("forbidden")
	ErrEstimateNotFound = errors.New("estimate not found")
	ErrInvalidInput     = errors.New("invalid estimate input")
)
