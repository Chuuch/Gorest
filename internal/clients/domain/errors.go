package domain

import "errors"

var (
	ErrClientNotFound    = errors.New("client not found")
	ErrClientNameExists  = errors.New("client name already exists")
	ErrClientHasInvoices = errors.New("client has invoices")
	ErrForbidden         = errors.New("forbidden")
)
