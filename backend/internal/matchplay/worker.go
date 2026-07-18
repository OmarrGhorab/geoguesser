package matchplay

import (
	"context"
	"log/slog"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// LifecycleWorkerConfig configures the match lifecycle sweep runner.
type LifecycleWorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// NewLifecycleRunner builds a workers.Runner that periodically forfeits disconnects
// past reconnect grace and closes inactive casual matches.
// Progression-neutral: sweeps never write competitive rating rows.
func NewLifecycleRunner(svc *Service, cfg LifecycleWorkerConfig, logger *slog.Logger) *workers.Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 4 * time.Second
	}
	return workers.NewRunner(cfg.Interval, cfg.Timeout, func(ctx context.Context) error {
		if svc == nil {
			return nil
		}
		return svc.RunLifecycleSweep(ctx)
	}, workers.WithName("matchplay-lifecycle"), workers.WithLogger(logger))
}

// CleanupWorkerConfig configures the chat/raw retention cleanup runner.
type CleanupWorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// NewCleanupRunner builds a workers.Runner that periodically deletes expired chat
// and raw upload objects in bounded batches (legal-hold excluded by cleaner).
func NewCleanupRunner(cleaner *Cleaner, cfg CleanupWorkerConfig, logger *slog.Logger) *workers.Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 15 * time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Minute
	}
	return workers.NewRunner(cfg.Interval, cfg.Timeout, func(ctx context.Context) error {
		if cleaner == nil {
			return nil
		}
		_, err := cleaner.RunOnce(ctx)
		return err
	}, workers.WithName("matchplay-chat-cleanup"), workers.WithLogger(logger))
}
