package middleware

import (
	"net/http"
	"strings"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/auth/security"
	"github.com/chuuch/gorest/internal/requestcontext"
)

func Auth(tokens security.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearerToken(r)
			if err != nil {
				api.WriteError(
					w,
					http.StatusUnauthorized,
					"invalid_token",
					"invalid token",
				)
				return
			}

			claims, err := tokens.ParseAccessToken(token)
			if err != nil {
				api.WriteError(
					w,
					http.StatusUnauthorized,
					"invalid_token",
					"invalid token",
				)
				return
			}

			ctx := requestcontext.WithUserID(
				r.Context(),
				claims.UserID,
			)

			ctx = requestcontext.WithOrganizationID(
				ctx,
				claims.OrganizationID,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")

	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return "", ErrMissingBearerToken
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))

	if token == "" {
		return "", ErrMissingBearerToken
	}

	return token, nil
}
