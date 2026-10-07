package search_test

import (
	"strings"
	"testing"

	"github.com/chuuch/gorest/internal/search"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	require.Equal(t, "", search.Normalize("   "))
	require.Equal(t, "acme", search.Normalize("  acme  "))

	long := strings.Repeat("a", search.MaxQueryLength+20)
	got := search.Normalize(long)
	require.Equal(t, search.MaxQueryLength, len([]rune(got)))
}

func TestLikePattern(t *testing.T) {
	require.Equal(t, "", search.LikePattern(""))
	require.Equal(t, `%acme%`, search.LikePattern("acme"))
	require.Equal(t, `%100\%%`, search.LikePattern("100%"))
	require.Equal(t, `%a\_b%`, search.LikePattern("a_b"))
	require.Equal(t, `%a\\b%`, search.LikePattern(`a\b`))
}

func TestMatches(t *testing.T) {
	require.True(t, search.Matches("Ada Lovelace", ""))
	require.True(t, search.Matches("Ada Lovelace", "ada"))
	require.True(t, search.Matches("ada@example.com", "EXAMPLE"))
	require.False(t, search.Matches("Ada Lovalace", "zzz"))
}
