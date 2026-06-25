package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Observability holds the baseline observability providers for the process.
// It is constructed once in cmd/api and injected everywhere it is needed.
type Observability struct {
	Logger  *slog.Logger
	Metrics *Metrics
	Tracer  Tracer
	Sentry  Sentry
}

// New builds the production observability stack from environment-level config.
func New(serviceName, version string) (*Observability, error) {
	logger := NewLogger(slog.LevelInfo)

	metrics, err := NewMetrics(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return &Observability{
		Logger:  logger,
		Metrics: metrics,
		Tracer:  NoopTracer{},
		Sentry:  NoopSentry{},
	}, nil
}

// NewLogger returns a JSON structured logger writing to stdout.
func NewLogger(level slog.Leveler) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// Metrics owns the process-level Prometheus registry and common collectors.
type Metrics struct {
	registry *prometheus.Registry

	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsTotal   *prometheus.CounterVec
	PostgresErrorsTotal prometheus.Counter
	RedisErrorsTotal    prometheus.Counter
}

// NewMetrics creates a fresh Prometheus registry with baseline collectors.
func NewMetrics(serviceName string) (*Metrics, error) {
	reg := prometheus.NewRegistry()

	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		return nil, err
	}

	m := &Metrics{
		registry: reg,
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		}, []string{"method", "path", "status"}),
		PostgresErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "postgres_errors_total",
			Help: "Total PostgreSQL errors.",
		}),
		RedisErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "redis_errors_total",
			Help: "Total Redis errors.",
		}),
	}

	for _, c := range []prometheus.Collector{
		m.HTTPRequestDuration,
		m.HTTPRequestsTotal,
		m.PostgresErrorsTotal,
		m.RedisErrorsTotal,
	} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}

	_ = serviceName // reserved for future service-info metric
	return m, nil
}

// Registry returns the Prometheus registry for the metrics endpoint.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Tracer is a thin abstraction over distributed tracing.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span is a minimal tracing span interface.
type Span interface {
	End()
	SetError(err error)
}

// NoopTracer is a no-op tracer used until OpenTelemetry is wired.
type NoopTracer struct{}

// StartSpan returns the original context and a no-op span.
func (NoopTracer) StartSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, NoopSpan{}
}

// NoopSpan is a no-op span.
type NoopSpan struct{}

// End is a no-op.
func (NoopSpan) End() {}

// SetError is a no-op.
func (NoopSpan) SetError(_ error) {}

// Sentry is a thin abstraction over error reporting.
type Sentry interface {
	CaptureException(err error)
}

// NoopSentry is a no-op error reporter used until Sentry is configured.
type NoopSentry struct{}

// CaptureException is a no-op.
func (NoopSentry) CaptureException(_ error) {}
