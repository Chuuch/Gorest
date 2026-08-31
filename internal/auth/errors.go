package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRefreshTokenInvalid = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)
