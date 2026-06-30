package config_test

import (
	"testing"
	"time"

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
	if cfg.RoomReconnectGrace != 30*time.Second {
		t.Errorf("RoomReconnectGrace default = %v, want 30s", cfg.RoomReconnectGrace)
	}
	if cfg.RoomHeartbeatInterval != 10*time.Second {
		t.Errorf("RoomHeartbeatInterval default = %v, want 10s", cfg.RoomHeartbeatInterval)
	}
	if cfg.RoomPresenceTTL != 30*time.Second {
		t.Errorf("RoomPresenceTTL default = %v, want 30s", cfg.RoomPresenceTTL)
	}
}

func TestLoadRoomRealtimeConfig(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")
	t.Setenv("ROOM_RECONNECT_GRACE_SECONDS", "45")
	t.Setenv("ROOM_HEARTBEAT_INTERVAL_SECONDS", "5")
	t.Setenv("ROOM_PRESENCE_TTL_SECONDS", "20")
	t.Setenv("ROOM_REALTIME_ALLOWED_HOST", "localhost:3000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.RoomReconnectGrace != 45*time.Second {
		t.Errorf("RoomReconnectGrace = %v, want 45s", cfg.RoomReconnectGrace)
	}
	if cfg.RoomHeartbeatInterval != 5*time.Second {
		t.Errorf("RoomHeartbeatInterval = %v, want 5s", cfg.RoomHeartbeatInterval)
	}
	if cfg.RoomPresenceTTL != 20*time.Second {
		t.Errorf("RoomPresenceTTL = %v, want 20s", cfg.RoomPresenceTTL)
	}
	if cfg.RoomRealtimeAllowedHost != "localhost:3000" {
		t.Errorf("RoomRealtimeAllowedHost = %q, want localhost:3000", cfg.RoomRealtimeAllowedHost)
	}
}

func TestValidateMissingEnv(t *testing.T) {
	cfg := config.Config{
		AppEnv:                "",
		Version:               "",
		HTTPAddr:              ":8080",
		DatabaseURL:           "postgres://localhost/db",
		RedisURL:              "redis://localhost:6379/0",
		AllowedOrigin:         "http://localhost:3000",
		ReadTimeout:           10,
		WriteTimeout:          10,
		IdleTimeout:           10,
		AccessTokenSecret:     "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:    "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:            "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:    "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:    30 * time.Second,
		RoomHeartbeatInterval: 10 * time.Second,
		RoomPresenceTTL:       30 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing APP_ENV and VERSION")
	}
}

func TestValidateTimeouts(t *testing.T) {
	cfg := config.Config{
		AppEnv:                "test",
		Version:               "0.0.0",
		HTTPAddr:              ":8080",
		DatabaseURL:           "postgres://localhost/db",
		RedisURL:              "redis://localhost:6379/0",
		AllowedOrigin:         "http://localhost:3000",
		ReadTimeout:           0,
		WriteTimeout:          10,
		IdleTimeout:           10,
		AccessTokenSecret:     "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:    "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:            "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:    "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:    30 * time.Second,
		RoomHeartbeatInterval: 10 * time.Second,
		RoomPresenceTTL:       30 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for non-positive timeout")
	}
}

func TestValidateProductionRequiresMetricsToken(t *testing.T) {
	cfg := config.Config{
		AppEnv:                "production",
		Version:               "0.0.0",
		HTTPAddr:              ":8080",
		DatabaseURL:           "postgres://localhost/db",
		RedisURL:              "redis://localhost:6379/0",
		AllowedOrigin:         "https://example.com",
		ReadTimeout:           10,
		WriteTimeout:          10,
		IdleTimeout:           10,
		AccessTokenSecret:     "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:    "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:            "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:    "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:    30 * time.Second,
		RoomHeartbeatInterval: 10 * time.Second,
		RoomPresenceTTL:       30 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing production metrics token")
	}

	cfg.MetricsAuthToken = "secret-token"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config with metrics token, got %v", err)
	}
}

func TestValidateRoomPresenceTTL(t *testing.T) {
	cfg := config.Config{
		AppEnv:                "test",
		Version:               "0.0.0",
		HTTPAddr:              ":8080",
		DatabaseURL:           "postgres://localhost/db",
		RedisURL:              "redis://localhost:6379/0",
		AllowedOrigin:         "http://localhost:3000",
		ReadTimeout:           10,
		WriteTimeout:          10,
		IdleTimeout:           10,
		AccessTokenSecret:     "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:    "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:            "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:    "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:    30 * time.Second,
		RoomHeartbeatInterval: 10 * time.Second,
		RoomPresenceTTL:       10 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for room presence TTL not greater than heartbeat interval")
	}
}
