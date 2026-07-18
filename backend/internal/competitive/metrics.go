package competitive

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds Prometheus instruments for competitive progression.
// Labels are intentionally bounded (no user IDs, ratings, or guess content).
type Metrics struct {
	FinalizationTotal              *prometheus.CounterVec
	FinalizationDurationSeconds    *prometheus.HistogramVec
	WorkerSweepTotal               *prometheus.CounterVec
	WorkerProcessedTotal           *prometheus.CounterVec
	AbandonPenaltyTotal            prometheus.Counter
	ProfileReadTotal               *prometheus.CounterVec
	ProfileReadDurationSeconds     *prometheus.HistogramVec
	LeaderboardReadTotal           *prometheus.CounterVec
	LeaderboardReadDurationSeconds *prometheus.HistogramVec
	RolloverTotal                  *prometheus.CounterVec
	RolloverDurationSeconds        *prometheus.HistogramVec
	CacheInvalidateTotal           *prometheus.CounterVec
}

// NewMetrics registers competitive metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		FinalizationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_finalization_total",
			Help: "Ranked progression finalization attempts by outcome and format.",
		}, []string{"outcome", "format"}),
		FinalizationDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "competitive_finalization_duration_seconds",
			Help:    "Ranked progression finalization latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome", "format"}),
		WorkerSweepTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_progression_worker_sweeps_total",
			Help: "Progression retry worker sweeps by outcome.",
		}, []string{"outcome"}),
		WorkerProcessedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_progression_worker_matches_total",
			Help: "Progression retry worker matches processed by outcome bucket.",
		}, []string{"outcome"}),
		AbandonPenaltyTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "competitive_abandon_penalties_total",
			Help: "Number of abandon penalties applied during rated finalization.",
		}),
		ProfileReadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_profile_reads_total",
			Help: "Competitive profile reads by bounded outcome.",
		}, []string{"outcome"}),
		ProfileReadDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "competitive_profile_read_duration_seconds",
			Help:    "Competitive profile read latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		LeaderboardReadTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_leaderboard_reads_total",
			Help: "Competitive leaderboard reads by bounded outcome.",
		}, []string{"outcome"}),
		LeaderboardReadDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "competitive_leaderboard_read_duration_seconds",
			Help:    "Competitive leaderboard read latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		RolloverTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_season_rollover_total",
			Help: "Season rollover attempts by outcome; season_sequence is the new season sequence or 0.",
		}, []string{"outcome", "season_sequence"}),
		RolloverDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "competitive_season_rollover_duration_seconds",
			Help:    "Season rollover latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		CacheInvalidateTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "competitive_cache_invalidations_total",
			Help: "Competitive cache invalidations by scope.",
		}, []string{"scope"}),
	}
	for _, c := range []prometheus.Collector{
		m.FinalizationTotal,
		m.FinalizationDurationSeconds,
		m.WorkerSweepTotal,
		m.WorkerProcessedTotal,
		m.AbandonPenaltyTotal,
		m.ProfileReadTotal,
		m.ProfileReadDurationSeconds,
		m.LeaderboardReadTotal,
		m.LeaderboardReadDurationSeconds,
		m.RolloverTotal,
		m.RolloverDurationSeconds,
		m.CacheInvalidateTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ObserveFinalization records a finalization attempt.
// outcome examples: applied, replay, pending, error, not_ranked, not_ready.
// format is solo/duo/squad or empty when unknown.
func (m *Metrics) ObserveFinalization(outcome, format string, duration time.Duration) {
	if m == nil {
		return
	}
	outcome = boundLabel(outcome, "error")
	format = boundFormat(format)
	if m.FinalizationTotal != nil {
		m.FinalizationTotal.WithLabelValues(outcome, format).Inc()
	}
	if m.FinalizationDurationSeconds != nil && duration > 0 {
		m.FinalizationDurationSeconds.WithLabelValues(outcome, format).Observe(duration.Seconds())
	}
}

// ObserveWorkerSweep records a worker tick summary.
func (m *Metrics) ObserveWorkerSweep(outcome string, processed int) {
	if m == nil {
		return
	}
	outcome = boundLabel(outcome, "error")
	if m.WorkerSweepTotal != nil {
		m.WorkerSweepTotal.WithLabelValues(outcome).Inc()
	}
	if m.WorkerProcessedTotal != nil && processed > 0 {
		m.WorkerProcessedTotal.WithLabelValues(outcome).Add(float64(processed))
	}
}

// ObserveAbandonPenalty records one applied abandon penalty.
func (m *Metrics) ObserveAbandonPenalty() {
	if m == nil || m.AbandonPenaltyTotal == nil {
		return
	}
	m.AbandonPenaltyTotal.Inc()
}

// ObserveProfileRead records a profile read.
func (m *Metrics) ObserveProfileRead(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	outcome = boundReadOutcome(outcome)
	if m.ProfileReadTotal != nil {
		m.ProfileReadTotal.WithLabelValues(outcome).Inc()
	}
	if m.ProfileReadDurationSeconds != nil && duration > 0 {
		m.ProfileReadDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveLeaderboardRead records a leaderboard read.
func (m *Metrics) ObserveLeaderboardRead(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	outcome = boundReadOutcome(outcome)
	if m.LeaderboardReadTotal != nil {
		m.LeaderboardReadTotal.WithLabelValues(outcome).Inc()
	}
	if m.LeaderboardReadDurationSeconds != nil && duration > 0 {
		m.LeaderboardReadDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveRollover records a season rollover attempt.
// seasonSequence is the new season sequence (or 0 when skipped/error).
func (m *Metrics) ObserveRollover(outcome string, seasonSequence int, duration time.Duration) {
	if m == nil {
		return
	}
	outcome = boundRolloverOutcome(outcome)
	seqLabel := "0"
	if seasonSequence > 0 && seasonSequence < 100000 {
		seqLabel = itoa(seasonSequence)
	}
	if m.RolloverTotal != nil {
		m.RolloverTotal.WithLabelValues(outcome, seqLabel).Inc()
	}
	if m.RolloverDurationSeconds != nil && duration > 0 {
		m.RolloverDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveCacheInvalidate records a cache invalidation.
func (m *Metrics) ObserveCacheInvalidate(scope string) {
	if m == nil || m.CacheInvalidateTotal == nil {
		return
	}
	scope = boundLabel(scope, "season")
	m.CacheInvalidateTotal.WithLabelValues(scope).Inc()
}

func boundReadOutcome(v string) string {
	switch v {
	case "ok", "cache_hit", "unauthorized", "validation", "no_season", "error":
		return v
	case "":
		return "error"
	default:
		return boundLabel(v, "error")
	}
}

func boundRolloverOutcome(v string) string {
	switch v {
	case "applied", "replay", "skipped", "error":
		return v
	case "":
		return "error"
	default:
		return boundLabel(v, "error")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	// Small non-negative integers only (season sequences).
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func boundLabel(v, fallback string) string {
	if v == "" {
		return fallback
	}
	// Cap length to avoid accidental high-cardinality dumps.
	if len(v) > 32 {
		return v[:32]
	}
	return v
}

func boundFormat(format string) string {
	switch format {
	case "solo", "duo", "squad":
		return format
	case "":
		return "unknown"
	default:
		return "other"
	}
}
