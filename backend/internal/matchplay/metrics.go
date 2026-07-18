package matchplay

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsRecorder is the service-facing metrics surface.
// Labels are bounded: no user IDs, tokens, coordinates, or Redis keys.
type MetricsRecorder interface {
	ObserveCommand(command, outcome string, duration time.Duration)
	ObserveSweep(kind, outcome string)
	ObserveDependencyFailure(dependency string)
	ObserveRateLimited(route string)
	ObserveIdempotency(outcome string)
	ObserveCSRFRejection()
	ObserveAbandon(reason, outcome string)
	ObserveChatMessage(outcome string)
	ObserveChatAttachment(outcome string)
	ObserveChatMute(outcome string)
	ObserveChatReport(outcome string)
	ObserveCleanup(kind, outcome string)
	ObserveCleanupDeleted(kind string, count int)
	ObserveMarker(outcome string)
	ObserveEventPublish(eventType, audienceKind, outcome string)
	// ObserveSpectate records spectate selection/denial/fallback with format label only.
	ObserveSpectate(outcome, format string)
	// ObserveViewUpdate records view-update outcomes (material/throttled/rejected/ok).
	ObserveViewUpdate(outcome, format string)
}

// NoopMetrics is a no-op MetricsRecorder.
type NoopMetrics struct{}

func (NoopMetrics) ObserveCommand(_, _ string, _ time.Duration) {}
func (NoopMetrics) ObserveSweep(_, _ string)                    {}
func (NoopMetrics) ObserveDependencyFailure(_ string)           {}
func (NoopMetrics) ObserveRateLimited(_ string)                 {}
func (NoopMetrics) ObserveIdempotency(_ string)                 {}
func (NoopMetrics) ObserveCSRFRejection()                       {}
func (NoopMetrics) ObserveAbandon(_, _ string)                  {}
func (NoopMetrics) ObserveChatMessage(_ string)                 {}
func (NoopMetrics) ObserveChatAttachment(_ string)              {}
func (NoopMetrics) ObserveChatMute(_ string)                    {}
func (NoopMetrics) ObserveChatReport(_ string)                  {}
func (NoopMetrics) ObserveCleanup(_, _ string)                  {}
func (NoopMetrics) ObserveCleanupDeleted(_ string, _ int)       {}
func (NoopMetrics) ObserveMarker(_ string)                      {}
func (NoopMetrics) ObserveEventPublish(_, _, _ string)          {}
func (NoopMetrics) ObserveSpectate(_, _ string)                 {}
func (NoopMetrics) ObserveViewUpdate(_, _ string)               {}

// Metrics holds Prometheus instruments for matchplay.
type Metrics struct {
	CommandsTotal          *prometheus.CounterVec
	CommandDurationSeconds *prometheus.HistogramVec
	SweepTotal             *prometheus.CounterVec
	DependencyFailures     *prometheus.CounterVec
	RateLimitedTotal       *prometheus.CounterVec
	IdempotencyTotal       *prometheus.CounterVec
	CSRFRejectionsTotal    prometheus.Counter
	AbandonTotal           *prometheus.CounterVec
	ChatMessagesTotal      *prometheus.CounterVec
	ChatAttachmentsTotal   *prometheus.CounterVec
	ChatMutesTotal         *prometheus.CounterVec
	ChatReportsTotal       *prometheus.CounterVec
	CleanupTotal           *prometheus.CounterVec
	CleanupDeletedTotal    *prometheus.CounterVec
	MarkerTotal            *prometheus.CounterVec
	EventPublishTotal      *prometheus.CounterVec
	SpectateTotal          *prometheus.CounterVec
	ViewUpdateTotal        *prometheus.CounterVec
}

