package matchmaking_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/matchmaking"
)

type fakeClaimReconciler struct {
	calls atomic.Int64
}

func (f *fakeClaimReconciler) ReconcileExpiredClaims(context.Context) {
	f.calls.Add(1)
}

func TestClaimSweepRunnerTicks(t *testing.T) {
	t.Parallel()
	fake := &fakeClaimReconciler{}
	runner := matchmaking.NewClaimSweepRunner(fake, matchmaking.ClaimSweepConfig{
		Interval: 20 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fake.calls.Load() >= 1 && runner.SuccessCount() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	runner.Stop()

	if fake.calls.Load() < 1 {
		t.Fatalf("expected at least one reconcile call, got %d", fake.calls.Load())
	}
	if runner.SuccessCount() < 1 {
		t.Fatalf("expected runner success, got %d", runner.SuccessCount())
	}
}
