package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/app"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

func TestRouterMountsHealthEndpoints(t *testing.T) {
	cfg := config.Config{
		AppEnv:        "test",
		Version:       "0.0.0",
		HTTPAddr:      ":8080",
		AllowedOrigin: "http://localhost:3000",
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:   60 * time.Second,
	}

	obs, err := observability.New("geoguess-test", cfg.Version)
	if err != nil {
		t.Fatalf("observability setup failed: %v", err)
	}

	healthHandler := health.NewHandlerWithPingers(cfg.Version, obs.Logger, nil)
	router := app.NewRouter(cfg, obs.Logger, obs, healthHandler)

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
