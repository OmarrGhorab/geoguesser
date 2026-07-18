package games

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics is the games package Prometheus surface for multiplayer/ranked.
// Labels are intentionally bounded (no user IDs, coordinates, or guess payloads).
type PrometheusMetrics struct {
	GuessDuration        *prometheus.HistogramVec
	CompletionsTotal     prometheus.Counter
	FinalizationTotal    *prometheus.CounterVec
	FinalizationDuration *prometheus.HistogramVec
	RoundCloseTotal      *prometheus.CounterVec
}

// NewPrometheusMetrics registers games multiplayer metrics against reg.
func NewPrometheusMetrics(reg prometheus.Registerer) (*PrometheusMetrics, error) {
	m := &PrometheusMetrics{
		GuessDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "games_guess_submission_duration_seconds",
			Help:    "Guess submission latency by outcome.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		CompletionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "games_completions_total",
			Help: "Completed games observed by the games service.",
		}),
		FinalizationTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "games_match_finalization_total",
			Help: "Matchmade game finalization outcomes (completed, pending, error).",
		}, []string{"playlist", "outcome"}),
		FinalizationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "game_match_finalization_duration_seconds",
			Help:    "Matchmade game finalization duration by playlist and outcome.",
			Buckets: prometheus.DefBuckets,
		}, []string{"playlist", "outcome"}),
		RoundCloseTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "game_round_close_total",
			Help: "Multiplayer round close outcomes by mode class.",
		}, []string{"mode_class", "outcome"}),
	}
	for _, c := range []prometheus.Collector{
		m.GuessDuration,
		m.CompletionsTotal,
		m.FinalizationTotal,
		m.FinalizationDuration,
		m.RoundCloseTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ObserveGuessSubmission implements MetricsRecorder.
func (m *PrometheusMetrics) ObserveGuessSubmission(outcome string, duration time.Duration) {
	if m == nil || m.GuessDuration == nil {
		return
	}
	m.GuessDuration.WithLabelValues(outcome).Observe(duration.Seconds())
}

// RecordGameCompleted implements MetricsRecorder.
func (m *PrometheusMetrics) RecordGameCompleted() {
	if m == nil || m.CompletionsTotal == nil {
		return
	}
	m.CompletionsTotal.Inc()
}

// ObserveFinalization records ranked/casual match finalization.
func (m *PrometheusMetrics) ObserveFinalization(playlist, outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	if playlist == "" {
		playlist = "unknown"
	}
	if m.FinalizationTotal != nil {
		m.FinalizationTotal.WithLabelValues(playlist, outcome).Inc()
	}
	if m.FinalizationDuration != nil && duration > 0 {
		m.FinalizationDuration.WithLabelValues(playlist, outcome).Observe(duration.Seconds())
	}
}

// ObserveRoundClose records multiplayer round closure.
func (m *PrometheusMetrics) ObserveRoundClose(modeClass, outcome string) {
	if m == nil || m.RoundCloseTotal == nil {
		return
	}
	m.RoundCloseTotal.WithLabelValues(modeClass, outcome).Inc()
}

// NoopGameMetrics is a no-op MetricsRecorder for games.
type NoopGameMetrics struct{}

// ObserveGuessSubmission implements MetricsRecorder.
func (NoopGameMetrics) ObserveGuessSubmission(string, time.Duration) {}

// RecordGameCompleted implements MetricsRecorder.
func (NoopGameMetrics) RecordGameCompleted() {}
