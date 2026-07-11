package friends

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records friends-related Prometheus counters and histograms.
// Labels are intentionally bounded (command/list names and outcomes only).
type Metrics struct {
	CommandsTotal          *prometheus.CounterVec
	CommandDurationSeconds *prometheus.HistogramVec
	ListsTotal             *prometheus.CounterVec
	DependencyFailures     *prometheus.CounterVec
	RateLimitedTotal       *prometheus.CounterVec
}

// NewMetrics registers friends metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "friends_commands_total",
			Help: "Friends command attempts by command and outcome.",
		}, []string{"command", "outcome"}),
		CommandDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "friends_command_duration_seconds",
			Help:    "Friends command latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"command", "outcome"}),
		ListsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "friends_lists_total",
			Help: "Friends list reads by list name and outcome.",
		}, []string{"list", "outcome"}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "friends_dependency_failures_total",
			Help: "Friends dependency failures by dependency name.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "friends_rate_limited_total",
			Help: "Friends requests rejected by rate limiting.",
		}, []string{"route"}),
	}

	for _, c := range []prometheus.Collector{
		m.CommandsTotal,
		m.CommandDurationSeconds,
		m.ListsTotal,
		m.DependencyFailures,
		m.RateLimitedTotal,
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

// ObserveList records a list read.
func (m *Metrics) ObserveList(list, outcome string) {
	if m == nil || m.ListsTotal == nil {
		return
	}
	m.ListsTotal.WithLabelValues(list, outcome).Inc()
}

// ObserveDependencyFailure records a postgres/other dependency failure.
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

// RecordRateLimited is a handler-friendly observer for middleware hooks.
func (m *Metrics) RecordRateLimited(route string) {
	m.ObserveRateLimited(route)
}
