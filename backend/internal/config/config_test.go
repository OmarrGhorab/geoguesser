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
	if cfg.QuickPlayRoundCount != 5 {
		t.Errorf("QuickPlayRoundCount = %d, want 5", cfg.QuickPlayRoundCount)
	}
	if cfg.QuickPlayTimerSeconds != 60 {
		t.Errorf("QuickPlayTimerSeconds = %d, want 60", cfg.QuickPlayTimerSeconds)
	}
}

func TestQuickPlayMapFallbackToChallenge(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")
	t.Setenv("QUICK_PLAY_DEFAULT_MAP_ID", "")
	t.Setenv("CHALLENGE_DEFAULT_MAP_ID", "00000000-0000-0000-0000-000000000042")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.QuickPlayDefaultMapID != "00000000-0000-0000-0000-000000000042" {
		t.Fatalf("QuickPlayDefaultMapID = %q, want challenge fallback", cfg.QuickPlayDefaultMapID)
	}
}

func TestQuickPlayMapFallbackChainToMatchmaking(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")
	t.Setenv("QUICK_PLAY_DEFAULT_MAP_ID", "")
	t.Setenv("CHALLENGE_DEFAULT_MAP_ID", "")
	t.Setenv("MATCHMAKING_DEFAULT_MAP_ID", "00000000-0000-0000-0000-000000000043")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.QuickPlayDefaultMapID != "00000000-0000-0000-0000-000000000043" {
		t.Fatalf("QuickPlayDefaultMapID = %q, want matchmaking fallback", cfg.QuickPlayDefaultMapID)
	}

	// Explicit quick play value must win over both fallbacks.
	t.Setenv("QUICK_PLAY_DEFAULT_MAP_ID", "00000000-0000-0000-0000-000000000044")
	t.Setenv("CHALLENGE_DEFAULT_MAP_ID", "00000000-0000-0000-0000-000000000042")
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.QuickPlayDefaultMapID != "00000000-0000-0000-0000-000000000044" {
		t.Fatalf("QuickPlayDefaultMapID = %q, want explicit value", cfg.QuickPlayDefaultMapID)
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
	base := validBaseConfig()
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
	cfg := validBaseConfig()
	cfg.AppEnv = "production"
	cfg.AllowedOrigin = "https://example.com"
	cfg.MetricsAuthToken = ""

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

func validBaseConfig() config.Config {
	return config.Config{
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
		QuickPlayRoundCount:           5,
		QuickPlayTimerSeconds:         60,
		// Casual / Ranked team modes — safe disabled defaults.
		CasualMatchmakingEnabled:      false,
		RankedTeamModesEnabled:        false,
		TeamChatImagesEnabled:         false,
		PartyInviteTTL:                900 * time.Second,
		MatchReconnectGrace:           90 * time.Second,
		CasualInactivity:              600 * time.Second,
		MatchSweepInterval:            5 * time.Second,
		MatchClaimSweepInterval:       5 * time.Second,
		CompetitiveSeasonDurationDays: 84,
		CompetitiveInitialRating:      800,
		CompetitiveEloK:               32,
		CompetitiveResetFactorBPS:     5000,
		CompetitiveAbandonPenalty:     15,
		CompetitiveTop500MinMatches:   25,
		RealtimeTicketTTL:             30 * time.Second,
		RealtimeAllowedOrigins:        []string{"http://localhost:3000"},
		RealtimeOutboundQueueSize:     128,
		TeamChatRetentionDays:         30,
		TeamChatReportRetentionDays:   180,
		TeamChatImageMaxBytes:         5 * 1024 * 1024,
		TeamChatImageMaxPixels:        20_000_000,
		TeamChatImageMaxDimension:     2048,
		TeamChatCleanupInterval:       900 * time.Second,
	}
}

func TestLoadCasualRankedDefaults(t *testing.T) {
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

	// Feature flags default off for safe rollout.
	if cfg.CasualMatchmakingEnabled {
		t.Error("CasualMatchmakingEnabled default = true, want false")
	}
	if cfg.RankedTeamModesEnabled {
		t.Error("RankedTeamModesEnabled default = true, want false")
	}
	if cfg.TeamChatImagesEnabled {
		t.Error("TeamChatImagesEnabled default = true, want false")
	}

	// Party / match timing.
	if cfg.PartyInviteTTL != 900*time.Second {
		t.Errorf("PartyInviteTTL = %v, want 900s", cfg.PartyInviteTTL)
	}
	if cfg.MatchReconnectGrace != 90*time.Second {
		t.Errorf("MatchReconnectGrace = %v, want 90s", cfg.MatchReconnectGrace)
	}
	if cfg.CasualInactivity != 600*time.Second {
		t.Errorf("CasualInactivity = %v, want 600s", cfg.CasualInactivity)
	}
	if cfg.MatchSweepInterval != 5*time.Second {
		t.Errorf("MatchSweepInterval = %v, want 5s", cfg.MatchSweepInterval)
	}
	if cfg.MatchClaimSweepInterval != 5*time.Second {
		t.Errorf("MatchClaimSweepInterval = %v, want 5s", cfg.MatchClaimSweepInterval)
	}

	// Competitive constants.
	if cfg.CompetitiveSeasonDurationDays != 84 {
		t.Errorf("CompetitiveSeasonDurationDays = %d, want 84", cfg.CompetitiveSeasonDurationDays)
	}
	if cfg.CompetitiveInitialRating != 800 {
		t.Errorf("CompetitiveInitialRating = %d, want 800", cfg.CompetitiveInitialRating)
	}
	if cfg.CompetitiveEloK != 32 {
		t.Errorf("CompetitiveEloK = %d, want 32", cfg.CompetitiveEloK)
	}
	if cfg.CompetitiveResetFactorBPS != 5000 {
		t.Errorf("CompetitiveResetFactorBPS = %d, want 5000", cfg.CompetitiveResetFactorBPS)
	}
	if cfg.CompetitiveAbandonPenalty != 15 {
		t.Errorf("CompetitiveAbandonPenalty = %d, want 15", cfg.CompetitiveAbandonPenalty)
	}
	if cfg.CompetitiveTop500MinMatches != 25 {
		t.Errorf("CompetitiveTop500MinMatches = %d, want 25", cfg.CompetitiveTop500MinMatches)
	}

	// Realtime limits.
	if cfg.RealtimeTicketTTL != 30*time.Second {
		t.Errorf("RealtimeTicketTTL = %v, want 30s", cfg.RealtimeTicketTTL)
	}
	if len(cfg.RealtimeAllowedOrigins) != 1 || cfg.RealtimeAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("RealtimeAllowedOrigins = %v, want [http://localhost:3000]", cfg.RealtimeAllowedOrigins)
	}
	if cfg.RealtimeOutboundQueueSize != 128 {
		t.Errorf("RealtimeOutboundQueueSize = %d, want 128", cfg.RealtimeOutboundQueueSize)
	}

	// Chat retention / image limits.
	if cfg.TeamChatRetentionDays != 30 {
		t.Errorf("TeamChatRetentionDays = %d, want 30", cfg.TeamChatRetentionDays)
	}
	if cfg.TeamChatReportRetentionDays != 180 {
		t.Errorf("TeamChatReportRetentionDays = %d, want 180", cfg.TeamChatReportRetentionDays)
	}
	if cfg.TeamChatImageMaxBytes != 5*1024*1024 {
		t.Errorf("TeamChatImageMaxBytes = %d, want 5242880", cfg.TeamChatImageMaxBytes)
	}
	if cfg.TeamChatImageMaxPixels != 20_000_000 {
		t.Errorf("TeamChatImageMaxPixels = %d, want 20000000", cfg.TeamChatImageMaxPixels)
	}
	if cfg.TeamChatImageMaxDimension != 2048 {
		t.Errorf("TeamChatImageMaxDimension = %d, want 2048", cfg.TeamChatImageMaxDimension)
	}
	if cfg.TeamChatCleanupInterval != 900*time.Second {
		t.Errorf("TeamChatCleanupInterval = %v, want 900s", cfg.TeamChatCleanupInterval)
	}
}

func TestLoadCasualRankedOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("VERSION", "0.0.0")
	t.Setenv("ACCESS_TOKEN_SECRET", "test-access-token-secret-at-least-32-bytes-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-token-secret-at-least-32-bytes-long")
	t.Setenv("CSRF_SECRET", "test-csrf-secret-at-least-32-bytes-long")
	t.Setenv("GUEST_SESSION_SECRET", "test-guest-secret-at-least-32-bytes-long")
	t.Setenv("CASUAL_MATCHMAKING_ENABLED", "true")
	t.Setenv("RANKED_TEAM_MODES_ENABLED", "true")
	t.Setenv("TEAM_CHAT_IMAGES_ENABLED", "true")
	t.Setenv("PARTY_INVITE_TTL_SECONDS", "600")
	t.Setenv("MATCH_RECONNECT_GRACE_SECONDS", "120")
	t.Setenv("CASUAL_INACTIVITY_SECONDS", "300")
	t.Setenv("MATCH_SWEEP_INTERVAL_SECONDS", "10")
	t.Setenv("MATCH_CLAIM_SWEEP_INTERVAL_SECONDS", "8")
	t.Setenv("COMPETITIVE_SEASON_DURATION_DAYS", "42")
	t.Setenv("COMPETITIVE_INITIAL_RATING", "1000")
	t.Setenv("COMPETITIVE_ELO_K", "24")
	t.Setenv("COMPETITIVE_RESET_FACTOR_BPS", "2500")
	t.Setenv("COMPETITIVE_ABANDON_PENALTY", "20")
	t.Setenv("COMPETITIVE_TOP500_MIN_MATCHES", "10")
	t.Setenv("REALTIME_TICKET_TTL_SECONDS", "45")
	t.Setenv("REALTIME_ALLOWED_ORIGINS", "http://localhost:3000,https://play.example.com")
	t.Setenv("REALTIME_OUTBOUND_QUEUE_SIZE", "64")
	t.Setenv("TEAM_CHAT_RETENTION_DAYS", "14")
	t.Setenv("TEAM_CHAT_REPORT_RETENTION_DAYS", "90")
	t.Setenv("TEAM_CHAT_IMAGE_MAX_BYTES", "1048576")
	t.Setenv("TEAM_CHAT_IMAGE_MAX_PIXELS", "10000000")
	t.Setenv("TEAM_CHAT_IMAGE_MAX_DIMENSION", "1024")
	t.Setenv("TEAM_CHAT_CLEANUP_INTERVAL_SECONDS", "300")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if !cfg.CasualMatchmakingEnabled {
		t.Error("CasualMatchmakingEnabled = false, want true")
	}
	if !cfg.RankedTeamModesEnabled {
		t.Error("RankedTeamModesEnabled = false, want true")
	}
	if !cfg.TeamChatImagesEnabled {
		t.Error("TeamChatImagesEnabled = false, want true")
	}
	if cfg.PartyInviteTTL != 600*time.Second {
		t.Errorf("PartyInviteTTL = %v, want 600s", cfg.PartyInviteTTL)
	}
	if cfg.MatchReconnectGrace != 120*time.Second {
		t.Errorf("MatchReconnectGrace = %v, want 120s", cfg.MatchReconnectGrace)
	}
	if cfg.CasualInactivity != 300*time.Second {
		t.Errorf("CasualInactivity = %v, want 300s", cfg.CasualInactivity)
	}
	if cfg.MatchSweepInterval != 10*time.Second {
		t.Errorf("MatchSweepInterval = %v, want 10s", cfg.MatchSweepInterval)
	}
	if cfg.MatchClaimSweepInterval != 8*time.Second {
		t.Errorf("MatchClaimSweepInterval = %v, want 8s", cfg.MatchClaimSweepInterval)
	}
	if cfg.CompetitiveSeasonDurationDays != 42 {
		t.Errorf("CompetitiveSeasonDurationDays = %d, want 42", cfg.CompetitiveSeasonDurationDays)
	}
	if cfg.CompetitiveInitialRating != 1000 {
		t.Errorf("CompetitiveInitialRating = %d, want 1000", cfg.CompetitiveInitialRating)
	}
	if cfg.CompetitiveEloK != 24 {
		t.Errorf("CompetitiveEloK = %d, want 24", cfg.CompetitiveEloK)
	}
	if cfg.CompetitiveResetFactorBPS != 2500 {
		t.Errorf("CompetitiveResetFactorBPS = %d, want 2500", cfg.CompetitiveResetFactorBPS)
	}
	if cfg.CompetitiveAbandonPenalty != 20 {
		t.Errorf("CompetitiveAbandonPenalty = %d, want 20", cfg.CompetitiveAbandonPenalty)
	}
	if cfg.CompetitiveTop500MinMatches != 10 {
		t.Errorf("CompetitiveTop500MinMatches = %d, want 10", cfg.CompetitiveTop500MinMatches)
	}
	if cfg.RealtimeTicketTTL != 45*time.Second {
		t.Errorf("RealtimeTicketTTL = %v, want 45s", cfg.RealtimeTicketTTL)
	}
	if len(cfg.RealtimeAllowedOrigins) != 2 {
		t.Fatalf("RealtimeAllowedOrigins len = %d, want 2", len(cfg.RealtimeAllowedOrigins))
	}
	if cfg.RealtimeAllowedOrigins[0] != "http://localhost:3000" || cfg.RealtimeAllowedOrigins[1] != "https://play.example.com" {
		t.Errorf("RealtimeAllowedOrigins = %v", cfg.RealtimeAllowedOrigins)
	}
	if cfg.RealtimeOutboundQueueSize != 64 {
		t.Errorf("RealtimeOutboundQueueSize = %d, want 64", cfg.RealtimeOutboundQueueSize)
	}
	if cfg.TeamChatRetentionDays != 14 {
		t.Errorf("TeamChatRetentionDays = %d, want 14", cfg.TeamChatRetentionDays)
	}
	if cfg.TeamChatReportRetentionDays != 90 {
		t.Errorf("TeamChatReportRetentionDays = %d, want 90", cfg.TeamChatReportRetentionDays)
	}
	if cfg.TeamChatImageMaxBytes != 1_048_576 {
		t.Errorf("TeamChatImageMaxBytes = %d, want 1048576", cfg.TeamChatImageMaxBytes)
	}
	if cfg.TeamChatImageMaxPixels != 10_000_000 {
		t.Errorf("TeamChatImageMaxPixels = %d, want 10000000", cfg.TeamChatImageMaxPixels)
	}
	if cfg.TeamChatImageMaxDimension != 1024 {
		t.Errorf("TeamChatImageMaxDimension = %d, want 1024", cfg.TeamChatImageMaxDimension)
	}
	if cfg.TeamChatCleanupInterval != 300*time.Second {
		t.Errorf("TeamChatCleanupInterval = %v, want 300s", cfg.TeamChatCleanupInterval)
	}
}

