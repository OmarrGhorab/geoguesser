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

// GuestSessionManager signs and validates guest session identifiers.
type GuestSessionManager struct {
	secret []byte
}

// NewGuestSessionManager returns a guest session manager.
func NewGuestSessionManager(secret string) (*GuestSessionManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("guest session secret must be at least 32 bytes")
	}
	return &GuestSessionManager{secret: []byte(secret)}, nil
}

// Generate creates a new signed guest session token and returns its raw value and hash.
func (g *GuestSessionManager) Generate() (string, string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate guest id: %w", err)
	}
	raw := "gst_" + hex.EncodeToString(b)
	signed := g.sign(raw)
	return signed, HashGuestID(raw), nil
}

// Validate checks the signature and returns the raw guest id.
func (g *GuestSessionManager) Validate(signed string) (string, error) {
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return "", ErrGuestSessionInvalid
	}
	raw := parts[0]
	expected := g.sign(raw)
	if !hmac.Equal([]byte(expected), []byte(signed)) {
		return "", ErrGuestSessionInvalid
	}
	return raw, nil
}

// HashGuestID returns a SHA-256 hash of a guest id.
func HashGuestID(guestID string) string {
	sum := sha256.Sum256([]byte(guestID))
	return hex.EncodeToString(sum[:])
}

func (g *GuestSessionManager) sign(raw string) string {
	mac := hmac.New(sha256.New, g.secret)
	_, _ = mac.Write([]byte(raw))
	return raw + "." + hex.EncodeToString(mac.Sum(nil))
}

// GuestSessionTTL is the lifetime of a signed guest session cookie.
const GuestSessionTTL = 365 * 24 * time.Hour
