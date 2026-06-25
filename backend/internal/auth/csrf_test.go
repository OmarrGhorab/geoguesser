package auth_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/auth"
)

func TestCSRFManagerRoundTrip(t *testing.T) {
	manager, err := auth.NewCSRFManager("test-csrf-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("new csrf manager failed: %v", err)
	}

	token, err := manager.Generate()
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}

	if !manager.Validate(token) {
		t.Fatal("expected token to be valid")
	}
}

func TestCSRFManagerRejectsInvalidToken(t *testing.T) {
	manager, err := auth.NewCSRFManager("test-csrf-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("new csrf manager failed: %v", err)
	}

	if manager.Validate("invalid-token") {
		t.Fatal("expected invalid token to be rejected")
	}

	token, _ := manager.Generate()
	if manager.Validate(token + "x") {
		t.Fatal("expected tampered token to be rejected")
	}
}
