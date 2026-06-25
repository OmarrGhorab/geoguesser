package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/app"
	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/users"
)

// noopRateLimiter is a test stub that always allows requests.
type noopRateLimiter struct{}

func (noopRateLimiter) Allow(context.Context, string, int, time.Duration) (bool, int, error) {
	return true, 0, nil
}

func TestRouterMountsHealthEndpoints(t *testing.T) {
	cfg := testConfig()

	obs, err := observability.New("geoguess-test", cfg.Version)
	if err != nil {
		t.Fatalf("observability setup failed: %v", err)
	}

	healthHandler := health.NewHandlerWithPingers(cfg.Version, obs.Logger, nil)
	router := app.NewRouter(cfg, obs.Logger, obs, noopRateLimiter{}, healthHandler, nil, nil, nil)

	endpoints := []string{"/health", "/ready", "/metrics", "/api/v1/health", "/api/v1/ready", "/api/v1/metrics"}
	for _, path := range endpoints {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(w, r)

		if w.Code == http.StatusNotFound {
			t.Errorf("endpoint %s returned 404", path)
		}
	}
}

func TestRouterMountsDocumentedAuthAndUserRoutes(t *testing.T) {
	cfg := testConfig()

	obs, err := observability.New("geoguess-test", cfg.Version)
	if err != nil {
		t.Fatalf("observability setup failed: %v", err)
	}

	csrfManager, err := auth.NewCSRFManager(cfg.CSRFSecret)
	if err != nil {
		t.Fatalf("csrf manager setup failed: %v", err)
	}

	authService := auth.NewService(nil, nil, nil, nil, csrfManager, nil, nil, nil, nil, nil, cfg, clock.NewSystem())
	authHandler := auth.NewHandler(authService, cfg, obs.Logger)
	usersHandler := users.NewHandler(users.NewService(nil), obs.Logger)
	healthHandler := health.NewHandlerWithPingers(cfg.Version, obs.Logger, nil)
	router := app.NewRouter(cfg, obs.Logger, obs, noopRateLimiter{}, healthHandler, authHandler, usersHandler, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{}"))
	router.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected documented auth route to reject missing csrf with 403, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid/stats", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected documented user stats route to return handler 404, got %d: %s", w.Code, w.Body.String())
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected user stats handler JSON response, got content-type %q and body %q", contentType, w.Body.String())
	}

	token, err := csrfManager.Generate()
	if err != nil {
		t.Fatalf("csrf token generation failed: %v", err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{}"))
	r.Header.Set("X-CSRF-Token", token)
	r.AddCookie(&http.Cookie{Name: auth.CSRFTokenCookieName, Value: token})
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected documented auth route to reach handler validation with 400, got %d: %s", w.Code, w.Body.String())
	}
}

func testConfig() config.Config {
	return config.Config{
		AppEnv:             "test",
		Version:            "0.0.0",
		HTTPAddr:           ":8080",
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
}
