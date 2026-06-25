package app

import (
	"log/slog"
	"net/http"

	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

func NewServer(cfg config.Config, logger *slog.Logger, obs *observability.Observability, healthHandler *health.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      NewRouter(cfg, logger, obs, healthHandler),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
