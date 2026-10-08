package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

// LimitAndCursor parses limit and cursor query params.
// On invalid input it writes a 400 via writeError and returns ok=false.
func LimitAndCursor(
	w http.ResponseWriter,
	r *http.Request,
	writeError func(w http.ResponseWriter, status int, code, message string),
) (limit int, cursor *Cursor, ok bool) {
	limit = DefaultLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "invalid limit")
			return 0, nil, false
		}
		limit = parsed
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if rawCursor := r.URL.Query().Get("cursor"); rawCursor != "" {
		decoded, err := Decode(rawCursor)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_cursor", "invalid cursor")
			return 0, nil, false
		}
		return limit, &decoded, true
	}

	return limit, nil, true
}
