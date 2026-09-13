package domain

import "errors"

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrMembershipNotFound   = errors.New("membership not found")
	ErrMemberAlreadyExists  = errors.New("member already exists")
	ErrForbidden            = errors.New("forbidden")
	ErrCannotCreateOwner    = errors.New("cannot create owner")
)
