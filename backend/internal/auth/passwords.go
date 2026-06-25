package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) error
}

// BCryptHasher implements PasswordHasher using bcrypt.
type BCryptHasher struct {
	cost int
}

// NewBCryptHasher returns a bcrypt password hasher.
func NewBCryptHasher() *BCryptHasher {
	return &BCryptHasher{cost: bcrypt.DefaultCost}
}

// NewBCryptHasherWithCost returns a bcrypt password hasher with a custom cost.
func NewBCryptHasherWithCost(cost int) *BCryptHasher {
	return &BCryptHasher{cost: cost}
}

// Hash returns a bcrypt hash of the password.
func (b *BCryptHasher) Hash(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// Verify checks a password against a bcrypt hash.
func (b *BCryptHasher) Verify(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}
