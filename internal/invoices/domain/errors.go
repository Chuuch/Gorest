package domain

import "errors"

var (
	ErrInvoiceNotFound          = errors.New("invoice not found")
	ErrInvoiceNumberExists      = errors.New("invoice number exists")
	ErrNoLineItems              = errors.New("no line items")
	ErrNotDraft                 = errors.New("invoice is not a draft")
	ErrNotSent                  = errors.New("invoice is not sent")
	ErrNoClientUsers            = errors.New("no client users")
	ErrForbidden                = errors.New("forbidden")
	ErrBillingProfileIncomplete = errors.New("billing profile incomplete")
)
