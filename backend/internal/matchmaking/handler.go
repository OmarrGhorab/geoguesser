package matchmaking

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

// ServiceAPI is the handler-facing service surface.
type ServiceAPI interface {
	JoinQueue(ctx context.Context, sess *session.Context, req JoinQueueRequest) (*StatusResponse, error)
	LeaveQueue(ctx context.Context, sess *session.Context) error
	GetStatus(ctx context.Context, sess *session.Context) (*StatusResponse, error)
}

// Handler serves matchmaking HTTP endpoints.
type Handler struct {
	service ServiceAPI
	logger  *slog.Logger
	metrics MetricsRecorder
}

// NewHandler constructs a matchmaking handler.
func NewHandler(service ServiceAPI, logger *slog.Logger) *Handler {
	return NewHandlerWithMetrics(service, logger, nil)
}

// NewHandlerWithMetrics constructs a matchmaking handler with rate-limit observers.
func NewHandlerWithMetrics(service ServiceAPI, logger *slog.Logger, metrics MetricsRecorder) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &Handler{service: service, logger: logger, metrics: metrics}
}

// RecordCommandRateLimited records a matchmaking command rate-limit rejection.
func (h *Handler) RecordCommandRateLimited(_ *http.Request) {
	if h == nil {
		return
	}
	h.metrics.ObserveRateLimited("mm-cmd")
}

// RecordStatusRateLimited records a matchmaking status rate-limit rejection.
func (h *Handler) RecordStatusRateLimited(_ *http.Request) {
	if h == nil {
		return
	}
	h.metrics.ObserveRateLimited("mm-status")
}

// RegisterRoutes mounts matchmaking routes on the provided router.
// Callers must wrap this group with registered auth, CSRF (for unsafe methods),
// and per-route rate limits.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/matchmaking/queue", h.JoinQueue)
	r.Delete("/matchmaking/queue", h.LeaveQueue)
	r.Get("/matchmaking/status", h.GetStatus)
}

// JoinQueue handles POST /matchmaking/queue.
func (h *Handler) JoinQueue(w http.ResponseWriter, r *http.Request) {
	var req JoinQueueRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}
	resp, err := h.service.JoinQueue(r.Context(), appmiddleware.SessionFromContext(r.Context()), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.JSON(w, r, http.StatusAccepted, resp)
}

// LeaveQueue handles DELETE /matchmaking/queue.
func (h *Handler) LeaveQueue(w http.ResponseWriter, r *http.Request) {
	if err := h.service.LeaveQueue(r.Context(), appmiddleware.SessionFromContext(r.Context())); err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.NoContent(w)
}

// GetStatus handles GET /matchmaking/status.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetStatus(r.Context(), appmiddleware.SessionFromContext(r.Context()))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	apphttp.OK(w, r, resp)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	apphttp.Error(w, r, h.logger, MapError(err))
}
