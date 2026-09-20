package domain

import "errors"

var (
	ErrClientUserNotFound      = errors.New("client user not found")
	ErrClientUserAlreadyExists = errors.New("client user already exists")
	ErrUserIsStaff             = errors.New("user is staff")
	ErrForbidden               = errors.New("forbidden")
)
