package games

import (
	"context"
	"log/slog"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// RankedDeadlineSweeper is the narrow worker dependency.
type RankedDeadlineSweeper interface {
	SweepExpiredRankedRounds(ctx context.Context, limit int) error
}

// RankedDeadlineWorkerConfig configures the bounded deadline runner.
type RankedDeadlineWorkerConfig struct {
	Interval  time.Duration
	Timeout   time.Duration
	BatchSize int
}

// NewRankedDeadlineRunner advances expired ranked rounds without client traffic.
func NewRankedDeadlineRunner(sweeper RankedDeadlineSweeper, cfg RankedDeadlineWorkerConfig, logger *slog.Logger) *workers.Runner {
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
		return sweeper.SweepExpiredRankedRounds(ctx, cfg.BatchSize)
	}, workers.WithName("ranked-round-deadlines"), workers.WithLogger(logger))
}
