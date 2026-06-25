package auth_test

import (
	"strings"
	"testing"

	"github.com/raven/geoguess/backend/internal/auth"
)

func TestGuestSessionManagerRoundTrip(t *testing.T) {
	manager, err := auth.NewGuestSessionManager("test-guest-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("new guest manager failed: %v", err)
	}

	signed, hash, err := manager.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.HasPrefix(signed, "gst_") {
		t.Fatalf("signed token must start with gst_: %s", signed)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}

	raw, err := manager.Validate(signed)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if auth.HashGuestID(raw) != hash {
		t.Fatal("hash of validated guest id must match generated hash")
	}
}

func TestGuestSessionManagerRejectsTamperedToken(t *testing.T) {
	manager, err := auth.NewGuestSessionManager("test-guest-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("new guest manager failed: %v", err)
	}

	signed, _, err := manager.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	tampered := signed + "x"
	if _, err := manager.Validate(tampered); err == nil {
		t.Fatal("expected validation to fail for tampered token")
	}
}