func TestValidateCasualRankedBounds(t *testing.T) {
	base := validBaseConfig()
	if err := base.Validate(); err != nil {
		t.Fatalf("base valid config rejected: %v", err)
	}

	cases := []struct {
		name string
		mut  func(*config.Config)
		want string
	}{
		{
			name: "party invite ttl",
			mut:  func(c *config.Config) { c.PartyInviteTTL = 0 },
		},
		{
			name: "match reconnect grace",
			mut:  func(c *config.Config) { c.MatchReconnectGrace = 0 },
		},
		{
			name: "casual inactivity",
			mut:  func(c *config.Config) { c.CasualInactivity = 0 },
		},
		{
			name: "match sweep interval",
			mut:  func(c *config.Config) { c.MatchSweepInterval = 0 },
		},
		{
			name: "match claim sweep interval",
			mut:  func(c *config.Config) { c.MatchClaimSweepInterval = 0 },
		},
		{
			name: "season duration days",
			mut:  func(c *config.Config) { c.CompetitiveSeasonDurationDays = 0 },
		},
		{
			name: "initial rating negative",
			mut:  func(c *config.Config) { c.CompetitiveInitialRating = -1 },
		},
		{
			name: "elo k",
			mut:  func(c *config.Config) { c.CompetitiveEloK = 0 },
		},
		{
			name: "reset factor bps low",
			mut:  func(c *config.Config) { c.CompetitiveResetFactorBPS = -1 },
		},
		{
			name: "reset factor bps high",
			mut:  func(c *config.Config) { c.CompetitiveResetFactorBPS = 10001 },
		},
		{
			name: "abandon penalty negative",
			mut:  func(c *config.Config) { c.CompetitiveAbandonPenalty = -1 },
		},
		{
			name: "top500 min matches",
			mut:  func(c *config.Config) { c.CompetitiveTop500MinMatches = 0 },
		},
		{
			name: "realtime ticket ttl",
			mut:  func(c *config.Config) { c.RealtimeTicketTTL = 0 },
		},
		{
			name: "realtime outbound queue size",
			mut:  func(c *config.Config) { c.RealtimeOutboundQueueSize = 0 },
		},
		{
			name: "team chat retention",
			mut:  func(c *config.Config) { c.TeamChatRetentionDays = 0 },
		},
		{
			name: "team chat report retention less than normal",
			mut:  func(c *config.Config) { c.TeamChatReportRetentionDays = 10 },
		},
		{
			name: "image max bytes",
			mut:  func(c *config.Config) { c.TeamChatImageMaxBytes = 0 },
		},
		{
			name: "image max pixels",
			mut:  func(c *config.Config) { c.TeamChatImageMaxPixels = 0 },
		},
		{
			name: "image max dimension",
			mut:  func(c *config.Config) { c.TeamChatImageMaxDimension = 0 },
		},
		{
			name: "cleanup interval",
			mut:  func(c *config.Config) { c.TeamChatCleanupInterval = 0 },
		},
		{
			name: "empty realtime origins",
			mut:  func(c *config.Config) { c.RealtimeAllowedOrigins = nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mut(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}
