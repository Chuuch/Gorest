package domain

import "errors"

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrForbidden       = errors.New("forbidden")
)
