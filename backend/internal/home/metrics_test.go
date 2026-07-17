package home

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRegisterAndObserve(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics, err := NewMetrics(registry)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}

	metrics.ObserveRead("success", 10*time.Millisecond)
	metrics.ObserveDependencyFailure("maps")
	metrics.RecordRateLimited()

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(families) != 4 {
		t.Fatalf("metric family count = %d, want 4", len(families))
	}
}
