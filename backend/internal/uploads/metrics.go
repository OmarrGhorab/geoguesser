package uploads

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsRecorder is the uploads service-facing metrics surface.
// Labels are bounded: no user IDs, storage keys, tokens, or raw content.
type MetricsRecorder interface {
	ObserveSanitize(outcome string, duration time.Duration)
	ObserveRawCleanup(outcome string)
	ObserveUpload(command, purpose, outcome string, duration time.Duration)
	ObserveStorageFailure(operation string)
}

// NoopMetrics is a no-op MetricsRecorder.
type NoopMetrics struct{}

func (NoopMetrics) ObserveSanitize(string, time.Duration)               {}
func (NoopMetrics) ObserveRawCleanup(string)                            {}
func (NoopMetrics) ObserveUpload(string, string, string, time.Duration) {}
func (NoopMetrics) ObserveStorageFailure(string)                        {}

// Metrics holds Prometheus instruments for uploads / image sanitization.
type Metrics struct {
	SanitizeTotal           *prometheus.CounterVec
	SanitizeDurationSeconds *prometheus.HistogramVec
	RawCleanupTotal         *prometheus.CounterVec
	UploadsTotal            *prometheus.CounterVec
	UploadDurationSeconds   *prometheus.HistogramVec
	StorageFailuresTotal    *prometheus.CounterVec
}

// NewMetrics registers uploads metrics against reg.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		SanitizeTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "uploads_sanitize_total",
			Help: "Image sanitization attempts by bounded outcome label.",
		}, []string{"outcome"}),
		SanitizeDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "uploads_sanitize_duration_seconds",
			Help:    "Image sanitization latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"outcome"}),
		RawCleanupTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "uploads_raw_cleanup_total",
			Help: "Raw upload object cleanup outcomes (deleted, retry).",
		}, []string{"outcome"}),
		UploadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "uploads_commands_total",
			Help: "Upload commands by command, purpose, and outcome.",
		}, []string{"command", "purpose", "outcome"}),
		UploadDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "uploads_command_duration_seconds",
			Help:    "Upload command latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"command", "purpose", "outcome"}),
		StorageFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "uploads_storage_failures_total",
			Help: "Object storage failures by operation (get, put, delete, head, presign).",
		}, []string{"operation"}),
	}

	for _, c := range []prometheus.Collector{
		m.SanitizeTotal,
		m.SanitizeDurationSeconds,
		m.RawCleanupTotal,
		m.UploadsTotal,
		m.UploadDurationSeconds,
		m.StorageFailuresTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// ObserveSanitize records a sanitization attempt.
// outcome is a bounded label: ready|unsafe|invalid|too_large|not_found|canceled|storage_unavailable|error
func (m *Metrics) ObserveSanitize(outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	outcome = boundLabel(outcome, "error")
	if m.SanitizeTotal != nil {
		m.SanitizeTotal.WithLabelValues(outcome).Inc()
	}
	if m.SanitizeDurationSeconds != nil && duration > 0 {
		m.SanitizeDurationSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
	}
}

// ObserveRawCleanup records raw object cleanup.
// outcome: deleted|retry
func (m *Metrics) ObserveRawCleanup(outcome string) {
	if m == nil || m.RawCleanupTotal == nil {
		return
	}
	m.RawCleanupTotal.WithLabelValues(boundLabel(outcome, "retry")).Inc()
}

// ObserveUpload records create/complete command outcomes.
// purpose: general|team_chat; outcome: ok|rejected|forbidden|error|...
func (m *Metrics) ObserveUpload(command, purpose, outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	purpose = boundPurpose(purpose)
	outcome = boundLabel(outcome, "error")
	command = boundLabel(command, "unknown")
	if m.UploadsTotal != nil {
		m.UploadsTotal.WithLabelValues(command, purpose, outcome).Inc()
	}
	if m.UploadDurationSeconds != nil && duration > 0 {
		m.UploadDurationSeconds.WithLabelValues(command, purpose, outcome).Observe(duration.Seconds())
	}
}

// ObserveStorageFailure records a storage dependency failure.
func (m *Metrics) ObserveStorageFailure(operation string) {
	if m == nil || m.StorageFailuresTotal == nil {
		return
	}
	m.StorageFailuresTotal.WithLabelValues(boundLabel(operation, "unknown")).Inc()
}

func boundPurpose(p string) string {
	switch p {
	case PurposeTeamChat:
		return PurposeTeamChat
	case PurposeGeneral, "":
		return PurposeGeneral
	default:
		return "other"
	}
}

func boundLabel(v, fallback string) string {
	if v == "" {
		return fallback
	}
	// Cap length to keep cardinality bounded even if a caller mislabels.
	if len(v) > 32 {
		return fallback
	}
	return v
}
