package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/auth"
	"github.com/google/uuid"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func Auth(tokens auth.TokenManager) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(
				r.Context(),
				userIDContextKey,
				claims.UserID,
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

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}
