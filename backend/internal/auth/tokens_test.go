package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/auth"
)

func TestTokenManagerRoundTrip(t *testing.T) {
	manager, err := auth.NewTokenManager("test-access-token-secret-at-least-32-bytes-long", 15*time.Minute)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}

	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	token, expiresAt, err := manager.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}
	if expiresAt.Before(time.Now().UTC()) {
		t.Fatal("expires_at must be in the future")
	}

	claims, err := manager.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.UserID != userID.String() {
		t.Fatalf("user_id = %q, want %q", claims.UserID, userID.String())
	}
	if claims.Role != "user" {
		t.Fatalf("role = %q, want user", claims.Role)
	}
}

func TestTokenManagerRejectsInvalidSecret(t *testing.T) {
	manager1, err := auth.NewTokenManager("secret-one-at-least-32-bytes-long-xxx", 15*time.Minute)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}
	manager2, err := auth.NewTokenManager("secret-two-at-least-32-bytes-long-xxx", 15*time.Minute)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}

	token, _, err := manager1.GenerateAccessToken(uuid.New(), "user")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if _, err := manager2.VerifyAccessToken(token); err == nil {
		t.Fatal("expected verify to fail with different secret")
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := auth.NewTokenManager("test-access-token-secret-at-least-32-bytes-long", -1*time.Second)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}

	token, _, err := manager.GenerateAccessToken(uuid.New(), "user")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if _, err := manager.VerifyAccessToken(token); err == nil {
		t.Fatal("expected verify to fail for expired token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("raw and hash must not be empty")
	}
	if raw == hash {
		t.Fatal("raw must not equal hash")
	}
	if auth.HashRefreshToken(raw) != hash {
		t.Fatal("hash refresh token must match generated hash")
	}
}
