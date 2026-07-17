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
	if cfg.RefreshTokenTTL != 30*24*time.Hour {
		t.Errorf("RefreshTokenTTL default = %v, want 30 days", cfg.RefreshTokenTTL)
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

func TestLoadMatchmakingDefaults(t *testing.T) {
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
	if cfg.MatchmakingQueueLease != 30*time.Second {
		t.Errorf("MatchmakingQueueLease = %v, want 30s", cfg.MatchmakingQueueLease)
	}
	if cfg.MatchmakingClaimTTL != 15*time.Second {
		t.Errorf("MatchmakingClaimTTL = %v, want 15s", cfg.MatchmakingClaimTTL)
	}
	if cfg.MatchmakingStartDelay != 5*time.Second {
		t.Errorf("MatchmakingStartDelay = %v, want 5s", cfg.MatchmakingStartDelay)
	}
	if cfg.MatchmakingRoundCount != 5 {
		t.Errorf("MatchmakingRoundCount = %d, want 5", cfg.MatchmakingRoundCount)
	}
	if cfg.MatchmakingTimerSeconds != 60 {
		t.Errorf("MatchmakingTimerSeconds = %d, want 60", cfg.MatchmakingTimerSeconds)
	}
	if cfg.MatchmakingCandidateScanLimit != 20 {
		t.Errorf("MatchmakingCandidateScanLimit = %d, want 20", cfg.MatchmakingCandidateScanLimit)
	}
}

func TestLoadMatchmakingOverridesAndMapID(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")
	t.Setenv("MATCHMAKING_DEFAULT_MAP_ID", "00000000-0000-0000-0000-000000000099")
	t.Setenv("MATCHMAKING_QUEUE_LEASE_SECONDS", "45")
	t.Setenv("MATCHMAKING_CLAIM_TTL_SECONDS", "20")
	t.Setenv("MATCHMAKING_START_DELAY_SECONDS", "8")
	t.Setenv("MATCHMAKING_ROUND_COUNT", "3")
	t.Setenv("MATCHMAKING_TIMER_SECONDS", "90")
	t.Setenv("MATCHMAKING_CANDIDATE_SCAN_LIMIT", "10")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.MatchmakingDefaultMapID != "00000000-0000-0000-0000-000000000099" {
		t.Errorf("MatchmakingDefaultMapID = %q", cfg.MatchmakingDefaultMapID)
	}
	if cfg.MatchmakingQueueLease != 45*time.Second {
		t.Errorf("MatchmakingQueueLease = %v", cfg.MatchmakingQueueLease)
	}
	if cfg.MatchmakingClaimTTL != 20*time.Second {
		t.Errorf("MatchmakingClaimTTL = %v", cfg.MatchmakingClaimTTL)
	}
	if cfg.MatchmakingStartDelay != 8*time.Second {
		t.Errorf("MatchmakingStartDelay = %v", cfg.MatchmakingStartDelay)
	}
	if cfg.MatchmakingRoundCount != 3 {
		t.Errorf("MatchmakingRoundCount = %d", cfg.MatchmakingRoundCount)
	}
	if cfg.MatchmakingTimerSeconds != 90 {
		t.Errorf("MatchmakingTimerSeconds = %d", cfg.MatchmakingTimerSeconds)
	}
	if cfg.MatchmakingCandidateScanLimit != 10 {
		t.Errorf("MatchmakingCandidateScanLimit = %d", cfg.MatchmakingCandidateScanLimit)
	}
}

func TestValidateMatchmakingBounds(t *testing.T) {
	base := config.Config{
		AppEnv:                        "test",
		Version:                       "0.0.0",
		HTTPAddr:                      ":8080",
		DatabaseURL:                   "postgres://localhost/db",
		RedisURL:                      "redis://localhost:6379/0",
		AllowedOrigin:                 "http://localhost:3000",
		ReadTimeout:                   time.Second,
		WriteTimeout:                  time.Second,
		IdleTimeout:                   time.Second,
		AccessTokenSecret:             "test-access-token-secret-at-least-32-bytes-long",
		RefreshTokenSecret:            "test-refresh-token-secret-at-least-32-bytes-long",
		CSRFSecret:                    "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret:            "test-guest-secret-at-least-32-bytes-long",
		RoomReconnectGrace:            30 * time.Second,
		RoomHeartbeatInterval:         10 * time.Second,
		RoomPresenceTTL:               30 * time.Second,
		MatchmakingQueueLease:         30 * time.Second,
		MatchmakingClaimTTL:           15 * time.Second,
		MatchmakingStartDelay:         5 * time.Second,
		MatchmakingRoundCount:         5,
		MatchmakingTimerSeconds:       60,
		MatchmakingCandidateScanLimit: 20,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("base valid config rejected: %v", err)
	}

	invalid := base
	invalid.MatchmakingCandidateScanLimit = 1
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected scan limit validation error")
	}

	invalid = base
	invalid.MatchmakingRoundCount = 0
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected round count validation error")
	}

	invalid = base
	invalid.MatchmakingDefaultMapID = "not-a-uuid"
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected map id validation error")
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
	cfg.MatchmakingQueueLease = 30 * time.Second
	cfg.MatchmakingClaimTTL = 15 * time.Second
	cfg.MatchmakingStartDelay = 5 * time.Second
	cfg.MatchmakingRoundCount = 5
	cfg.MatchmakingTimerSeconds = 60
	cfg.MatchmakingCandidateScanLimit = 20
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
