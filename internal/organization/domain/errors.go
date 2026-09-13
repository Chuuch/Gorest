package domain

import "errors"

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrMembershipNotFound   = errors.New("membership not found")
)
