package friends_test

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raven/geoguess/backend/internal/friends"
)

func TestNewMetricsRegisters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := friends.NewMetrics(reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	m.ObserveCommand("request", "success", 10*time.Millisecond)
	m.ObserveList("friends", "success")
	m.ObserveDependencyFailure("postgres")
	m.ObserveRateLimited("friends-request")
	m.RecordRateLimited("friends-read")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	if len(families) == 0 {
		t.Fatal("expected metrics families")
	}
}

func TestNilMetricsNoop(t *testing.T) {
	var m *friends.Metrics
	m.ObserveCommand("request", "success", time.Millisecond)
	m.ObserveList("friends", "success")
	m.ObserveDependencyFailure("postgres")
	m.ObserveRateLimited("friends-read")
	m.RecordRateLimited("friends-read")
}
