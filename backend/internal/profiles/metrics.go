package profiles

import "github.com/prometheus/client_golang/prometheus"

// Metrics records profile-related Prometheus counters. It is safe to use a
// nil *Metrics; all methods no-op in that case so tests and call sites that
// do not care about metrics do not need to wire a registry.
type Metrics struct {
	ProfileReadsTotal         prometheus.Counter
	ProfileUpdatesTotal       *prometheus.CounterVec
	ProfileValidationFailures prometheus.Counter
	PublicStatsReadsTotal     prometheus.Counter
	GameHistoryReadsTotal     prometheus.Counter
	ProfileRateLimitedTotal   prometheus.Counter
}

// NewMetrics creates profile metrics registered against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		ProfileReadsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "profile_reads_total",
			Help: "Total current-profile read requests.",
		}),
		ProfileUpdatesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "profile_updates_total",
			Help: "Total profile update attempts by outcome.",
		}, []string{"outcome"}),
		ProfileValidationFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "profile_validation_failures_total",
			Help: "Total profile update validation failures.",
		}),
		PublicStatsReadsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "profile_public_stats_reads_total",
			Help: "Total public stats read requests.",
		}),
		GameHistoryReadsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "profile_game_history_reads_total",
			Help: "Total game history read requests.",
		}),
		ProfileRateLimitedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "profile_updates_rate_limited_total",
			Help: "Total profile update requests rejected by rate limiting.",
		}),
	}

	for _, c := range []prometheus.Collector{
		m.ProfileReadsTotal,
		m.ProfileUpdatesTotal,
		m.ProfileValidationFailures,
		m.PublicStatsReadsTotal,
		m.GameHistoryReadsTotal,
		m.ProfileRateLimitedTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// RecordProfileRead records a current-profile read.
func (m *Metrics) RecordProfileRead() {
	if m == nil || m.ProfileReadsTotal == nil {
		return
	}
	m.ProfileReadsTotal.Inc()
}

// RecordProfileUpdate records a profile update attempt by outcome, e.g.
// "success" or "validation_failed".
func (m *Metrics) RecordProfileUpdate(outcome string) {
	if m == nil || m.ProfileUpdatesTotal == nil {
		return
	}
	m.ProfileUpdatesTotal.WithLabelValues(outcome).Inc()
}

// RecordValidationFailure records a profile update validation failure.
func (m *Metrics) RecordValidationFailure() {
	if m == nil || m.ProfileValidationFailures == nil {
		return
	}
	m.ProfileValidationFailures.Inc()
}

// RecordPublicStatsRead records a public stats read.
func (m *Metrics) RecordPublicStatsRead() {
	if m == nil || m.PublicStatsReadsTotal == nil {
		return
	}
	m.PublicStatsReadsTotal.Inc()
}

// RecordGameHistoryRead records a game history read.
func (m *Metrics) RecordGameHistoryRead() {
	if m == nil || m.GameHistoryReadsTotal == nil {
		return
	}
	m.GameHistoryReadsTotal.Inc()
}

// RecordRateLimited records a profile update rejected by rate limiting.
func (m *Metrics) RecordRateLimited() {
	if m == nil || m.ProfileRateLimitedTotal == nil {
		return
	}
	m.ProfileRateLimitedTotal.Inc()
}
