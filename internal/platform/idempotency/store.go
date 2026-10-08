package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrKeyMismatch   = errors.New("idempotency key reused with different request")
	ErrKeyInProgress = errors.New("idempotency key request already in progress")
)

type Record struct {
	OrganizationID uuid.UUID
	Key            string
	Method         string
	Path           string
	RequestHash    string
	StatusCode     *int
	ResponseBody   []byte
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

func (r Record) Complete() bool {
	return r.StatusCode != nil
}

type Store interface {
	// TryClaim inserts a pending row. Returns the existing record when the key
	// is already claimed (caller decides replay vs conflict). Returns nil record
	// when this caller owns the claim.
	TryClaim(
		ctx context.Context,
		organizationID uuid.UUID,
		key string,
		method string,
		path string,
		requestHash string,
		expiresAt time.Time,
	) (existing *Record, err error)

	Complete(
		ctx context.Context,
		organizationID uuid.UUID,
		key string,
		statusCode int,
		responseBody []byte,
	) error

	Release(ctx context.Context, organizationID uuid.UUID, key string) error
}
