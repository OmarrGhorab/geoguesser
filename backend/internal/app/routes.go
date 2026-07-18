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
	"github.com/raven/geoguess/backend/internal/competitive"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/friends"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/home"
	"github.com/raven/geoguess/backend/internal/leaderboards"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/matchplay"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/parties"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/profiles"
	"github.com/raven/geoguess/backend/internal/realtime"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/uploads"
)

func NewRouter(cfg config.Config, logger *slog.Logger, obs *observability.Observability, rateLimiter appmiddleware.RateLimiter, healthHandler *health.Handler, authHandler *auth.Handler, profilesHandler *profiles.Handler, uploadsHandler *uploads.Handler, mapsHandler *maps.Handler, locationsHandler *locations.Handler, gamesHandler *games.Handler, challengesHandler *challenges.Handler, leaderboardsHandler *leaderboards.Handler, roomsHandler *rooms.Handler, realtimeHandler *realtime.Handler, matchmakingHandler *matchmaking.Handler, friendsHandler *friends.Handler, homeHandler *home.Handler, partiesHandler *parties.Handler, matchplayHandler *matchplay.Handler, competitiveHandler *competitive.Handler) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(appmiddleware.RequestLogger(logger))
	router.Use(middleware.Recoverer)
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
		// HTTP request deadlines must not wrap long-lived WebSocket transports,
		// which are mounted outside this group under /realtime.
		api.Use(middleware.Timeout(cfg.WriteTimeout))
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
				g.With(appmiddleware.RateLimit(rateLimiter, guessLimit, appmiddleware.RateLimitByIP("practice-next"), logger)).Post("/{gameId}/rounds/next", gamesHandler.NextPracticeRound)
				g.Get("/{gameId}/rounds", gamesHandler.GetPracticeHistory)
				g.With(appmiddleware.RateLimit(rateLimiter, gameCreateLimit, appmiddleware.RateLimitByIP("practice-end"), logger)).Post("/{gameId}/end", gamesHandler.EndPractice)
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

		if partiesHandler != nil {
			// Command budget is intentionally lower than status/snapshot polling budget.
			partyCommandLimit := appmiddleware.RateLimitConfig{Limit: 20, Window: 1 * time.Minute}
			partyReadLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.Group(func(p chi.Router) {
				// Read routes
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyReadLimit, appmiddleware.RateLimitByRegisteredUser("parties-read"), logger, partiesHandler.RecordRateLimited),
				).Get("/parties/current", partiesHandler.GetCurrentParty)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyReadLimit, appmiddleware.RateLimitByRegisteredUser("parties-read"), logger, partiesHandler.RecordRateLimited),
				).Get("/parties/{partyId}", partiesHandler.GetParty)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyReadLimit, appmiddleware.RateLimitByRegisteredUser("parties-read"), logger, partiesHandler.RecordRateLimited),
				).Get("/party-invites", partiesHandler.ListInvites)

				// Command routes (CSRF already applied globally for unsafe methods)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Post("/parties", partiesHandler.CreateParty)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Post("/parties/{partyId}/invites", partiesHandler.CreateInvite)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Post("/party-invites/{inviteId}/accept", partiesHandler.AcceptInvite)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Post("/party-invites/{inviteId}/decline", partiesHandler.DeclineInvite)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Put("/parties/{partyId}/readiness/me", partiesHandler.SetReadiness)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Delete("/parties/{partyId}/members/me", partiesHandler.LeaveParty)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Delete("/parties/{partyId}/members/{userId}", partiesHandler.KickMember)
				p.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, partyCommandLimit, appmiddleware.RateLimitByRegisteredUser("parties-cmd"), logger, partiesHandler.RecordRateLimited),
				).Delete("/parties/{partyId}", partiesHandler.DisbandParty)
			})
		}

		if matchplayHandler != nil {
			matchReadLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			matchLeaveLimit := appmiddleware.RateLimitConfig{Limit: 20, Window: 1 * time.Minute}
			matchChatLimit := appmiddleware.RateLimitConfig{Limit: 60, Window: 1 * time.Minute}
			api.Group(func(m chi.Router) {
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchReadLimit, appmiddleware.RateLimitByRegisteredUser("match-read"), logger, matchplayHandler.RecordRateLimited),
				).Get("/matches/{matchId}", matchplayHandler.GetSnapshot)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchReadLimit, appmiddleware.RateLimitByRegisteredUser("match-read"), logger, matchplayHandler.RecordRateLimited),
				).Get("/matches/{matchId}/rounds/{roundId}/results", matchplayHandler.GetRoundResults)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchReadLimit, appmiddleware.RateLimitByRegisteredUser("match-read"), logger, matchplayHandler.RecordRateLimited),
				).Get("/matches/{matchId}/results", matchplayHandler.GetTerminalResult)
				// POST leave — CSRF enforced by global middleware.
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchLeaveLimit, appmiddleware.RateLimitByRegisteredUser("match-leave"), logger, matchplayHandler.RecordRateLimited),
				).Post("/matches/{matchId}/leave", matchplayHandler.Leave)

				// Team chat / mute / report (US3) — handlers no-op with unavailable when chat service nil.
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchChatLimit, appmiddleware.RateLimitByRegisteredUser("match-chat"), logger, matchplayHandler.RecordRateLimited),
				).Post("/matches/{matchId}/messages", matchplayHandler.SendMessage)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchReadLimit, appmiddleware.RateLimitByRegisteredUser("match-chat-read"), logger, matchplayHandler.RecordRateLimited),
				).Get("/matches/{matchId}/messages", matchplayHandler.ListMessages)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchReadLimit, appmiddleware.RateLimitByRegisteredUser("match-chat-read"), logger, matchplayHandler.RecordRateLimited),
				).Get("/matches/{matchId}/messages/{messageId}/attachment", matchplayHandler.GetAttachment)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchChatLimit, appmiddleware.RateLimitByRegisteredUser("match-chat"), logger, matchplayHandler.RecordRateLimited),
				).Put("/matches/{matchId}/mutes/{userId}", matchplayHandler.MuteTeammate)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchChatLimit, appmiddleware.RateLimitByRegisteredUser("match-chat"), logger, matchplayHandler.RecordRateLimited),
				).Delete("/matches/{matchId}/mutes/{userId}", matchplayHandler.UnmuteTeammate)
				m.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, matchChatLimit, appmiddleware.RateLimitByRegisteredUser("match-chat"), logger, matchplayHandler.RecordRateLimited),
				).Post("/matches/{matchId}/messages/{messageId}/report", matchplayHandler.ReportMessage)
			})
		}

		// One-time realtime tickets for party/match WebSocket upgrade.
		if realtimeHandler != nil && realtimeHandler.HasTicketIssuer() {
			ticketLimit := appmiddleware.RateLimitConfig{Limit: 30, Window: 1 * time.Minute}
			api.With(
				appmiddleware.RequireAuth(logger),
				appmiddleware.RateLimit(rateLimiter, ticketLimit, appmiddleware.RateLimitByRegisteredUser("realtime-ticket"), logger),
			).Post("/realtime/tickets", realtimeHandler.IssueTicket)
		}

		// Competitive profile, history, seasons, and top-500 leaderboard (US5).
		if competitiveHandler != nil {
			competitiveReadLimit := appmiddleware.RateLimitConfig{Limit: 120, Window: 1 * time.Minute}
			api.Group(func(c chi.Router) {
				c.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, competitiveReadLimit, appmiddleware.RateLimitByRegisteredUser("competitive-read"), logger, competitiveHandler.RecordRateLimited),
				).Get("/competitive/profile", competitiveHandler.GetProfile)
				c.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, competitiveReadLimit, appmiddleware.RateLimitByRegisteredUser("competitive-read"), logger, competitiveHandler.RecordRateLimited),
				).Get("/competitive/leaderboard", competitiveHandler.GetLeaderboard)
				c.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, competitiveReadLimit, appmiddleware.RateLimitByRegisteredUser("competitive-read"), logger, competitiveHandler.RecordRateLimited),
				).Get("/competitive/history", competitiveHandler.GetHistory)
				c.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, competitiveReadLimit, appmiddleware.RateLimitByRegisteredUser("competitive-read"), logger, competitiveHandler.RecordRateLimited),
				).Get("/competitive/seasons", competitiveHandler.ListSeasons)
				c.With(
					appmiddleware.RequireAuth(logger),
					appmiddleware.RateLimitWithObserver(rateLimiter, competitiveReadLimit, appmiddleware.RateLimitByRegisteredUser("competitive-read"), logger, competitiveHandler.RecordRateLimited),
				).Get("/competitive/seasons/{seasonId}/leaderboard", competitiveHandler.GetSeasonLeaderboard)
			})
		}
	})

	if realtimeHandler != nil {
		router.Get("/realtime/rooms/{roomCode}", realtimeHandler.Room)
		if realtimeHandler.HasMatchTransport() {
			router.Get("/realtime/matches/{matchId}", realtimeHandler.MatchWS)
			router.Get("/realtime/parties/{partyId}", realtimeHandler.PartyWS)
		}
	}

	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)
	router.With(appmiddleware.MetricsAuth(cfg.MetricsAuthToken)).Get("/metrics", healthHandler.Metrics)

	return router
}
