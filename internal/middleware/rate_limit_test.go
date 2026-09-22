package middleware_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/api"
	"github.com/chuuch/gorest/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestLimiter_AllowsUpToLimit(t *testing.T) {
	limiter := middleware.NewLimiter(2, time.Minute)

	require.True(t, limiter.Allow("login:10.0.0.1:a@example.com"))
	require.True(t, limiter.Allow("login:10.0.0.1:a@example.com"))
	require.False(t, limiter.Allow("login:10.0.0.1:a@example.com"))
}

func TestLimiter_SeparateKeys(t *testing.T) {
	limiter := middleware.NewLimiter(1, time.Minute)

	require.True(t, limiter.Allow("login:10.0.0.1:a@example.com"))
	require.True(t, limiter.Allow("login:10.0.0.1:b@example.com"))
	require.True(t, limiter.Allow("login:10.0.0.2:a@example.com"))
	require.False(t, limiter.Allow("login:10.0.0.1:a@example.com"))
}

func TestLimiter_WindowExpires(t *testing.T) {
	limiter := middleware.NewLimiter(1, 20*time.Millisecond)

	require.True(t, limiter.Allow("k"))
	require.False(t, limiter.Allow("k"))

	time.Sleep(30 * time.Millisecond)

	require.True(t, limiter.Allow("k"))
}

func TestLogin_RateLimitPreservesBody(t *testing.T) {
	limiter := middleware.NewLimiter(10, time.Minute)
	var gotEmail string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		gotEmail = req.Email
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Login(next)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"Ada@Example.com","password":"secret"}`),
	)
	req.RemoteAddr = "10.0.0.1:4444"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "Ada@Example.com", gotEmail)
}

func TestLogin_TooManyRequests(t *testing.T) {
	limiter := middleware.NewLimiter(2, time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Login(next)

	for range 2 {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/login",
			strings.NewReader(`{"email":"ada@example.com","password":"secret"}`),
		)
		req.RemoteAddr = "10.0.0.1:4444"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"ada@example.com","password":"secret"}`),
	)
	req.RemoteAddr = "10.0.0.1:4444"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	var response api.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "rate_limited", response.Error.Code)
	require.Equal(t, "too many login attempts", response.Error.Message)
}

func TestRefresh_TooManyRequests(t *testing.T) {
	limiter := middleware.NewLimiter(1, time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Refresh(next)

	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	okReq.RemoteAddr = "10.0.0.1:4444"
	okRec := httptest.NewRecorder()
	handler.ServeHTTP(okRec, okReq)
	require.Equal(t, http.StatusOK, okRec.Code)

	blockedReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	blockedReq.RemoteAddr = "10.0.0.1:4444"
	blockedRec := httptest.NewRecorder()
	handler.ServeHTTP(blockedRec, blockedReq)

	require.Equal(t, http.StatusTooManyRequests, blockedRec.Code)

	var response api.ErrorResponse
	require.NoError(t, json.Unmarshal(blockedRec.Body.Bytes(), &response))
	require.Equal(t, "rate_limited", response.Error.Code)
}

func TestLogin_UsesForwardedFor(t *testing.T) {
	limiter := middleware.NewLimiter(1, time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := limiter.Login(next)

	first := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"ada@example.com","password":"secret"}`),
	)
	first.RemoteAddr = "10.0.0.1:4444"
	first.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	require.Equal(t, http.StatusOK, firstRec.Code)

	second := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"ada@example.com","password":"secret"}`),
	)
	second.RemoteAddr = "10.0.0.2:5555"
	second.Header.Set("X-Forwarded-For", "203.0.113.9")
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)

	require.Equal(t, http.StatusTooManyRequests, secondRec.Code)
}
