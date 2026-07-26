package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/app"
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
	"github.com/raven/geoguess/backend/internal/parties"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/platform/email"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
	"github.com/raven/geoguess/backend/internal/platform/storage"
	"github.com/raven/geoguess/backend/internal/profiles"
	"github.com/raven/geoguess/backend/internal/realtime"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/uploads"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check whether the process can start")
	flag.Parse()

	if *healthcheck {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Default().Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	obs, err := observability.New("geoguess-api", cfg.Version)
	if err != nil {
		slog.Default().Error("failed to initialize observability", slog.Any("error", err))
		os.Exit(1)
	}
	logger := obs.Logger

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}

	redisClient, err := redisplatform.Open(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Error("failed to close redis client", slog.Any("error", closeErr))
		}
	}()

	healthHandler := health.NewHandlerWithObservability(cfg.Version, logger, obs, health.NewDefaultPingers(db, redisClient))

	authRepo := auth.NewRepository(db)
	profilesRepo := profiles.NewRepository(db)
	uploadsRepo := uploads.NewRepository(db)
	mapsRepo := maps.NewRepository(db)
	locationsRepo := locations.NewRepository(db)
	gamesRepo := games.NewRepository(db)
	challengesRepo := challenges.NewRepository(db)
	leaderboardsRepo := leaderboards.NewRepository(db)
	roomsRepo := rooms.NewRepository(db)
	roomCoordinator := redisplatform.NewRoomCoordinator(redisClient)
	friendsRepo := friends.NewRepository(db)

	// Shared Redis infrastructure for matchmaking v2, parties, and matchplay.
	v1Coord := redisplatform.NewMatchmakingCoordinator(redisClient)
	v2Coord := redisplatform.NewMatchmakingV2Coordinator(redisClient)
	matchmakingQueue := matchmaking.NewRedisQueueAdapterWithV2(v1Coord, v2Coord)
	cmdIdem := redisplatform.NewCommandIdempotencyStore(redisClient)
	realtimeStore := redisplatform.NewRealtimeStore(redisClient)
	hub := realtime.NewHubWithQueueSize(cfg.RealtimeOutboundQueueSize)
	var channelPub realtime.ChannelPublisher

	hasher := auth.NewBCryptHasher()
	tokenManager, err := auth.NewTokenManager(cfg.AccessTokenSecret, cfg.AccessTokenTTL)
	if err != nil {
		logger.Error("failed to create token manager", slog.Any("error", err))
		os.Exit(1)
	}
	guestManager, err := auth.NewGuestSessionManager(cfg.GuestSessionSecret)
	if err != nil {
		logger.Error("failed to create guest session manager", slog.Any("error", err))
		os.Exit(1)
	}
	csrfManager, err := auth.NewCSRFManager(cfg.CSRFSecret)
	if err != nil {
		logger.Error("failed to create csrf manager", slog.Any("error", err))
		os.Exit(1)
	}
	oauthManager := auth.NewOAuthManager(cfg)

	otpStore := auth.NewOTPStore(redisClient, cfg.OTPTTL)
	sessionStore := auth.NewRedisSessionStore(redisClient)
	var emailSender email.Sender
	switch strings.ToLower(cfg.EmailProvider) {
	case "resend":
		if cfg.ResendAPIKey == "" {
			logger.Error("RESEND_API_KEY is required when EMAIL_PROVIDER=resend")
			os.Exit(1)
		}
		emailSender = email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom)
	default:
		emailSender = email.NewLoggerSender(logger)
	}

	authService := auth.NewService(authRepo, hasher, tokenManager, guestManager, csrfManager, oauthManager, sessionStore, otpStore, emailSender, redisClient, cfg, clock.NewSystem())
	profilesMetrics, err := profiles.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register profiles metrics", slog.Any("error", err))
		os.Exit(1)
	}
	profilesService := profiles.NewServiceWithLogger(profilesRepo, profilesMetrics, logger)
	mapsService := maps.NewService(mapsRepo)
	locationsService := locations.NewService(locationsRepo, locations.StaticProvider{})
	var defaultChallengeMapID uuid.UUID
	if cfg.ChallengeDefaultMapID != "" {
		parsed, parseErr := uuid.Parse(cfg.ChallengeDefaultMapID)
		if parseErr != nil {
			logger.Error("failed to parse CHALLENGE_DEFAULT_MAP_ID", slog.Any("error", parseErr))
			os.Exit(1)
		}
		defaultChallengeMapID = parsed
	}
	challengesService := challenges.NewServiceWithIdempotency(challengesRepo, mapsService, clock.NewSystem(), logger, cfg.ChallengeResetHourUTC, defaultChallengeMapID, obs.Metrics, challenges.NewRedisIdempotencyStore(redisClient))
	leaderboardsMetrics, err := leaderboards.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register leaderboards metrics", slog.Any("error", err))
		os.Exit(1)
	}
	leaderboardsService := leaderboards.NewService(leaderboardsRepo, leaderboards.NewRedisPageCache(redisClient), clock.NewSystem(), logger, cfg.ChallengeResetHourUTC, challengesService).
		WithMetrics(leaderboardsMetrics)
	gamesMetrics, err := games.NewPrometheusMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register games metrics", slog.Any("error", err))
		os.Exit(1)
	}
	gamesService := games.NewServiceWithHook(gamesRepo, mapsService, locations.StaticProvider{}, clock.NewSystem(), logger, games.NewRedisIdempotencyStore(redisClient), gamesMetrics, leaderboardsService).
		WithPracticeCursorSigningSecret(cfg.GuestSessionSecret)
	// Quick Play map: config.Load already resolved the fallback chain
	// (QUICK_PLAY > CHALLENGE > MATCHMAKING) and Validate checked the UUID.
	if cfg.QuickPlayDefaultMapID != "" {
		parsed, parseErr := uuid.Parse(cfg.QuickPlayDefaultMapID)
		if parseErr != nil {
			logger.Error("failed to parse Quick Play default map id", slog.Any("error", parseErr))
			os.Exit(1)
		}
		gamesService.WithQuickPlayDefaults(parsed, cfg.QuickPlayRoundCount, cfg.QuickPlayTimerSeconds)
	} else {
		logger.Warn("QUICK_PLAY_DEFAULT_MAP_ID unset; POST /games/quick-play returns 503 until configured")
	}
	// Ranked lifecycle adapter is wired after matchmakingRepo is constructed below.

	var storageProvider storage.Provider
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" && cfg.R2Bucket != "" {
		storageProvider, err = storage.NewR2Provider(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2Endpoint, cfg.R2PublicURL)
		if err != nil {
			logger.Error("failed to create R2 provider", slog.Any("error", err))
			os.Exit(1)
		}
	} else {
		logger.Info("R2 not configured, using local storage provider")
		storageProvider, err = storage.NewLocalProvider("./tmp/uploads")
		if err != nil {
			logger.Error("failed to create local storage provider", slog.Any("error", err))
			os.Exit(1)
		}
	}

	uploadsMetrics, err := uploads.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register uploads metrics", slog.Any("error", err))
		os.Exit(1)
	}
	uploadsService := uploads.NewService(uploadsRepo, storageProvider, cfg).WithMetrics(uploadsMetrics)

	authHandler := auth.NewHandler(authService, cfg, logger)
	profilesHandler := profiles.NewHandler(profilesService, logger)
	uploadsHandler := uploads.NewHandler(uploadsService, logger)
	mapsHandler := maps.NewHandler(mapsService, logger)
	locationsHandler := locations.NewHandler(locationsService, logger)
	gamesHandler := games.NewHandler(gamesService, logger)
	challengesHandler := challenges.NewHandler(challengesService, logger)
	leaderboardsHandler := leaderboards.NewHandler(leaderboardsService, logger).WithMetrics(leaderboardsMetrics)
	roomsService := rooms.NewServiceWithGames(roomsRepo, roomCoordinator, gamesService, logger, nil)
	gamesService.WithHostedRoomLifecycle(roomsRepo)
	roomsHandler := rooms.NewHandler(roomsService, logger)

	// Realtime metrics + multi-instance Pub/Sub fanout for party/match channels.
	realtimeMetrics, err := realtime.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register realtime metrics", slog.Any("error", err))
		os.Exit(1)
	}
	var fanoutPub *realtime.FanoutPublisher
	realtimePubSub := redisplatform.NewRealtimePubSub(redisClient, func(kind, id string, payload []byte) {
		if fanoutPub != nil {
			fanoutPub.HandlePubSubMessage(kind, id, payload)
		}
	}).WithLogger(logger)
	fanoutPub = realtime.NewFanoutPublisher(hub, realtimePubSub, logger)
	channelPub = fanoutPub

	// Share one hub between room realtime and matchplay event fanout.
	realtimeHandler := realtime.NewHandler(hub, roomsService, logger, realtimeMetrics)

	friendsMetrics, err := friends.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register friends metrics", slog.Any("error", err))
		os.Exit(1)
	}
	friendsService := friends.NewServiceWithLogger(friendsRepo, friendsMetrics, logger)
	friendsHandler := friends.NewHandler(friendsService, logger)

	// Parties depend on friends policy; construct after friendsRepo.
	partiesMetrics, err := parties.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register parties metrics", slog.Any("error", err))
		os.Exit(1)
	}
	partiesRepo := parties.NewRepository(db)
	partyPolicy := friends.NewPartyPolicy(friendsRepo)
	// Enable party mutations when either casual or ranked team modes are staged on.
	partyFeatureEnabled := cfg.CasualMatchmakingEnabled || cfg.RankedTeamModesEnabled
	partiesService := parties.NewServiceWithOptions(
		partiesRepo,
		partyPolicy,
		partiesMetrics,
		app.NewPartyIdempotencyAdapter(cmdIdem),
		parties.Config{
			CasualEnabled: partyFeatureEnabled,
			InviteTTL:     cfg.PartyInviteTTL,
		},
		logger,
	).WithEvents(parties.NewEventPublisher(channelPub, logger))
	partiesHandler := parties.NewHandler(partiesService, logger)

	matchmakingMetrics, err := matchmaking.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register matchmaking metrics", slog.Any("error", err))
		os.Exit(1)
	}
	matchmakingRepo := matchmaking.NewRepository(db)
	var defaultMapID uuid.UUID
	if cfg.MatchmakingDefaultMapID != "" {
		parsed, parseErr := uuid.Parse(cfg.MatchmakingDefaultMapID)
		if parseErr != nil {
			logger.Error("failed to parse MATCHMAKING_DEFAULT_MAP_ID", slog.Any("error", parseErr))
			os.Exit(1)
		}
		defaultMapID = parsed
	}
	// Competitive standings + Elo finalization (US2). Progression is applied by the
	// background retry worker so match completion stays durable even if rating writes fail.
	competitiveMetrics, err := competitive.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register competitive metrics", slog.Any("error", err))
		os.Exit(1)
	}
	competitiveRepo := competitive.NewRepository(db)
	competitiveService := competitive.NewService(competitiveRepo, competitive.Config{
		EloK:             cfg.CompetitiveEloK,
		AbandonPenalty:   cfg.CompetitiveAbandonPenalty,
		InitialRating:    cfg.CompetitiveInitialRating,
		WorkerBatchSize:  50,
		SeasonDuration:   time.Duration(cfg.CompetitiveSeasonDurationDays) * 24 * time.Hour,
		ResetFactorBPS:   cfg.CompetitiveResetFactorBPS,
		Top500MinMatches: cfg.CompetitiveTop500MinMatches,
	}, logger).
		WithMetrics(competitiveMetrics).
		WithCache(competitive.NewRedisPageCache(redisClient)).
		WithRolloverConfig(competitive.RolloverConfig{
			Duration:         time.Duration(cfg.CompetitiveSeasonDurationDays) * 24 * time.Hour,
			InitialRating:    cfg.CompetitiveInitialRating,
			ResetFactorBPS:   cfg.CompetitiveResetFactorBPS,
			Top500MinMatches: cfg.CompetitiveTop500MinMatches,
			EloK:             cfg.CompetitiveEloK,
		})
	competitiveHandler := competitive.NewHandler(competitiveService, logger).
		WithFeatureEnabled(cfg.RankedTeamModesEnabled)

	matchLifecycle := matchmaking.NewMatchLifecycleAdapter(matchmakingRepo).
		WithLogger(logger).
		WithMetrics(matchmakingMetrics)
	gamesService.WithRankedLifecycle(matchLifecycle)
	gamesService.WithMatchLifecycle(matchLifecycle)

	matchmakingEvents := matchmaking.NewEventPublisherAdapter(channelPub, logger)
	matchmakingService := matchmaking.NewService(matchmakingRepo, matchmakingQueue, matchmaking.Config{
		DefaultMapID:             defaultMapID,
		QueueLease:               cfg.MatchmakingQueueLease,
		ClaimTTL:                 cfg.MatchmakingClaimTTL,
		StartDelay:               cfg.MatchmakingStartDelay,
		RoundCount:               cfg.MatchmakingRoundCount,
		TimerSeconds:             cfg.MatchmakingTimerSeconds,
		CandidateScanLimit:       cfg.MatchmakingCandidateScanLimit,
		CasualMatchmakingEnabled: cfg.CasualMatchmakingEnabled,
		RankedTeamModesEnabled:   cfg.RankedTeamModesEnabled,
	}, logger, matchmakingMetrics).
		WithLocations(matchmaking.NewMapsLocationSelector(mapsService)).
		WithTickets(matchmakingQueue).
		WithParties(partiesService).
		WithCompetitive(competitiveService).
		WithNotifier(matchmakingEvents).
		WithEvents(matchmakingEvents)
	matchmakingHandler := matchmaking.NewHandlerWithMetrics(matchmakingService, logger, matchmakingMetrics)

	matchplayMetrics, err := matchplay.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register matchplay metrics", slog.Any("error", err))
		os.Exit(1)
	}
	matchplayRepo := matchplay.NewRepository(db)
	matchLiveStore := redisplatform.NewMatchLiveStore(redisClient)
	eventPublisher := matchplay.NewPublisher(channelPub, logger).WithMetrics(matchplayMetrics)
	gamesService.WithMultiplayerEvents(games.MultiplayerEventFanout{
		roomsService,
		app.NewGameOutcomeEventAdapter(matchplayRepo, eventPublisher, realtimeStore),
	})
	matchplayService := matchplay.NewService(matchplayRepo, logger, matchplayMetrics, matchplay.ServiceConfig{
		ReconnectGrace:   cfg.MatchReconnectGrace,
		CasualInactivity: cfg.CasualInactivity,
	}).
		WithPartyRestorer(partiesService).
		WithVersionStore(app.NewMatchVersionAdapter(realtimeStore)).
		WithPresence(app.NewMatchPresenceAdapter(realtimeStore)).
		WithEvents(eventPublisher).
		WithChatStore(matchplayRepo).
		WithChatConfig(matchplay.ChatServiceConfig{
			TeamChatImagesEnabled: cfg.TeamChatImagesEnabled,
			RetentionDays:         cfg.TeamChatRetentionDays,
			ReportRetentionDays:   cfg.TeamChatReportRetentionDays,
		}).
		WithAttachmentSigner(app.NewAttachmentSignerAdapter(storageProvider))
	// Team-chat upload authorization against match participants.
	uploadsService.WithMatchAccess(app.NewMatchUploadAccessAdapter(matchplayService))
	matchplayHandler := matchplay.NewHandlerWithMetrics(matchplayService, logger, matchplayMetrics)

	// Realtime ticket + party/match WebSocket transport (feature-gated with team modes).
	realtimeFeatureEnabled := cfg.CasualMatchmakingEnabled || cfg.RankedTeamModesEnabled
	ticketStore := app.NewRealtimeTicketStore(realtimeStore)
	channelServices := app.RealtimeChannelServices{
		Matchplay: matchplayService,
		Parties:   partiesService,
		Live:      matchLiveStore,
		Metrics:   matchplayMetrics,
	}
	ticketHandler := realtime.NewTicketHandler(
		ticketStore,
		channelServices,
		realtime.TicketHandlerConfig{
			TTL:            cfg.RealtimeTicketTTL,
			FeatureEnabled: realtimeFeatureEnabled,
		},
		logger,
		realtimeMetrics,
	)
	liveCommands := app.NewLiveCommandAdapter(matchLiveStore, matchplayMetrics).WithMatchplay(matchplayService)
	matchWSHandler := realtime.NewMatchHandler(
		hub,
		ticketStore,
		channelServices,
		channelServices,
		liveCommands,
		fanoutPub,
		realtime.MatchHandlerConfig{
			AllowedOrigins: cfg.RealtimeAllowedOrigins,
			QueueSize:      cfg.RealtimeOutboundQueueSize,
		},
		logger,
		realtimeMetrics,
	).WithConnectionLifecycle(app.NewRealtimeConnectionLifecycle(realtimeStore, matchplayService, cfg.MatchReconnectGrace))
	realtimeHandler = realtimeHandler.WithTicket(ticketHandler).WithMatch(matchWSHandler)

	// Background workers: match lifecycle sweeps, expired claim recovery, rating finalization, chat cleanup.
	lifecycleWorker := matchplay.NewLifecycleRunner(matchplayService, matchplay.LifecycleWorkerConfig{
		Interval: cfg.MatchSweepInterval,
	}, logger)
	deadlineWorker := games.NewTimedMultiplayerDeadlineRunner(gamesService, games.TimedMultiplayerDeadlineWorkerConfig{
		Interval:  cfg.MatchSweepInterval,
		BatchSize: 50,
	}, logger)
	claimWorker := matchmaking.NewClaimSweepRunner(matchmakingService, matchmaking.ClaimSweepConfig{
		Interval: cfg.MatchClaimSweepInterval,
	}, logger)
	progressionWorker := competitive.NewProgressionRetryRunner(competitiveService, competitive.ProgressionWorkerConfig{
		Interval: cfg.MatchSweepInterval,
	}, logger)
	rolloverObs := &competitive.RolloverWorkerObservations{}
	rolloverWorker := competitive.NewSeasonRolloverRunner(competitiveService, competitive.RolloverWorkerConfig{
		Interval: time.Minute,
	}, logger, rolloverObs)
	chatCleaner := matchplay.NewCleaner(
		matchplayRepo,
		app.NewObjectDeleterAdapter(storageProvider),
		logger,
		matchplayMetrics,
		matchplay.CleanupConfig{},
	)
	cleanupWorker := matchplay.NewCleanupRunner(chatCleaner, matchplay.CleanupWorkerConfig{
		Interval: cfg.TeamChatCleanupInterval,
	}, logger)
	lifecycleWorker.Start(ctx)
	deadlineWorker.Start(ctx)
	claimWorker.Start(ctx)
	progressionWorker.Start(ctx)
	rolloverWorker.Start(ctx)
	cleanupWorker.Start(ctx)

	homeMetrics, err := home.NewMetrics(obs.Metrics.Registry())
	if err != nil {
		logger.Error("failed to register authenticated home metrics", slog.Any("error", err))
		os.Exit(1)
	}
	homeService := home.NewService(profilesRepo, challengesService, mapsService, logger, homeMetrics)
	homeHandler := home.NewHandler(homeService, logger, homeMetrics)

	server := app.NewServer(cfg, logger, obs, redisplatform.NewRateLimiter(redisClient), healthHandler, authHandler, profilesHandler, uploadsHandler, mapsHandler, locationsHandler, gamesHandler, challengesHandler, leaderboardsHandler, roomsHandler, realtimeHandler, matchmakingHandler, friendsHandler, homeHandler, partiesHandler, matchplayHandler, competitiveHandler)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server listening", slog.String("addr", cfg.HTTPAddr))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop background workers before/around HTTP drain so ticks do not race shutdown.
	lifecycleWorker.Stop()
	deadlineWorker.Stop()
	claimWorker.Stop()
	progressionWorker.Stop()
	rolloverWorker.Stop()
	cleanupWorker.Stop()
	_ = realtimePubSub.Shutdown(shutdownCtx)

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("api server stopped")
}
