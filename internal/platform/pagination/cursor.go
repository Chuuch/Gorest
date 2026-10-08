package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func Encode(c Cursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawStdEncoding.EncodeToString([]byte(raw))
}

func Decode(raw string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w", ErrInvalidCursor)
	}

	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("%w", ErrInvalidCursor)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, fmt.Errorf("%w", ErrInvalidCursor)
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Cursor{}, fmt.Errorf("%w", ErrInvalidCursor)
	}

	return Cursor{CreatedAt: createdAt, ID: id}, nil
}

func NextCursor[T any](items []T, limit int, key func(T) Cursor) ([]T, *string) {
	if limit < 1 {
		limit = 1
	}

	if len(items) <= limit {
		return items, nil
	}

	page := items[:limit]
	encoded := Encode(key(page[len(page)-1]))
	return page, &encoded
}
