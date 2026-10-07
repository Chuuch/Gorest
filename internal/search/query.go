package search

import (
	"strings"
	"unicode/utf8"
)

const MaxQueryLength = 100

func Normalize(raw string) string {
	q := strings.TrimSpace(raw)
	if q == "" {
		return ""
	}
	if utf8.RuneCountInString(q) > MaxQueryLength {
		runes := []rune(q)
		q = string(runes[:MaxQueryLength])
	}
	return q
}

func LikePattern(normalized string) string {
	if normalized == "" {
		return ""
	}
	return "%" + escapeLike(normalized) + "%"
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(s)
}

func Matches(haystack, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(normalizedQuery))
}
