package middleware

import "errors"

var ErrMissingBearerToken = errors.New("missing bearer token")