// NewMetrics registers matchplay metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_commands_total",
			Help: "Matchplay command attempts by command and outcome.",
		}, []string{"command", "outcome"}),
		CommandDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "matchplay_command_duration_seconds",
			Help:    "Matchplay command latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"command", "outcome"}),
		SweepTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_sweep_total",
			Help: "Match lifecycle sweep outcomes by kind.",
		}, []string{"kind", "outcome"}),
		DependencyFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_dependency_failures_total",
			Help: "Matchplay dependency failures by dependency name.",
		}, []string{"dependency"}),
		RateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_rate_limited_total",
			Help: "Matchplay requests rejected by rate limiting.",
		}, []string{"route"}),
		IdempotencyTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_idempotency_total",
			Help: "Matchplay idempotency outcomes (replay, conflict, miss).",
		}, []string{"outcome"}),
		CSRFRejectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "matchplay_csrf_rejections_total",
			Help: "Matchplay unsafe requests rejected by CSRF middleware.",
		}),
		AbandonTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_abandon_total",
			Help: "Match abandonment outcomes by reason and result (no user IDs).",
		}, []string{"reason", "outcome"}),
		ChatMessagesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_chat_messages_total",
			Help: "Team chat send outcomes (bounded outcome labels only).",
		}, []string{"outcome"}),
		ChatAttachmentsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_chat_attachments_total",
			Help: "Team chat attachment access/bind outcomes.",
		}, []string{"outcome"}),
		ChatMutesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_chat_mutes_total",
			Help: "Team chat mute/unmute outcomes.",
		}, []string{"outcome"}),
		ChatReportsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_chat_reports_total",
			Help: "Team chat report outcomes.",
		}, []string{"outcome"}),
		CleanupTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_cleanup_total",
			Help: "Chat/raw retention cleanup pass outcomes by kind.",
		}, []string{"kind", "outcome"}),
		CleanupDeletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_cleanup_deleted_total",
			Help: "Rows/objects deleted by chat retention cleanup (kind=messages|files|uploads|objects).",
		}, []string{"kind"}),
		MarkerTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_marker_commands_total",
			Help: "Team marker collaboration outcomes (bounded labels only).",
		}, []string{"outcome"}),
		EventPublishTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_event_publish_total",
			Help: "Matchplay event publish attempts by type, audience kind, and outcome.",
		}, []string{"event_type", "audience_kind", "outcome"}),
		SpectateTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_spectate_total",
			Help: "Spectate select outcomes by outcome and format (no user IDs).",
		}, []string{"outcome", "format"}),
		ViewUpdateTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "matchplay_view_update_total",
			Help: "Provider-safe view update outcomes by outcome and format.",
		}, []string{"outcome", "format"}),
	}

	for _, c := range []prometheus.Collector{
		m.CommandsTotal,
		m.CommandDurationSeconds,
		m.SweepTotal,
		m.DependencyFailures,
		m.RateLimitedTotal,
		m.IdempotencyTotal,
		m.CSRFRejectionsTotal,
		m.AbandonTotal,
		m.ChatMessagesTotal,
		m.ChatAttachmentsTotal,
		m.ChatMutesTotal,
		m.ChatReportsTotal,
		m.CleanupTotal,
		m.CleanupDeletedTotal,
		m.MarkerTotal,
		m.EventPublishTotal,
		m.SpectateTotal,
		m.ViewUpdateTotal,
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

// ObserveSweep records a lifecycle sweep outcome.
func (m *Metrics) ObserveSweep(kind, outcome string) {
	if m == nil || m.SweepTotal == nil {
		return
	}
	m.SweepTotal.WithLabelValues(kind, outcome).Inc()
}

// ObserveDependencyFailure records a redis/postgres/parties failure.
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

// ObserveIdempotency records an idempotency outcome.
func (m *Metrics) ObserveIdempotency(outcome string) {
	if m == nil || m.IdempotencyTotal == nil {
		return
	}
	m.IdempotencyTotal.WithLabelValues(outcome).Inc()
}

// ObserveCSRFRejection records a CSRF rejection.
func (m *Metrics) ObserveCSRFRejection() {
	if m == nil || m.CSRFRejectionsTotal == nil {
		return
	}
	m.CSRFRejectionsTotal.Inc()
}

// ObserveAbandon records a ranked/casual abandon outcome.
// reason: explicit|disconnect; outcome: forfeited|replay|error (bounded labels only).
func (m *Metrics) ObserveAbandon(reason, outcome string) {
	if m == nil || m.AbandonTotal == nil {
		return
	}
	m.AbandonTotal.WithLabelValues(reason, outcome).Inc()
}

// ObserveChatMessage records a team chat send outcome.
func (m *Metrics) ObserveChatMessage(outcome string) {
	if m == nil || m.ChatMessagesTotal == nil {
		return
	}
	m.ChatMessagesTotal.WithLabelValues(outcome).Inc()
}

// ObserveChatAttachment records attachment bind/sign outcomes.
func (m *Metrics) ObserveChatAttachment(outcome string) {
	if m == nil || m.ChatAttachmentsTotal == nil {
		return
	}
	m.ChatAttachmentsTotal.WithLabelValues(outcome).Inc()
}

// ObserveChatMute records mute/unmute outcomes.
func (m *Metrics) ObserveChatMute(outcome string) {
	if m == nil || m.ChatMutesTotal == nil {
		return
	}
	m.ChatMutesTotal.WithLabelValues(outcome).Inc()
}

// ObserveChatReport records report outcomes.
func (m *Metrics) ObserveChatReport(outcome string) {
	if m == nil || m.ChatReportsTotal == nil {
		return
	}
	m.ChatReportsTotal.WithLabelValues(outcome).Inc()
}

// ObserveCleanup records a retention cleanup pass outcome.
// kind: messages|raw_uploads; outcome: ok|error (bounded).
func (m *Metrics) ObserveCleanup(kind, outcome string) {
	if m == nil || m.CleanupTotal == nil {
		return
	}
	m.CleanupTotal.WithLabelValues(kind, outcome).Inc()
}

// ObserveCleanupDeleted increments deleted-entity counters by kind and count.
func (m *Metrics) ObserveCleanupDeleted(kind string, count int) {
	if m == nil || m.CleanupDeletedTotal == nil || count <= 0 {
		return
	}
	m.CleanupDeletedTotal.WithLabelValues(kind).Add(float64(count))
}

// RecordRateLimited is a handler-friendly observer for middleware hooks.
func (m *Metrics) RecordRateLimited(route string) {
	m.ObserveRateLimited(route)
}

// RecordCSRFRejection is a handler-friendly CSRF observer.
func (m *Metrics) RecordCSRFRejection() {
	m.ObserveCSRFRejection()
}

// ObserveMarker records a marker collaboration outcome.
func (m *Metrics) ObserveMarker(outcome string) {
	if m == nil || m.MarkerTotal == nil {
		return
	}
	m.MarkerTotal.WithLabelValues(outcome).Inc()
}

// ObserveEventPublish records an audience-targeted event publish.
func (m *Metrics) ObserveEventPublish(eventType, audienceKind, outcome string) {
	if m == nil || m.EventPublishTotal == nil {
		return
	}
	m.EventPublishTotal.WithLabelValues(eventType, audienceKind, outcome).Inc()
}

// ObserveSpectate records spectate selection, denial, or fallback.
// outcome: selected|fallback|denied|error (bounded). format: solo|duo|squad|unknown.
func (m *Metrics) ObserveSpectate(outcome, format string) {
	if m == nil || m.SpectateTotal == nil {
		return
	}
	m.SpectateTotal.WithLabelValues(boundSpectateOutcome(outcome), boundFormat(format)).Inc()
}

// ObserveViewUpdate records a view-update outcome.
// outcome: ok|unchanged|throttled|rejected|error (bounded).
func (m *Metrics) ObserveViewUpdate(outcome, format string) {
	if m == nil || m.ViewUpdateTotal == nil {
		return
	}
	m.ViewUpdateTotal.WithLabelValues(boundViewOutcome(outcome), boundFormat(format)).Inc()
}

func boundSpectateOutcome(outcome string) string {
	switch outcome {
	case "selected", "fallback", "denied", "error":
		return outcome
	default:
		return "other"
	}
}

func boundViewOutcome(outcome string) string {
	switch outcome {
	case "ok", "unchanged", "throttled", "rejected", "error":
		return outcome
	default:
		return "other"
	}
}

func boundFormat(format string) string {
	switch format {
	case FormatSolo, FormatDuo, FormatSquad:
		return format
	default:
		return "unknown"
	}
}
