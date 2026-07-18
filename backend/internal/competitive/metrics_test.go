package competitive_test

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestMetrics_ObserveWithoutUserIDs(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewRegistry()
	m, err := competitive.NewMetrics(reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	m.ObserveFinalization("applied", "solo", 12*time.Millisecond)
	m.ObserveFinalization("replay", "duo", time.Millisecond)
	m.ObserveWorkerSweep("ok", 3)
	m.ObserveAbandonPenalty()
	m.ObserveProfileRead("ok", 5*time.Millisecond)
	m.ObserveLeaderboardRead("cache_hit", 2*time.Millisecond)
	m.ObserveRollover("applied", 3, 15*time.Millisecond)
	m.ObserveCacheInvalidate("season")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(families) == 0 {
		t.Fatal("expected metric families")
	}
	for _, f := range families {
		name := f.GetName()
		// Ensure metric names stay in the competitive_* namespace.
		if len(name) < len("competitive_") || name[:len("competitive_")] != "competitive_" {
			t.Fatalf("unexpected metric name %q", name)
		}
		for _, metric := range f.GetMetric() {
			for _, lp := range metric.GetLabel() {
				// Labels must not look like UUIDs / user identifiers.
				v := lp.GetValue()
				if len(v) == 36 && v[8] == '-' && v[13] == '-' {
					t.Fatalf("label %s looks like a user id: %s", lp.GetName(), v)
				}
			}
		}
	}
}
