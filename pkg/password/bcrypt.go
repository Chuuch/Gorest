package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const defaultCost = bcrypt.DefaultCost

var ErrPasswordMismatch = errors.New("password mismatch")

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{
		cost: defaultCost,
	}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
	[]byte(password),
	h.cost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *BcryptHasher) Compare(password, hash string) error {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}

		return err
	}
	return nil
}
