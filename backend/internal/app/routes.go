package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/challenges"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/realtime"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/uploads"
	"github.com/raven/geoguess/backend/internal/users"
)

func NewRouter(cfg config.Config, logger *slog.Logger, obs *observability.Observability, rateLimiter appmiddleware.RateLimiter, healthHandler *health.Handler, authHandler *auth.Handler, usersHandler *users.Handler, uploadsHandler *uploads.Handler, mapsHandler *maps.Handler, locationsHandler *locations.Handler, gamesHandler *games.Handler, challengesHandler *challenges.Handler, roomsHandler *rooms.Handler, realtimeHandler *realtime.Handler) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
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

		if mapsHandler != nil {
			mapsHandler.RegisterRoutes(api)
		}

		if locationsHandler != nil {
			locationsHandler.RegisterRoutes(api)
		}

		if gamesHandler != nil {
			gameCreateLimit := appmiddleware.RateLimitConfig{Limit: 20, Window: 1 * time.Minute}
			guessLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.Route("/games", func(g chi.Router) {
				g.With(appmiddleware.RateLimit(rateLimiter, gameCreateLimit, appmiddleware.RateLimitByIP("game-create"), logger)).Post("/", gamesHandler.CreateGame)
				g.Get("/{gameId}", gamesHandler.GetGame)
				g.Post("/{gameId}/start", gamesHandler.StartGame)
				g.Get("/{gameId}/rounds/current", gamesHandler.GetCurrentRound)
				g.With(appmiddleware.RateLimit(rateLimiter, guessLimit, appmiddleware.RateLimitByIP("guess"), logger)).Post("/{gameId}/rounds/{roundId}/guesses", gamesHandler.SubmitGuess)
				g.Get("/{gameId}/results", gamesHandler.GetResults)
			})
		}

		if challengesHandler != nil {
			challengeLimit := appmiddleware.RateLimitConfig{Limit: 60, Window: 1 * time.Minute}
			api.With(appmiddleware.RateLimit(rateLimiter, challengeLimit, appmiddleware.RateLimitByIP("challenges"), logger)).
				Group(func(c chi.Router) {
					challengesHandler.RegisterRoutes(c)
				})
		}

		if roomsHandler != nil {
			roomLimit := appmiddleware.RateLimitConfig{Limit: 60, Window: 1 * time.Minute}
			api.With(appmiddleware.RateLimit(rateLimiter, roomLimit, appmiddleware.RateLimitByIP("rooms"), logger)).
				Group(func(r chi.Router) {
					roomsHandler.RegisterRoutes(r)
				})
		}
	})

	if realtimeHandler != nil {
		router.Get("/realtime/rooms/{roomCode}", realtimeHandler.Room)
	}

	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)
	router.With(appmiddleware.MetricsAuth(cfg.MetricsAuthToken)).Get("/metrics", healthHandler.Metrics)

	return router
}
