package storage

import "context"

type ObjectStore interface {
	PresignPut(ctx context.Context, key, contentType string) (string, error)
	PresignGet(ctx context.Context, key, filename string) (string, error)
}
