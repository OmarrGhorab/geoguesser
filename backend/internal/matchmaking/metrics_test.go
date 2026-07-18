package matchmaking_test

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raven/geoguess/backend/internal/matchmaking"
)

func TestNewMetricsRegisters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := matchmaking.NewMetrics(reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	m.ObserveCommand("join", "success", 5*time.Millisecond)
	m.ObserveStatus(matchmaking.PublicStatusSearching)
	m.ObserveFormation("matched", 10*time.Millisecond)
	m.ObserveRankedFormation(matchmaking.PlaylistRanked, matchmaking.FormatSolo, "formed", 10*time.Millisecond)
	m.ObserveRecovery("finalized")
	m.ObserveStaleEntry()
	m.ObserveDependencyFailure("redis")
	m.ObserveRateLimited("queue")

	metricFamilies, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	if len(metricFamilies) == 0 {
		t.Fatal("expected registered metrics")
	}
}

func TestNoopMetricsSafe(t *testing.T) {
	var m matchmaking.NoopMetrics
	m.ObserveCommand("join", "success", time.Millisecond)
	m.ObserveStatus(matchmaking.PublicStatusNotQueued)
	m.ObserveFormation("none", 0)
	m.ObserveRankedFormation(matchmaking.PlaylistRanked, matchmaking.FormatDuo, "formed", 0)
	m.ObserveRecovery("none")
	m.ObserveStaleEntry()
	m.ObserveDependencyFailure("redis")
	m.ObserveRateLimited("status")
}

func TestMetricsNoHighCardinalityUserLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := matchmaking.NewMetrics(reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	// Regression: outcomes must stay categorical.
	for _, outcome := range []string{"formed", "no_pair", "ineligible", "locations_unavailable", "formation_failed"} {
		m.ObserveFormation(outcome, time.Millisecond)
	}
	for _, status := range []string{
		matchmaking.PublicStatusNotQueued,
		matchmaking.PublicStatusSearching,
		matchmaking.PublicStatusMatched,
		matchmaking.PublicStatusTemporarilyUnavailable,
	} {
		m.ObserveStatus(status)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	for _, family := range families {
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				v := label.GetValue()
				if strings.Contains(v, "@") || strings.Contains(v, "user:") {
					t.Fatalf("metric leaked identity-like label %s=%q", label.GetName(), v)
				}
			}
		}
	}
}

func TestMetricsBoundedLabelsOnly(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := matchmaking.NewMetrics(reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	// Bounded categorical labels only — never user IDs / tokens / Redis keys.
	m.ObserveCommand("join", "completed", time.Millisecond)
	m.ObserveCommand("leave", "unauthorized", 0)
	m.ObserveStatus(matchmaking.PublicStatusTemporarilyUnavailable)
	m.ObserveFormation("formation_failed", time.Millisecond)
	m.ObserveRecovery("claimed_durable_hit")
	m.ObserveDependencyFailure("postgres")
	m.ObserveDependencyFailure("redis")
	m.ObserveRateLimited("mm-cmd")
	m.ObserveRateLimited("mm-status")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	for _, family := range families {
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				name := label.GetName()
				value := label.GetValue()
				switch name {
				case "command", "outcome", "status", "dependency", "route", "playlist", "format":
					if value == "" {
						t.Fatalf("empty label value for %s", name)
					}
					if len(value) > 64 {
						t.Fatalf("label %s value too long: %q", name, value)
					}
				default:
					t.Fatalf("unexpected metric label %q", name)
				}
			}
		}
	}
}
