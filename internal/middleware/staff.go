package middleware

import (
	"net/http"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/requestcontext"
)

func Staff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := requestcontext.Role(r.Context())
		if !ok || (role != "owner" && role != "admin" && role != "member") {
			api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ClientPortal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := requestcontext.Role(r.Context())
		if !ok || role != "client" {
			api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}

		if _, ok := requestcontext.ClientID(r.Context()); !ok {
			api.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}
