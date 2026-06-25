package config_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.AppEnv != "test" {
		t.Errorf("AppEnv = %q, want test", cfg.AppEnv)
	}
	if cfg.Version != "0.0.0" {
		t.Errorf("Version = %q, want 0.0.0", cfg.Version)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr default = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ReadTimeout <= 0 {
		t.Errorf("ReadTimeout must be positive, got %v", cfg.ReadTimeout)
	}
}

func TestValidateMissingEnv(t *testing.T) {
	cfg := config.Config{
		AppEnv:             "",
		Version:            "",
		HTTPAddr:           ":8080",
		DatabaseURL:        "postgres://localhost/db",
		RedisURL:           "redis://localhost:6379/0",
		AllowedOrigin:      "http://localhost:3000",
		ReadTimeout:        10,
		WriteTimeout:       10,
		IdleTimeout:        10,
		AccessTokenSecret:  "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret: "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:         "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret: "test-guest-secret-at-least-32-bytes-long",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing APP_ENV and VERSION")
	}
}

func TestValidateTimeouts(t *testing.T) {
	cfg := config.Config{
		AppEnv:             "test",
		Version:            "0.0.0",
		HTTPAddr:           ":8080",
		DatabaseURL:        "postgres://localhost/db",
		RedisURL:           "redis://localhost:6379/0",
		AllowedOrigin:      "http://localhost:3000",
		ReadTimeout:        0,
		WriteTimeout:       10,
		IdleTimeout:        10,
		AccessTokenSecret:  "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret: "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:         "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret: "test-guest-secret-at-least-32-bytes-long",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for non-positive timeout")
	}
}

func TestValidateProductionRequiresMetricsToken(t *testing.T) {
	cfg := config.Config{
		AppEnv:             "production",
		Version:            "0.0.0",
		HTTPAddr:           ":8080",
		DatabaseURL:        "postgres://localhost/db",
		RedisURL:           "redis://localhost:6379/0",
		AllowedOrigin:      "https://example.com",
		ReadTimeout:        10,
		WriteTimeout:       10,
		IdleTimeout:        10,
		AccessTokenSecret:  "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret: "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:         "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret: "test-guest-secret-at-least-32-bytes-long",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing production metrics token")
	}

	cfg.MetricsAuthToken = "secret-token"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config with metrics token, got %v", err)
	}
}
