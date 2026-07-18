package parties

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records parties-related Prometheus counters and histograms.
// Labels are intentionally bounded (command/read names and outcomes only).
type Metrics struct {
	CommandsTotal          *prometheus.CounterVec
	CommandDurationSeconds *prometheus.HistogramVec
	ReadsTotal             *prometheus.CounterVec
	DependencyFailures     *prometheus.CounterVec
	RateLimitedTotal       *prometheus.CounterVec
}

// NewMetrics registers parties metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "parties_commands_total",
			Help: "Party command attempts by command and outcome.",
		}, []string{"command", "outcome"}),
		CommandDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "parties_command_duration_seconds",
			Help:    "Party command latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"command", "outcome"}),
		ReadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "parties_reads_total",
			Help: "Party read attempts by name and outcome.",
		}, []string{"read", "outcome"}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "parties_dependency_failures_total",
			Help: "Party dependency failures by dependency name.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "parties_rate_limited_total",
			Help: "Party requests rejected by rate limiting.",
		}, []string{"route"}),
	}

	for _, c := range []prometheus.Collector{
		m.CommandsTotal,
		m.CommandDurationSeconds,
		m.ReadsTotal,
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

// ObserveRead records a read attempt.
func (m *Metrics) ObserveRead(read, outcome string) {
	if m == nil || m.ReadsTotal == nil {
		return
	}
	m.ReadsTotal.WithLabelValues(read, outcome).Inc()
}

// ObserveDependencyFailure records a postgres/redis dependency failure.
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
