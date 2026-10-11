package domain

import "errors"

var (
	ErrForbidden          = errors.New("forbidden")
	ErrEstimateNotFound   = errors.New("estimate not found")
	ErrInvalidInput       = errors.New("invalid estimate input")
	ErrIntakeNotFound     = errors.New("intake not found")
	ErrIntakeExpired      = errors.New("intake expired")
	ErrIntakeSubmitted    = errors.New("intake already submitted")
	ErrIntakeConsumed     = errors.New("intake already consumed")
	ErrIntakeNotPending   = errors.New("intake is not pending")
	ErrIntakeNotSubmitted = errors.New("intake is not submitted")
	ErrInvalidIntakeScope = errors.New("invalid intake scope")
)
