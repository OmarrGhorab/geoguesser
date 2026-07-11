package leaderboards

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records friends-leaderboard observations with bounded labels.
type Metrics struct {
	FriendsReadsTotal          *prometheus.CounterVec
	FriendsReadDurationSeconds *prometheus.HistogramVec
	DependencyFailures         *prometheus.CounterVec
	RateLimitedTotal           prometheus.Counter
}

// NewMetrics registers leaderboard metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		FriendsReadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "leaderboards_friends_reads_total",
			Help: "Friends leaderboard reads by outcome.",
		}, []string{"outcome"}),
		FriendsReadDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "leaderboards_friends_read_duration_seconds",
			Help:    "Friends leaderboard read latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "leaderboards_dependency_failures_total",
			Help: "Leaderboard dependency failures by dependency name.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "leaderboards_friends_rate_limited_total",
			Help: "Friends leaderboard requests rejected by rate limiting.",
		}),
	}
	for _, c := range []prometheus.Collector{
		m.FriendsReadsTotal,
		m.FriendsReadDurationSeconds,
		m.DependencyFailures,
		m.RateLimitedTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ObserveFriendsRead records a friends-leaderboard read attempt.
func (m *Metrics) ObserveFriendsRead(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	if m.FriendsReadsTotal != nil {
		m.FriendsReadsTotal.WithLabelValues(outcome).Inc()
	}
	if m.FriendsReadDurationSeconds != nil && duration > 0 {
		m.FriendsReadDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveDependencyFailure records a postgres/other dependency failure.
func (m *Metrics) ObserveDependencyFailure(dependency string) {
	if m == nil || m.DependencyFailures == nil {
		return
	}
	m.DependencyFailures.WithLabelValues(dependency).Inc()
}

// ObserveRateLimited records a friends-leaderboard rate-limit rejection.
func (m *Metrics) ObserveRateLimited() {
	if m == nil || m.RateLimitedTotal == nil {
		return
	}
	m.RateLimitedTotal.Inc()
}
