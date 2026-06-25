package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/health"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/uploads"
	"github.com/raven/geoguess/backend/internal/users"
)

func NewRouter(cfg config.Config, logger *slog.Logger, obs *observability.Observability, rateLimiter appmiddleware.RateLimiter, healthHandler *health.Handler, authHandler *auth.Handler, usersHandler *users.Handler, uploadsHandler *uploads.Handler) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(appmiddleware.RequestLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(cfg.WriteTimeout))
	router.Use(appmiddleware.SecurityHeaders)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedOrigin},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	router.Use(appmiddleware.Metrics(obs.Metrics))

	cookieOpts := auth.NewCookieOptions(cfg)

	if authHandler != nil {
		authService := authHandler.Service()
		router.Use(appmiddleware.SessionLoader(authService, auth.AccessTokenCookieName, auth.GuestSessionCookieName))
		router.Use(appmiddleware.CSRF(authService, cookieOpts, logger))
	}

	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthHandler.Health)
		api.Get("/ready", healthHandler.Ready)
		api.With(appmiddleware.MetricsAuth(cfg.MetricsAuthToken)).Get("/metrics", healthHandler.Metrics)

		if authHandler != nil {
			authRateLimit := appmiddleware.RateLimitConfig{Limit: 10, Window: 1 * time.Minute}
			api.With(appmiddleware.RateLimit(rateLimiter, authRateLimit, appmiddleware.RateLimitByIP("auth"), logger)).
				Group(func(a chi.Router) {
					authHandler.RegisterRoutes(a)
				})
		}

		if usersHandler != nil {
			usersHandler.RegisterRoutes(api)
		}

		if uploadsHandler != nil {
			api.Route("/uploads", func(u chi.Router) {
				uploadsHandler.RegisterUploadRoutes(u)
			})
			api.Route("/files", func(u chi.Router) {
				uploadsHandler.RegisterFileRoutes(u)
			})
		}
	})

	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)
	router.With(appmiddleware.MetricsAuth(cfg.MetricsAuthToken)).Get("/metrics", healthHandler.Metrics)

	return router
}
