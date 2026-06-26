package app

import (
	"log/slog"
	"net/http"

	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/uploads"
	"github.com/raven/geoguess/backend/internal/users"
)

func NewServer(cfg config.Config, logger *slog.Logger, obs *observability.Observability, rateLimiter appmiddleware.RateLimiter, healthHandler *health.Handler, authHandler *auth.Handler, usersHandler *users.Handler, uploadsHandler *uploads.Handler, mapsHandler *maps.Handler, locationsHandler *locations.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      NewRouter(cfg, logger, obs, rateLimiter, healthHandler, authHandler, usersHandler, uploadsHandler, mapsHandler, locationsHandler),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
