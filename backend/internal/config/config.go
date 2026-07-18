package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv              string
	Version             string
	HTTPAddr            string
	DatabaseURL         string
	RedisURL            string
	AllowedOrigin       string
	MetricsAuthToken    string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	AccessTokenSecret   string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	RefreshTokenSecret  string
	CSRFSecret          string
	GuestSessionSecret  string
	CookieDomain        string
	CookieSecure        bool
	CookieSameSite      string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURL  string
	OAuthStateTTL       time.Duration

	EmailProvider  string
	EmailFrom      string
	ResendAPIKey   string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPassword   string
	OTPTTL         time.Duration
	OTPMaxAttempts int
	OTPRateLimit   int

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2Endpoint        string
	R2PublicURL       string
	R2SignedURLTTL    time.Duration
	R2MaxFileSize     int64

	ChallengeResetHourUTC int
	ChallengeDefaultMapID string

	RoomReconnectGrace      time.Duration
	RoomHeartbeatInterval   time.Duration
	RoomPresenceTTL         time.Duration
	RoomRealtimeAllowedHost string

	// Ranked matchmaking (Phase 9).
	MatchmakingDefaultMapID       string
	MatchmakingQueueLease         time.Duration
	MatchmakingClaimTTL           time.Duration
	MatchmakingStartDelay         time.Duration
	MatchmakingRoundCount         int
	MatchmakingTimerSeconds       int
	MatchmakingCandidateScanLimit int

	// Casual / Ranked team modes (012). Feature flags default false for safe rollout.
	CasualMatchmakingEnabled bool
	RankedTeamModesEnabled   bool
	TeamChatImagesEnabled    bool

	// Party / match timing.
	PartyInviteTTL          time.Duration
	MatchReconnectGrace     time.Duration
	CasualInactivity        time.Duration
	MatchSweepInterval      time.Duration
	MatchClaimSweepInterval time.Duration

	// Competitive constants.
	CompetitiveSeasonDurationDays int
	CompetitiveInitialRating      int
	CompetitiveEloK               int
	CompetitiveResetFactorBPS     int
	CompetitiveAbandonPenalty     int
	CompetitiveTop500MinMatches   int

	// Realtime limits.
	RealtimeTicketTTL         time.Duration
	RealtimeAllowedOrigins    []string
	RealtimeOutboundQueueSize int

	// Team chat retention and image limits.
	TeamChatRetentionDays       int
	TeamChatReportRetentionDays int
	TeamChatImageMaxBytes       int64
	TeamChatImageMaxPixels      int
	TeamChatImageMaxDimension   int
	TeamChatCleanupInterval     time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Version:           getEnv("VERSION", "0.1.0"),
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://geoguess:geoguess@localhost:5432/geoguess?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		MetricsAuthToken:  strings.TrimSpace(os.Getenv("METRICS_AUTH_TOKEN")),
		ReadTimeout:       durationSeconds("HTTP_READ_TIMEOUT_SECONDS", 10),
		WriteTimeout:      durationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 15),
		IdleTimeout:       durationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60),
		AccessTokenSecret: strings.TrimSpace(os.Getenv("ACCESS_TOKEN_SECRET")),
		AccessTokenTTL:    durationSeconds("ACCESS_TOKEN_TTL_SECONDS", 15*60),
		// Keep the rotating refresh session for 30 days. Access tokens remain
		// short-lived; the BFF refreshes them transparently, so a normal login
		// remains valid for the full remembered-session period.
		RefreshTokenTTL:     durationSeconds("REFRESH_TOKEN_TTL_SECONDS", 30*24*60*60),
		RefreshTokenSecret:  strings.TrimSpace(os.Getenv("REFRESH_TOKEN_SECRET")),
		CSRFSecret:          strings.TrimSpace(os.Getenv("CSRF_SECRET")),
		GuestSessionSecret:  strings.TrimSpace(os.Getenv("GUEST_SESSION_SECRET")),
		CookieDomain:        strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		CookieSecure:        strings.EqualFold(getEnv("COOKIE_SECURE", "false"), "true"),
		CookieSameSite:      getEnv("COOKIE_SAME_SITE", "lax"),
		GoogleClientID:      strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:  strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:   strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URL")),
		DiscordClientID:     strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")),
		DiscordClientSecret: strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET")),
		DiscordRedirectURL:  strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URL")),
		OAuthStateTTL:       durationSeconds("OAUTH_STATE_TTL_SECONDS", 10*60),

		EmailProvider:  strings.TrimSpace(getEnv("EMAIL_PROVIDER", "logger")),
		EmailFrom:      strings.TrimSpace(getEnv("EMAIL_FROM", "noreply@geoguess.local")),
		ResendAPIKey:   strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		SMTPHost:       strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:       intEnv("SMTP_PORT", 587),
		SMTPUser:       strings.TrimSpace(os.Getenv("SMTP_USER")),
		SMTPPassword:   strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		OTPTTL:         durationSeconds("OTP_TTL_SECONDS", 10*60),
		OTPMaxAttempts: intEnv("OTP_MAX_ATTEMPTS", 3),
		OTPRateLimit:   intEnv("OTP_RATE_LIMIT", 3),

		R2AccountID:       strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
		R2AccessKeyID:     strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		R2SecretAccessKey: strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		R2Bucket:          strings.TrimSpace(os.Getenv("R2_BUCKET")),
		R2Endpoint:        strings.TrimSpace(os.Getenv("R2_ENDPOINT")),
		R2PublicURL:       strings.TrimSpace(os.Getenv("R2_PUBLIC_URL")),
		R2SignedURLTTL:    durationSeconds("R2_SIGNED_URL_TTL_SECONDS", 15*60),
		R2MaxFileSize:     int64Env("R2_MAX_FILE_SIZE_BYTES", 10*1024*1024),

		ChallengeResetHourUTC: intEnvAllowZero("CHALLENGE_RESET_HOUR_UTC", 0),
		ChallengeDefaultMapID: strings.TrimSpace(os.Getenv("CHALLENGE_DEFAULT_MAP_ID")),

		RoomReconnectGrace:      durationSeconds("ROOM_RECONNECT_GRACE_SECONDS", 30),
		RoomHeartbeatInterval:   durationSeconds("ROOM_HEARTBEAT_INTERVAL_SECONDS", 10),
		RoomPresenceTTL:         durationSeconds("ROOM_PRESENCE_TTL_SECONDS", 30),
		RoomRealtimeAllowedHost: strings.TrimSpace(os.Getenv("ROOM_REALTIME_ALLOWED_HOST")),

		MatchmakingDefaultMapID:       strings.TrimSpace(os.Getenv("MATCHMAKING_DEFAULT_MAP_ID")),
		MatchmakingQueueLease:         durationSeconds("MATCHMAKING_QUEUE_LEASE_SECONDS", 30),
		MatchmakingClaimTTL:           durationSeconds("MATCHMAKING_CLAIM_TTL_SECONDS", 15),
		MatchmakingStartDelay:         durationSeconds("MATCHMAKING_START_DELAY_SECONDS", 5),
		MatchmakingRoundCount:         intEnv("MATCHMAKING_ROUND_COUNT", 5),
		MatchmakingTimerSeconds:       intEnv("MATCHMAKING_TIMER_SECONDS", 60),
		MatchmakingCandidateScanLimit: intEnv("MATCHMAKING_CANDIDATE_SCAN_LIMIT", 20),

		// Casual / Ranked team modes — disabled by default until migration and rollout.
		CasualMatchmakingEnabled: boolEnv("CASUAL_MATCHMAKING_ENABLED", false),
		RankedTeamModesEnabled:   boolEnv("RANKED_TEAM_MODES_ENABLED", false),
		TeamChatImagesEnabled:    boolEnv("TEAM_CHAT_IMAGES_ENABLED", false),

		PartyInviteTTL:          durationSeconds("PARTY_INVITE_TTL_SECONDS", 900),
		MatchReconnectGrace:     durationSeconds("MATCH_RECONNECT_GRACE_SECONDS", 90),
		CasualInactivity:        durationSeconds("CASUAL_INACTIVITY_SECONDS", 600),
		MatchSweepInterval:      durationSeconds("MATCH_SWEEP_INTERVAL_SECONDS", 5),
		MatchClaimSweepInterval: durationSeconds("MATCH_CLAIM_SWEEP_INTERVAL_SECONDS", 5),

		CompetitiveSeasonDurationDays: intEnv("COMPETITIVE_SEASON_DURATION_DAYS", 84),
		CompetitiveInitialRating:      intEnvAllowZero("COMPETITIVE_INITIAL_RATING", 800),
		CompetitiveEloK:               intEnv("COMPETITIVE_ELO_K", 32),
		CompetitiveResetFactorBPS:     intEnvAllowZero("COMPETITIVE_RESET_FACTOR_BPS", 5000),
		CompetitiveAbandonPenalty:     intEnvAllowZero("COMPETITIVE_ABANDON_PENALTY", 15),
		CompetitiveTop500MinMatches:   intEnv("COMPETITIVE_TOP500_MIN_MATCHES", 25),

		RealtimeTicketTTL:         durationSeconds("REALTIME_TICKET_TTL_SECONDS", 30),
		RealtimeAllowedOrigins:    csvEnv("REALTIME_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		RealtimeOutboundQueueSize: intEnv("REALTIME_OUTBOUND_QUEUE_SIZE", 128),

		TeamChatRetentionDays:       intEnv("TEAM_CHAT_RETENTION_DAYS", 30),
		TeamChatReportRetentionDays: intEnv("TEAM_CHAT_REPORT_RETENTION_DAYS", 180),
		TeamChatImageMaxBytes:       int64Env("TEAM_CHAT_IMAGE_MAX_BYTES", 5*1024*1024),
		TeamChatImageMaxPixels:      intEnv("TEAM_CHAT_IMAGE_MAX_PIXELS", 20_000_000),
		TeamChatImageMaxDimension:   intEnv("TEAM_CHAT_IMAGE_MAX_DIMENSION", 2048),
		TeamChatCleanupInterval:     durationSeconds("TEAM_CHAT_CLEANUP_INTERVAL_SECONDS", 900),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AppEnv) == "" {
		return errors.New("APP_ENV is required")
	}
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("VERSION is required")
	}
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return errors.New("REDIS_URL is required")
	}
	if strings.TrimSpace(c.AllowedOrigin) == "" {
		return errors.New("ALLOWED_ORIGIN is required")
	}
	if strings.EqualFold(c.AppEnv, "production") && strings.TrimSpace(c.MetricsAuthToken) == "" {
		return errors.New("METRICS_AUTH_TOKEN is required in production")
	}
	if c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 {
		return errors.New("http timeouts must be positive")
	}
	if strings.TrimSpace(c.AccessTokenSecret) == "" {
		return errors.New("ACCESS_TOKEN_SECRET is required")
	}
	if strings.TrimSpace(c.RefreshTokenSecret) == "" {
		return errors.New("REFRESH_TOKEN_SECRET is required")
	}
	if strings.TrimSpace(c.CSRFSecret) == "" {
		return errors.New("CSRF_SECRET is required")
	}
	if strings.TrimSpace(c.GuestSessionSecret) == "" {
		return errors.New("GUEST_SESSION_SECRET is required")
	}
	if c.ChallengeResetHourUTC < 0 || c.ChallengeResetHourUTC > 23 {
		return errors.New("CHALLENGE_RESET_HOUR_UTC must be between 0 and 23")
	}
	if c.RoomReconnectGrace <= 0 {
		return errors.New("ROOM_RECONNECT_GRACE_SECONDS must be positive")
	}
	if c.RoomHeartbeatInterval <= 0 {
		return errors.New("ROOM_HEARTBEAT_INTERVAL_SECONDS must be positive")
	}
	if c.RoomPresenceTTL <= c.RoomHeartbeatInterval {
		return errors.New("ROOM_PRESENCE_TTL_SECONDS must be greater than ROOM_HEARTBEAT_INTERVAL_SECONDS")
	}
	if c.MatchmakingQueueLease <= 0 {
		return errors.New("MATCHMAKING_QUEUE_LEASE_SECONDS must be positive")
	}
	if c.MatchmakingClaimTTL <= 0 {
		return errors.New("MATCHMAKING_CLAIM_TTL_SECONDS must be positive")
	}
	if c.MatchmakingStartDelay <= 0 {
		return errors.New("MATCHMAKING_START_DELAY_SECONDS must be positive")
	}
	if c.MatchmakingRoundCount < 1 || c.MatchmakingRoundCount > 10 {
		return errors.New("MATCHMAKING_ROUND_COUNT must be between 1 and 10")
	}
	if c.MatchmakingTimerSeconds < 10 || c.MatchmakingTimerSeconds > 600 {
		return errors.New("MATCHMAKING_TIMER_SECONDS must be between 10 and 600")
	}
	if c.MatchmakingCandidateScanLimit < 2 || c.MatchmakingCandidateScanLimit > 100 {
		return errors.New("MATCHMAKING_CANDIDATE_SCAN_LIMIT must be between 2 and 100")
	}
	if mapID := strings.TrimSpace(c.MatchmakingDefaultMapID); mapID != "" {
		if _, err := parseUUID(mapID); err != nil {
			return errors.New("MATCHMAKING_DEFAULT_MAP_ID must be a valid UUID when set")
		}
	}

	// Casual / Ranked team mode bounds.
	if c.PartyInviteTTL <= 0 {
		return errors.New("PARTY_INVITE_TTL_SECONDS must be positive")
	}
	if c.MatchReconnectGrace <= 0 {
		return errors.New("MATCH_RECONNECT_GRACE_SECONDS must be positive")
	}
	if c.CasualInactivity <= 0 {
		return errors.New("CASUAL_INACTIVITY_SECONDS must be positive")
	}
	if c.MatchSweepInterval <= 0 {
		return errors.New("MATCH_SWEEP_INTERVAL_SECONDS must be positive")
	}
	if c.MatchClaimSweepInterval <= 0 {
		return errors.New("MATCH_CLAIM_SWEEP_INTERVAL_SECONDS must be positive")
	}
	if c.CompetitiveSeasonDurationDays < 1 || c.CompetitiveSeasonDurationDays > 365 {
		return errors.New("COMPETITIVE_SEASON_DURATION_DAYS must be between 1 and 365")
	}
	if c.CompetitiveInitialRating < 0 {
		return errors.New("COMPETITIVE_INITIAL_RATING must be non-negative")
	}
	if c.CompetitiveEloK < 1 || c.CompetitiveEloK > 128 {
		return errors.New("COMPETITIVE_ELO_K must be between 1 and 128")
	}
	if c.CompetitiveResetFactorBPS < 0 || c.CompetitiveResetFactorBPS > 10000 {
		return errors.New("COMPETITIVE_RESET_FACTOR_BPS must be between 0 and 10000")
	}
	if c.CompetitiveAbandonPenalty < 0 {
		return errors.New("COMPETITIVE_ABANDON_PENALTY must be non-negative")
	}
	if c.CompetitiveTop500MinMatches < 1 {
		return errors.New("COMPETITIVE_TOP500_MIN_MATCHES must be positive")
	}
	if c.RealtimeTicketTTL <= 0 {
		return errors.New("REALTIME_TICKET_TTL_SECONDS must be positive")
	}
	if c.RealtimeOutboundQueueSize < 1 || c.RealtimeOutboundQueueSize > 4096 {
		return errors.New("REALTIME_OUTBOUND_QUEUE_SIZE must be between 1 and 4096")
	}
	if len(c.RealtimeAllowedOrigins) == 0 {
		return errors.New("REALTIME_ALLOWED_ORIGINS must include at least one origin")
	}
	for _, origin := range c.RealtimeAllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			return errors.New("REALTIME_ALLOWED_ORIGINS entries must be non-empty")
		}
	}
	if c.TeamChatRetentionDays < 1 {
		return errors.New("TEAM_CHAT_RETENTION_DAYS must be positive")
	}
	if c.TeamChatReportRetentionDays < c.TeamChatRetentionDays {
		return errors.New("TEAM_CHAT_REPORT_RETENTION_DAYS must be >= TEAM_CHAT_RETENTION_DAYS")
	}
	if c.TeamChatImageMaxBytes < 1 {
		return errors.New("TEAM_CHAT_IMAGE_MAX_BYTES must be positive")
	}
	if c.TeamChatImageMaxPixels < 1 {
		return errors.New("TEAM_CHAT_IMAGE_MAX_PIXELS must be positive")
	}
	if c.TeamChatImageMaxDimension < 1 {
		return errors.New("TEAM_CHAT_IMAGE_MAX_DIMENSION must be positive")
	}
	if c.TeamChatCleanupInterval <= 0 {
		return errors.New("TEAM_CHAT_CLEANUP_INTERVAL_SECONDS must be positive")
	}

	return nil
}

func parseUUID(value string) ([16]byte, error) {
	// Lightweight UUID format check without adding a config-layer dependency.
	// Accepts standard 8-4-4-4-12 hex form.
	var zero [16]byte
	if len(value) != 36 {
		return zero, errors.New("invalid uuid length")
	}
	for i, ch := range value {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return zero, errors.New("invalid uuid separators")
			}
		default:
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
				return zero, errors.New("invalid uuid hex")
			}
		}
	}
	return zero, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func durationSeconds(key string, fallback int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallback) * time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(fallback) * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func intEnvAllowZero(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func int64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func csvEnv(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		out := make([]string, len(fallback))
		copy(out, fallback)
		return out
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		out := make([]string, len(fallback))
		copy(out, fallback)
		return out
	}
	return out
}

func (c Config) String() string {
	return fmt.Sprintf("env=%s version=%s addr=%s", c.AppEnv, c.Version, c.HTTPAddr)
}
