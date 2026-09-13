package domain

import (
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
)

type RegisterRequest struct {
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=8"`
	OrganizationName string `json:"organization_name" validate:"required,min=2,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string                         `json:"access_token"`
	User         userdomain.UserResponse        `json:"user"`
	Organization orgdomain.OrganizationResponse `json:"organization"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
