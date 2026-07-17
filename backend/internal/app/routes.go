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
	"github.com/raven/geoguess/backend/internal/friends"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/home"
	"github.com/raven/geoguess/backend/internal/leaderboards"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/profiles"
	"github.com/raven/geoguess/backend/internal/realtime"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/uploads"
)

func NewRouter(cfg config.Config, logger *slog.Logger, obs *observability.Observability, rateLimiter appmiddleware.RateLimiter, healthHandler *health.Handler, authHandler *auth.Handler, profilesHandler *profiles.Handler, uploadsHandler *uploads.Handler, mapsHandler *maps.Handler, locationsHandler *locations.Handler, gamesHandler *games.Handler, challengesHandler *challenges.Handler, leaderboardsHandler *leaderboards.Handler, roomsHandler *rooms.Handler, realtimeHandler *realtime.Handler, matchmakingHandler *matchmaking.Handler, friendsHandler *friends.Handler, homeHandler *home.Handler) http.Handler {
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

		if profilesHandler != nil {
			profileUpdateLimit := appmiddleware.RateLimitConfig{Limit: 10, Window: 1 * time.Minute}
			api.Group(func(p chi.Router) {
				p.Get("/profile", profilesHandler.GetCurrentProfile)
				p.Get("/users/{userId}/stats", profilesHandler.GetPublicProfile)
				p.Get("/users/{userId}/games", profilesHandler.GetGameHistory)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, profileUpdateLimit, appmiddleware.RateLimitByCookie("profile-update", auth.AccessTokenCookieName), logger, profilesHandler.RecordRateLimited),
				).Patch("/profile", profilesHandler.UpdateProfile)
			})
		}

		if homeHandler != nil {
			homeReadLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.With(
				appmiddleware.RequireAuth(logger),
				appmiddleware.RateLimitWithObserver(rateLimiter, homeReadLimit, appmiddleware.RateLimitByRegisteredUser("home-read"), logger, homeHandler.RecordRateLimited),
			).Get("/home", homeHandler.Get)
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
				g.With(appmiddleware.RateLimit(rateLimiter, guessLimit, appmiddleware.RateLimitByIP("guess-timeout"), logger)).Post("/{gameId}/rounds/{roundId}/timeout", gamesHandler.ExpireRound)
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

		if leaderboardsHandler != nil {
			leaderboardLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.With(appmiddleware.RateLimit(rateLimiter, leaderboardLimit, appmiddleware.RateLimitByIP("leaderboards"), logger)).
				Group(func(l chi.Router) {
					leaderboardsHandler.RegisterRoutes(l)
				})
			api.With(
				appmiddleware.RequireAuth(logger),
				appmiddleware.RateLimitWithObserver(rateLimiter, leaderboardLimit, appmiddleware.RateLimitByRegisteredUser("lb-friends"), logger, leaderboardsHandler.RecordRateLimited),
			).Get("/leaderboards/friends", leaderboardsHandler.GetFriends)
		}

		if roomsHandler != nil {
			roomLimit := appmiddleware.RateLimitConfig{Limit: 60, Window: 1 * time.Minute}
			api.With(appmiddleware.RateLimit(rateLimiter, roomLimit, appmiddleware.RateLimitByIP("rooms"), logger)).
				Group(func(r chi.Router) {
					roomsHandler.RegisterRoutes(r)
				})
		}

		if matchmakingHandler != nil {
			// Command budget is intentionally lower than status polling budget.
			commandLimit := appmiddleware.RateLimitConfig{Limit: 20, Window: 1 * time.Minute}
			statusLimit := appmiddleware.RateLimitConfig{Limit: 60, Window: 1 * time.Minute}
			api.Group(func(m chi.Router) {
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, commandLimit, appmiddleware.RateLimitByRegisteredUser("mm-cmd"), logger, matchmakingHandler.RecordCommandRateLimited),
				).Post("/matchmaking/queue", matchmakingHandler.JoinQueue)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, commandLimit, appmiddleware.RateLimitByRegisteredUser("mm-cmd"), logger, matchmakingHandler.RecordCommandRateLimited),
				).Delete("/matchmaking/queue", matchmakingHandler.LeaveQueue)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, statusLimit, appmiddleware.RateLimitByRegisteredUser("mm-status"), logger, matchmakingHandler.RecordStatusRateLimited),
				).Get("/matchmaking/status", matchmakingHandler.GetStatus)
			})
		}

		if friendsHandler != nil {
			requestLimit := appmiddleware.RateLimitConfig{Limit: 10, Window: 1 * time.Minute}
			actionLimit := appmiddleware.RateLimitConfig{Limit: 30, Window: 1 * time.Minute}
			readLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.Group(func(f chi.Router) {
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, requestLimit, appmiddleware.RateLimitByRegisteredUser("friends-req"), logger, friendsHandler.RecordRateLimited),
				).Post("/friends/requests", friendsHandler.CreateRequest)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, readLimit, appmiddleware.RateLimitByRegisteredUser("friends-read"), logger, friendsHandler.RecordRateLimited),
				).Get("/friends/requests/incoming", friendsHandler.ListIncoming)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, readLimit, appmiddleware.RateLimitByRegisteredUser("friends-read"), logger, friendsHandler.RecordRateLimited),
				).Get("/friends/requests/outgoing", friendsHandler.ListOutgoing)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, actionLimit, appmiddleware.RateLimitByRegisteredUser("friends-action"), logger, friendsHandler.RecordRateLimited),
				).Post("/friends/requests/{requestId}/accept", friendsHandler.AcceptRequest)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, actionLimit, appmiddleware.RateLimitByRegisteredUser("friends-action"), logger, friendsHandler.RecordRateLimited),
				).Post("/friends/requests/{requestId}/decline", friendsHandler.DeclineRequest)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, readLimit, appmiddleware.RateLimitByRegisteredUser("friends-read"), logger, friendsHandler.RecordRateLimited),
				).Get("/friends", friendsHandler.ListFriends)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, actionLimit, appmiddleware.RateLimitByRegisteredUser("friends-action"), logger, friendsHandler.RecordRateLimited),
				).Delete("/friends/{userId}", friendsHandler.RemoveFriend)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, actionLimit, appmiddleware.RateLimitByRegisteredUser("friends-action"), logger, friendsHandler.RecordRateLimited),
				).Post("/friends/{userId}/block", friendsHandler.BlockUser)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, actionLimit, appmiddleware.RateLimitByRegisteredUser("friends-action"), logger, friendsHandler.RecordRateLimited),
				).Delete("/friends/{userId}/block", friendsHandler.UnblockUser)
				f.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, readLimit, appmiddleware.RateLimitByRegisteredUser("friends-read"), logger, friendsHandler.RecordRateLimited),
				).Get("/friends/blocked", friendsHandler.ListBlocked)
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
