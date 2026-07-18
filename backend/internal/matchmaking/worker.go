package matchmaking

import (
	"context"
	"log/slog"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// ClaimSweepConfig configures the expired-claim recovery worker.
type ClaimSweepConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// ClaimReconciler is the service surface used by the claim sweep worker.
// Implemented by *Service via ReconcileExpiredClaims.
type ClaimReconciler interface {
	ReconcileExpiredClaims(ctx context.Context)
}

// ReconcileExpiredClaims runs durable-first expired claim recovery.
// Exported for the bounded claim-sweep worker (request path also invokes it).
func (s *Service) ReconcileExpiredClaims(ctx context.Context) {
	if s == nil {
		return
	}
	s.reconcileExpiredClaims(ctx)
}

// NewClaimSweepRunner builds a workers.Runner that periodically recovers abandoned
// team claims without depending on user traffic.
func NewClaimSweepRunner(svc ClaimReconciler, cfg ClaimSweepConfig, logger *slog.Logger) *workers.Runner {
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
		svc.ReconcileExpiredClaims(ctx)
		return nil
	}, workers.WithName("matchmaking-claim-sweep"), workers.WithLogger(logger))
}
