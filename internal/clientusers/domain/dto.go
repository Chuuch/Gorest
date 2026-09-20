package domain

import (
	"time"

	clientdomain "github.com/chuuch/gorest/internal/client/domain"
	orgdomain "github.com/chuuch/gorest/internal/organization/domain"
	userdomain "github.com/chuuch/gorest/internal/user/domain"
	"github.com/google/uuid"
)

type CreateClientUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ClientUserResponse struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ClientID       uuid.UUID `json:"client_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type AuthResponse struct {
	AccessToken  string                         `json:"access_token"`
	User         userdomain.UserResponse        `json:"user"`
	Organization orgdomain.OrganizationResponse `json:"organization"`
	Client       clientdomain.ClientResponse    `json:"client"`
	Role         string                         `json:"role"`
}
