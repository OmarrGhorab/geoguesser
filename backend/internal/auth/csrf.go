package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// CSRFManager generates and validates CSRF tokens.
type CSRFManager struct {
	secret []byte
}

// NewCSRFManager returns a CSRF manager.
func NewCSRFManager(secret string) (*CSRFManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("csrf secret must be at least 32 bytes")
	}
	return &CSRFManager{secret: []byte(secret)}, nil
}

// Generate creates a new signed CSRF token.
func (c *CSRFManager) Generate() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate csrf token: %w", err)
	}
	raw := hex.EncodeToString(b)
	return c.sign(raw), nil
}

// Validate checks a CSRF token value against its signature.
func (c *CSRFManager) Validate(token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	raw := parts[0]
	expected := c.sign(raw)
	return hmac.Equal([]byte(expected), []byte(token))
}

func (c *CSRFManager) sign(raw string) string {
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(raw))
	return raw + "." + hex.EncodeToString(mac.Sum(nil))
}

// CSRFTokenTTL is the lifetime of a CSRF token cookie.
const CSRFTokenTTL = 24 * time.Hour
