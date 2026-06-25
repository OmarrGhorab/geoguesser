package observability_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/platform/observability"
)

func TestNewMetrics(t *testing.T) {
	m, err := observability.NewMetrics("geoguess-test")
	if err != nil {
		t.Fatalf("NewMetrics error = %v", err)
	}

	if m.Registry() == nil {
		t.Error("Registry() returned nil")
	}

	// Observe and increment to confirm collectors are wired.
	m.HTTPRequestDuration.WithLabelValues("GET", "/health", "200").Observe(0.001)
	m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "200").Inc()
	m.PostgresErrorsTotal.Inc()
	m.RedisErrorsTotal.Inc()
}

func TestNewLogger(t *testing.T) {
	logger := observability.NewLogger(nil)
	if logger == nil {
		t.Error("NewLogger returned nil")
	}
}
