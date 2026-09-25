package invites

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PurposeInvite = "invite"
	KindStaff     = "staff"
	KindPortal    = "portal"
)

var (
	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteExpired  = errors.New("invite expired")
	ErrInviteUsed     = errors.New("invite used")
)

type Token struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	Purpose   string
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
}

type IssueInput struct {
	UserID           uuid.UUID
	Email            string
	OrganizationName string
	ClientName       string
	Kind             string
}

func DiscardedPassword() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
