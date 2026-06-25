package health

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

// Pinger checks the health of a dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler exposes liveness, readiness, and metrics probes.
type Handler struct {
	version       string
	logger        *slog.Logger
	observability *observability.Observability
	pingers       map[string]Pinger
}

// HealthResponse is the body returned by the liveness endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// ReadinessResponse is the body returned by the readiness endpoint.
type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// NewHandlerWithPingers creates a health handler from explicit dependency pingers.
// It is the flexible constructor used by tests and the production wiring layer.
func NewHandlerWithPingers(version string, logger *slog.Logger, pingers map[string]Pinger) *Handler {
	return &Handler{
		version: version,
		logger:  logger,
		pingers: pingers,
	}
}

// NewHandlerWithObservability creates a health handler with the full
// observability stack, including the Prometheus metrics endpoint.
func NewHandlerWithObservability(version string, logger *slog.Logger, obs *observability.Observability, pingers map[string]Pinger) *Handler {
	return &Handler{
		version:       version,
		logger:        logger,
		observability: obs,
		pingers:       pingers,
	}
}

// Health returns a liveness response. It does not check dependencies so the
// process can be restarted by orchestrators if it can no longer serve at all.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	apphttp.OK(w, r, HealthResponse{Status: "ok", Version: h.version})
}

// Ready checks configured dependencies and reports readiness.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string, len(h.pingers))
	status := http.StatusOK
	for name := range h.pingers {
		checks[name] = "ok"
	}

	for name, pinger := range h.pingers {
		if err := pinger.Ping(ctx); err != nil {
			checks[name] = "error"
			status = http.StatusServiceUnavailable
			h.logger.ErrorContext(ctx, "readiness check failed",
				slog.String("dependency", name),
				slog.Any("error", err),
			)
		}
	}

	bodyStatus := "ready"
	if status != http.StatusOK {
		bodyStatus = "not_ready"
	}

	apphttp.JSON(w, r, status, ReadinessResponse{
		Status: bodyStatus,
		Checks: checks,
	})
}

// Metrics exposes Prometheus metrics in text exposition format.
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if h.observability == nil || h.observability.Metrics == nil {
		apphttp.Error(w, r, h.logger, apphttp.ErrInternal.WithCause(errors.New("metrics not configured")))
		return
	}

	promhttp.HandlerFor(h.observability.Metrics.Registry(), promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}).ServeHTTP(w, r)
}
