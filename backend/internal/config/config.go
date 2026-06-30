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
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		Version:             getEnv("VERSION", "0.1.0"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://geoguess:geoguess@localhost:5432/geoguess?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AllowedOrigin:       getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		MetricsAuthToken:    strings.TrimSpace(os.Getenv("METRICS_AUTH_TOKEN")),
		ReadTimeout:         durationSeconds("HTTP_READ_TIMEOUT_SECONDS", 10),
		WriteTimeout:        durationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 15),
		IdleTimeout:         durationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60),
		AccessTokenSecret:   strings.TrimSpace(os.Getenv("ACCESS_TOKEN_SECRET")),
		AccessTokenTTL:      durationSeconds("ACCESS_TOKEN_TTL_SECONDS", 15*60),
		RefreshTokenTTL:     durationSeconds("REFRESH_TOKEN_TTL_SECONDS", 7*24*60*60),
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

	return nil
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

func (c Config) String() string {
	return fmt.Sprintf("env=%s version=%s addr=%s", c.AppEnv, c.Version, c.HTTPAddr)
}
