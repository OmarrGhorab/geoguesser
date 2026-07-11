package middleware

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/raven/geoguess/backend/internal/session"
)

func TestRateLimitByIPUsesRemoteAddrHost(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	if err != nil {
		t.Fatalf("request setup failed: %v", err)
	}
	req.RemoteAddr = "203.0.113.10:49152"
	req.Header.Set("X-Forwarded-For", "198.51.100.25")
	req.Header.Set("X-Real-IP", "198.51.100.26")

	key := RateLimitByIP("auth")(req)
	if key != "auth:203.0.113.10" {
		t.Fatalf("key = %q, want remote address host", key)
	}
}

func TestRateLimitByCookieFallsBackToRemoteAddrHost(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	if err != nil {
		t.Fatalf("request setup failed: %v", err)
	}
	req.RemoteAddr = "[2001:db8::1]:49152"

	key := RateLimitByCookie("auth", "missing")(req)
	if key != "auth:2001:db8::1" {
		t.Fatalf("key = %q, want remote address host", key)
	}
}

func TestRateLimitByRegisteredUserIsolatesUsersAndHashesIdentity(t *testing.T) {
	userA := "11111111-1111-1111-1111-111111111111"
	userB := "22222222-2222-2222-2222-222222222222"

	reqA, err := http.NewRequest(http.MethodPost, "/api/v1/matchmaking/queue", nil)
	if err != nil {
		t.Fatalf("request setup failed: %v", err)
	}
	reqA.RemoteAddr = "203.0.113.10:49152"
	reqA = reqA.WithContext(context.WithValue(reqA.Context(), sessionContextKey{}, &session.Context{
		Kind:   session.KindUser,
		UserID: &userA,
		Role:   "user",
	}))

	reqB, err := http.NewRequest(http.MethodPost, "/api/v1/matchmaking/queue", nil)
	if err != nil {
		t.Fatalf("request setup failed: %v", err)
	}
	reqB.RemoteAddr = "203.0.113.10:49153"
	reqB = reqB.WithContext(context.WithValue(reqB.Context(), sessionContextKey{}, &session.Context{
		Kind:   session.KindUser,
		UserID: &userB,
		Role:   "user",
	}))

	keyA := RateLimitByRegisteredUser("mm-cmd")(reqA)
	keyB := RateLimitByRegisteredUser("mm-cmd")(reqB)
	if keyA == keyB {
		t.Fatalf("expected distinct keys for different users sharing an IP, got %q", keyA)
	}
	if strings.Contains(keyA, userA) || strings.Contains(keyB, userB) {
		t.Fatalf("rate-limit key leaked raw user id: %q / %q", keyA, keyB)
	}
	if strings.Contains(keyA, "access_token") || strings.Contains(keyA, "Bearer") {
		t.Fatalf("rate-limit key appears to contain token material: %q", keyA)
	}
	if !strings.HasPrefix(keyA, "mm-cmd:user:") {
		t.Fatalf("keyA = %q, want hashed user prefix", keyA)
	}
}

func TestRateLimitByRegisteredUserFallsBackToIP(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/api/v1/matchmaking/status", nil)
	if err != nil {
		t.Fatalf("request setup failed: %v", err)
	}
	req.RemoteAddr = "198.51.100.20:443"
	key := RateLimitByRegisteredUser("mm-status")(req)
	if key != "mm-status:ip:198.51.100.20" {
		t.Fatalf("key = %q, want IP fallback", key)
	}
}
