package competitive

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// ProgressionWorkerConfig configures the bounded progression retry worker.
type ProgressionWorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// ProgressionRetrier is the service surface used by the progression worker.
type ProgressionRetrier interface {
	RetryPendingFinalizations(ctx context.Context) (processed, applied, failed int, err error)
}

// SeasonRoller is the service surface used by the season rollover worker.
type SeasonRoller interface {
	RollOverIfDue(ctx context.Context) (*RolloverOutcome, error)
}

// NewProgressionRetryRunner builds a workers.Runner that periodically finalizes
// completed Ranked matches lacking progression_finalized_at.
func NewProgressionRetryRunner(svc ProgressionRetrier, cfg ProgressionWorkerConfig, logger *slog.Logger) *workers.Runner {
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
		processed, applied, failed, err := svc.RetryPendingFinalizations(ctx)
		if err != nil {
			logger.Warn("competitive progression worker tick failed",
				"error", err.Error(),
			)
			return err
		}
		if processed > 0 {
			logger.Info("competitive progression worker tick",
				"processed", processed,
				"applied", applied,
				"failed", failed,
			)
		}
		return nil
	}, workers.WithName("competitive-progression-retry"), workers.WithLogger(logger))
}

// RolloverWorkerConfig configures the minute-based season rollover worker.
type RolloverWorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// RolloverWorkerObservations tracks last success/failure for operational readiness.
type RolloverWorkerObservations struct {
	mu            sync.RWMutex
	LastSuccessAt time.Time
	LastFailureAt time.Time
	LastOutcome   string
	LastError     string
}

// Snapshot returns a copy of the observation state.
func (o *RolloverWorkerObservations) Snapshot() (lastSuccess, lastFailure time.Time, outcome, lastErr string) {
	if o == nil {
		return time.Time{}, time.Time{}, "", ""
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.LastSuccessAt, o.LastFailureAt, o.LastOutcome, o.LastError
}

func (o *RolloverWorkerObservations) recordSuccess(outcome string) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.LastSuccessAt = time.Now().UTC()
	o.LastOutcome = outcome
	o.LastError = ""
}

func (o *RolloverWorkerObservations) recordFailure(err error) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.LastFailureAt = time.Now().UTC()
	o.LastOutcome = "error"
	if err != nil {
		o.LastError = err.Error()
	}
}

// NewSeasonRolloverRunner builds a workers.Runner that checks for season end
// approximately every minute and applies advisory-locked rollover when due.
func NewSeasonRolloverRunner(svc SeasonRoller, cfg RolloverWorkerConfig, logger *slog.Logger, obs *RolloverWorkerObservations) *workers.Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 50 * time.Second
	}
	return workers.NewRunner(cfg.Interval, cfg.Timeout, func(ctx context.Context) error {
		if svc == nil {
			return nil
		}
		out, err := svc.RollOverIfDue(ctx)
		if err != nil {
			if obs != nil {
				obs.recordFailure(err)
			}
			logger.Warn("competitive season rollover worker tick failed",
				"error", err.Error(),
			)
			return err
		}
		outcome := "skipped"
		if out != nil {
			switch {
			case out.Applied:
				outcome = "applied"
			case out.Replay:
				outcome = "replay"
			case out.Skipped:
				outcome = "skipped"
			}
		}
		if obs != nil {
			obs.recordSuccess(outcome)
		}
		if outcome == "applied" || outcome == "replay" {
			logger.Info("competitive season rollover worker tick",
				"outcome", outcome,
			)
		}
		return nil
	}, workers.WithName("competitive-season-rollover"), workers.WithLogger(logger))
}

// Ensure Service implements worker interfaces.
var _ ProgressionRetrier = (*Service)(nil)
var _ SeasonRoller = (*Service)(nil)
