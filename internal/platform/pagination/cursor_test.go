package pagination_test

import (
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/platform/pagination"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()

	original := pagination.Cursor{
		CreatedAt: time.Date(2026, 10, 6, 12, 0, 0, 123456789, time.UTC),
		ID:        uuid.MustParse("11111111-1111-4111-8111-111111111111"),
	}

	decoded, err := pagination.Decode(pagination.Encode(original))
	require.NoError(t, err)
	require.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
	require.Equal(t, original.ID, decoded.ID)
}

func TestDecodeInvalid(t *testing.T) {
	t.Parallel()

	_, err := pagination.Decode("not-a-cursor")
	require.ErrorIs(t, err, pagination.ErrInvalidCursor)
}

func TestNextCursor(t *testing.T) {
	t.Parallel()

	type row struct {
		createdAt time.Time
		id        uuid.UUID
	}

	a := row{time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")}
	b := row{time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")}
	c := row{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), uuid.MustParse("cccccccc-cccc-4ccc-8ccc-cccccccccccc")}

	page, next := pagination.NextCursor([]row{a, b, c}, 2, func(r row) pagination.Cursor {
		return pagination.Cursor{CreatedAt: r.createdAt, ID: r.id}
	})
	require.Len(t, page, 2)
	require.NotNil(t, next)

	decoded, err := pagination.Decode(*next)
	require.NoError(t, err)
	require.Equal(t, b.id, decoded.ID)

	page, next = pagination.NextCursor([]row{a, b}, 2, func(r row) pagination.Cursor {
		return pagination.Cursor{CreatedAt: r.createdAt, ID: r.id}
	})
	require.Len(t, page, 2)
	require.Nil(t, next)
}
