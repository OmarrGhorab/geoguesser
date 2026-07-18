package matchmaking

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsRecorder is the service-facing metrics surface.
type MetricsRecorder interface {
	ObserveCommand(command, outcome string, duration time.Duration)
	ObserveStatus(status string)
	ObserveFormation(outcome string, duration time.Duration)
	ObserveRecovery(outcome string)
	ObserveStaleEntry()
	ObserveDependencyFailure(dependency string)
	ObserveRateLimited(route string)
	ObserveRankedFormation(playlist, format, outcome string, duration time.Duration)
}

// NoopMetrics is a no-op MetricsRecorder.
type NoopMetrics struct{}

func (NoopMetrics) ObserveCommand(_, _ string, _ time.Duration)            {}
func (NoopMetrics) ObserveStatus(_ string)                                 {}
func (NoopMetrics) ObserveFormation(_ string, _ time.Duration)             {}
func (NoopMetrics) ObserveRecovery(_ string)                               {}
func (NoopMetrics) ObserveStaleEntry()                                     {}
func (NoopMetrics) ObserveDependencyFailure(_ string)                      {}
func (NoopMetrics) ObserveRateLimited(_ string)                            {}
func (NoopMetrics) ObserveRankedFormation(_, _, _ string, _ time.Duration) {}

// Metrics holds Prometheus instruments for matchmaking.
type Metrics struct {
	CommandsTotal          *prometheus.CounterVec
	CommandDurationSeconds *prometheus.HistogramVec
	StatusTotal            *prometheus.CounterVec
	FormationTotal         *prometheus.CounterVec
	FormationDuration      *prometheus.HistogramVec
	RecoveryTotal          *prometheus.CounterVec
	StaleEntriesTotal      prometheus.Counter
	DependencyFailures     *prometheus.CounterVec
	RateLimitedTotal       *prometheus.CounterVec
	// RankedFormationDuration is a histogram for ranked ticket formation by format/outcome.
	RankedFormationDuration *prometheus.HistogramVec
}

// NewMetrics registers matchmaking metrics against reg.
// Labels are intentionally bounded (no user IDs, tokens, or Redis keys).
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_commands_total",
			Help: "Matchmaking command attempts by command and outcome.",
		}, []string{"command", "outcome"}),
		CommandDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "matchmaking_command_duration_seconds",
			Help:    "Matchmaking command latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"command", "outcome"}),
		StatusTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_status_total",
			Help: "Matchmaking status responses by public status.",
		}, []string{"status"}),
		FormationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_formation_total",
			Help: "Match formation attempts by outcome.",
		}, []string{"outcome"}),
		FormationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "matchmaking_formation_duration_seconds",
			Help:    "Match formation duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		RecoveryTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_recovery_total",
			Help: "Claim/status recovery attempts by outcome.",
		}, []string{"outcome"}),
		StaleEntriesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "matchmaking_stale_entries_total",
			Help: "Stale queue entries pruned during scans.",
		}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_dependency_failures_total",
			Help: "Matchmaking dependency failures by dependency name.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchmaking_rate_limited_total",
			Help: "Matchmaking requests rejected by rate limiting.",
		}, []string{"route"}),
		RankedFormationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "matchmaking_ranked_formation_duration_seconds",
			Help:    "Ranked match formation duration by format and outcome (no user IDs).",
			Buckets: prometheus.DefBuckets,
		}, []string{"playlist", "format", "outcome"}),
	}

	for _, c := range []prometheus.Collector{
		m.CommandsTotal,
		m.CommandDurationSeconds,
		m.StatusTotal,
		m.FormationTotal,
		m.FormationDuration,
		m.RecoveryTotal,
		m.StaleEntriesTotal,
		m.DependencyFailures,
		m.RateLimitedTotal,
		m.RankedFormationDuration,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ObserveCommand records a command attempt.
func (m *Metrics) ObserveCommand(command, outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.CommandsTotal != nil {
		m.CommandsTotal.WithLabelValues(command, outcome).Inc()
	}
	if m.CommandDurationSeconds != nil && duration > 0 {
		m.CommandDurationSeconds.WithLabelValues(command, outcome).Observe(duration.Seconds())
	}
}

// ObserveStatus records a public status outcome.
func (m *Metrics) ObserveStatus(status string) {
	if m == nil || m.StatusTotal == nil {
		return
	}
	m.StatusTotal.WithLabelValues(status).Inc()
}

// ObserveFormation records a formation attempt.
func (m *Metrics) ObserveFormation(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.FormationTotal != nil {
		m.FormationTotal.WithLabelValues(outcome).Inc()
	}
	if m.FormationDuration != nil && duration > 0 {
		m.FormationDuration.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveRecovery records a recovery attempt.
func (m *Metrics) ObserveRecovery(outcome string) {
	if m == nil || m.RecoveryTotal == nil {
		return
	}
	m.RecoveryTotal.WithLabelValues(outcome).Inc()
}

// ObserveStaleEntry records a stale queue entry prune.
func (m *Metrics) ObserveStaleEntry() {
	if m == nil || m.StaleEntriesTotal == nil {
		return
	}
	m.StaleEntriesTotal.Inc()
}

// ObserveDependencyFailure records a redis/postgres failure.
func (m *Metrics) ObserveDependencyFailure(dependency string) {
	if m == nil || m.DependencyFailures == nil {
		return
	}
	m.DependencyFailures.WithLabelValues(dependency).Inc()
}

// ObserveRateLimited records a rate-limit rejection.
func (m *Metrics) ObserveRateLimited(route string) {
	if m == nil || m.RateLimitedTotal == nil {
		return
	}
	m.RateLimitedTotal.WithLabelValues(route).Inc()
}

// ObserveRankedFormation records ranked formation latency with bounded labels.
func (m *Metrics) ObserveRankedFormation(playlist, format, outcome string, duration time.Duration) {
	if m == nil || m.RankedFormationDuration == nil {
		return
	}
	if playlist == "" {
		playlist = PlaylistRanked
	}
	if format == "" {
		format = FormatSolo
	}
	if duration > 0 {
		m.RankedFormationDuration.WithLabelValues(playlist, format, outcome).Observe(duration.Seconds())
	} else {
		m.RankedFormationDuration.WithLabelValues(playlist, format, outcome).Observe(0)
	}
}

// RecordRateLimited is a handler-friendly observer for middleware hooks.
func (m *Metrics) RecordRateLimited(route string) {
	m.ObserveRateLimited(route)
}
