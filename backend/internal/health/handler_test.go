package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raven/geoguess/backend/internal/health"
)

type fakePinger struct {
	err error
}

func (f *fakePinger) Ping(_ context.Context) error {
	return f.err
}

func setupTestHandler(t *testing.T, pingers map[string]health.Pinger) *health.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return health.NewHandlerWithPingers("0.1.0-test", logger, pingers)
}

func TestHealth(t *testing.T) {
	h := setupTestHandler(t, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.Health(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body health.HealthResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != "0.1.0-test" {
		t.Errorf("version = %q, want 0.1.0-test", body.Version)
	}
}

func TestReadyAllHealthy(t *testing.T) {
	h := setupTestHandler(t, map[string]health.Pinger{
		"postgres": &fakePinger{},
		"redis":    &fakePinger{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/ready", nil)
	h.Ready(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body health.ReadinessResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode ready response: %v", err)
	}
	if body.Status != "ready" {
		t.Errorf("status = %q, want ready", body.Status)
	}
	if body.Checks["postgres"] != "ok" {
		t.Errorf("postgres check = %q, want ok", body.Checks["postgres"])
	}
	if body.Checks["redis"] != "ok" {
		t.Errorf("redis check = %q, want ok", body.Checks["redis"])
	}
}

func TestReadyDependencyFailure(t *testing.T) {
	h := setupTestHandler(t, map[string]health.Pinger{
		"postgres": &fakePinger{err: errors.New("connection refused")},
		"redis":    &fakePinger{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/ready", nil)
	h.Ready(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusServiceUnavailable)
	}

	var body health.ReadinessResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode ready response: %v", err)
	}
	if body.Status != "not_ready" {
		t.Errorf("status = %q, want not_ready", body.Status)
	}
	if body.Checks["postgres"] != "error" {
		t.Errorf("postgres check = %q, want error", body.Checks["postgres"])
	}
}
