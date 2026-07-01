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
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/health"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
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
	roomsRepo := rooms.NewRepository(db)
	roomCoordinator := redisplatform.NewRoomCoordinator(redisClient)

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
	profilesService := profiles.NewService(profilesRepo, profilesMetrics)
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
	gamesService := games.NewServiceWithHook(gamesRepo, mapsService, locations.StaticProvider{}, clock.NewSystem(), logger, games.NewRedisIdempotencyStore(redisClient), obs.Metrics, challengesService)

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

	uploadsService := uploads.NewService(uploadsRepo, storageProvider, cfg)

	authHandler := auth.NewHandler(authService, cfg, logger)
	profilesHandler := profiles.NewHandler(profilesService, logger)
	uploadsHandler := uploads.NewHandler(uploadsService, logger)
	mapsHandler := maps.NewHandler(mapsService, logger)
	locationsHandler := locations.NewHandler(locationsService, logger)
	gamesHandler := games.NewHandler(gamesService, logger)
	challengesHandler := challenges.NewHandler(challengesService, logger)
	roomsService := rooms.NewServiceWithGames(roomsRepo, roomCoordinator, gamesService, logger, nil)
	roomsHandler := rooms.NewHandler(roomsService, logger)
	realtimeHandler := realtime.NewHandler(realtime.NewHub(), roomsService, logger, nil)

	server := app.NewServer(cfg, logger, obs, redisplatform.NewRateLimiter(redisClient), healthHandler, authHandler, profilesHandler, uploadsHandler, mapsHandler, locationsHandler, gamesHandler, challengesHandler, roomsHandler, realtimeHandler)

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

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("api server stopped")
}
