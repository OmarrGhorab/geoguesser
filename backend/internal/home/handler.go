package home

import (
	"context"
	"log/slog"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

type reader interface {
	Get(ctx context.Context, sess *session.Context) (*Response, error)
}

// Handler serves the authenticated-home read model.
type Handler struct {
	service reader
	logger  *slog.Logger
	metrics *Metrics
}

// NewHandler returns an authenticated-home handler.
func NewHandler(service reader, logger *slog.Logger, metrics *Metrics) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, logger: logger, metrics: metrics}
}

// Get handles GET /home.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.Get(r.Context(), appmiddleware.SessionFromContext(r.Context()))
	if err != nil {
		apphttp.Error(w, r, h.logger, ToAPIError(err))
		return
	}
	apphttp.OK(w, r, response)
}

// RecordRateLimited records and logs a route rejection without session data.
func (h *Handler) RecordRateLimited(r *http.Request) {
	h.metrics.RecordRateLimited()
	h.logger.InfoContext(r.Context(), "authenticated home read rate limited", slog.String("path", r.URL.Path))
}
