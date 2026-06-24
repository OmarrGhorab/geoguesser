package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raven/geoguess/backend/internal/app"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check whether the process can start")
	flag.Parse()

	if *healthcheck {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

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

	server := app.NewServer(cfg, logger, db, redisClient)

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
