package home

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records bounded-label authenticated-home observations.
type Metrics struct {
	ReadsTotal          *prometheus.CounterVec
	ReadDurationSeconds *prometheus.HistogramVec
	DependencyFailures  *prometheus.CounterVec
	RateLimitedTotal    prometheus.Counter
}

// NewMetrics registers authenticated-home metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	metrics := &Metrics{
		ReadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "home_reads_total",
			Help: "Authenticated home reads by outcome.",
		}, []string{"outcome"}),
		ReadDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "home_read_duration_seconds",
			Help:    "Authenticated home read latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "home_dependency_failures_total",
			Help: "Authenticated home dependency failures by dependency.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "home_rate_limited_total",
			Help: "Authenticated home requests rejected by rate limiting.",
		}),
	}

	for _, collector := range []prometheus.Collector{
		metrics.ReadsTotal,
		metrics.ReadDurationSeconds,
		metrics.DependencyFailures,
		metrics.RateLimitedTotal,
	} {
		if err := reg.Register(collector); err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

// ObserveRead records an authenticated-home read attempt.
func (m *Metrics) ObserveRead(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.ReadsTotal != nil {
		m.ReadsTotal.WithLabelValues(outcome).Inc()
	}
	if m.ReadDurationSeconds != nil {
		m.ReadDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveDependencyFailure records a failure from a bounded dependency name.
func (m *Metrics) ObserveDependencyFailure(dependency string) {
	if m == nil || m.DependencyFailures == nil {
		return
	}
	m.DependencyFailures.WithLabelValues(dependency).Inc()
}

// RecordRateLimited is compatible with the route middleware observer.
func (m *Metrics) RecordRateLimited() {
	if m == nil || m.RateLimitedTotal == nil {
		return
	}
	m.RateLimitedTotal.Inc()
}
