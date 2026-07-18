package games

import (
	"context"
	"log/slog"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// TimedMultiplayerDeadlineSweeper is the narrow worker dependency.
type TimedMultiplayerDeadlineSweeper interface {
	SweepExpiredTimedMultiplayerRounds(ctx context.Context, limit int) error
}

// TimedMultiplayerDeadlineWorkerConfig configures the bounded deadline runner.
type TimedMultiplayerDeadlineWorkerConfig struct {
	Interval  time.Duration
	Timeout   time.Duration
	BatchSize int
}

// NewTimedMultiplayerDeadlineRunner advances expired timed multiplayer rounds without client traffic.
func NewTimedMultiplayerDeadlineRunner(sweeper TimedMultiplayerDeadlineSweeper, cfg TimedMultiplayerDeadlineWorkerConfig, logger *slog.Logger) *workers.Runner {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = cfg.Interval
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	return workers.NewRunner(cfg.Interval, cfg.Timeout, func(ctx context.Context) error {
		if sweeper == nil {
			return nil
		}
		return sweeper.SweepExpiredTimedMultiplayerRounds(ctx, cfg.BatchSize)
	}, workers.WithName("timed-multiplayer-round-deadlines"), workers.WithLogger(logger))
}
