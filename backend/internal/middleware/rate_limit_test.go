package middleware

import (
	"net/http"
	"testing"
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
