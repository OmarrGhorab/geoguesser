package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/platform/email"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
	"github.com/redis/go-redis/v9"
)

func setupAuthHandler(t *testing.T) (*auth.Handler, *redis.Client) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	if databaseURL == "" || redisURL == "" {
		t.Skip("DATABASE_URL and REDIS_URL required for integration tests")
	}

	cfg := config.Config{
		AppEnv:             "test",
		Version:            "0.0.0",
		HTTPAddr:           ":8080",
		DatabaseURL:        databaseURL,
		RedisURL:           redisURL,
		AllowedOrigin:      "http://localhost:3000",
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        60 * time.Second,
		AccessTokenSecret:  "test-access-token-secret-at-least-32-bytes-long",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenSecret: "test-refresh-token-secret-at-least-32-bytes-long",
		RefreshTokenTTL:    7 * 24 * time.Hour,
		CSRFSecret:         "test-csrf-secret-at-least-32-bytes-long",
		GuestSessionSecret: "test-guest-secret-at-least-32-bytes-long",
	}

	obs, err := observability.New("geoguess-test", cfg.Version)
	if err != nil {
		t.Fatalf("observability setup failed: %v", err)
	}

	ctx := context.Background()
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("postgres connection failed: %v", err)
	}
	redisClient, err := redisplatform.Open(ctx, redisURL)
	if err != nil {
		t.Fatalf("redis connection failed: %v", err)
	}

	repo := auth.NewRepository(db)
	hasher := auth.NewBCryptHasherWithCost(4)
	tokenManager, err := auth.NewTokenManager(cfg.AccessTokenSecret, cfg.AccessTokenTTL)
	if err != nil {
		t.Fatalf("token manager setup failed: %v", err)
	}
	guestManager, err := auth.NewGuestSessionManager(cfg.GuestSessionSecret)
	if err != nil {
		t.Fatalf("guest manager setup failed: %v", err)
	}
	csrfManager, err := auth.NewCSRFManager(cfg.CSRFSecret)
	if err != nil {
		t.Fatalf("csrf manager setup failed: %v", err)
	}
	oauthManager := auth.NewOAuthManager(cfg)

	otpStore := auth.NewOTPStore(redisClient, cfg.OTPTTL)
	sessionStore := auth.NewRedisSessionStore(redisClient)
	emailSender := email.NewLoggerSender(obs.Logger)
	service := auth.NewService(repo, hasher, tokenManager, guestManager, csrfManager, oauthManager, sessionStore, otpStore, emailSender, redisClient, cfg, clock.NewSystem())
	handler := auth.NewHandler(service, cfg, obs.Logger)
	return handler, redisClient
}

func TestRegisterAndLoginIntegration(t *testing.T) {
	handler, redisClient := setupAuthHandler(t)
	defer func() { _ = redisClient.Close() }()

	registerBody := `{"email":"test-register@example.com","password":"correct horse battery staple","display_name":"Tester"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(registerBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "invalid")
	handler.Register(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without csrf, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/me", nil)
	handler.Me(w, r)
	csrfCookie := w.Header().Values("Set-Cookie")
	var csrfToken string
	for _, c := range csrfCookie {
		if strings.HasPrefix(c, "csrf_token=") {
			csrfToken = strings.TrimPrefix(strings.SplitN(c, ";", 2)[0], "csrf_token=")
			break
		}
	}
	if csrfToken == "" {
		t.Fatal("expected csrf_token cookie from /me")
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(registerBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", csrfToken)
	handler.Register(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
