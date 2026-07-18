package workers

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerPanicIsolation(t *testing.T) {
	var calls atomic.Int64
	r := NewRunner(20*time.Millisecond, 50*time.Millisecond, func(ctx context.Context) error {
		n := calls.Add(1)
		if n == 1 {
			panic("boom")
		}
		return nil
	}, WithName("panic-test"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r.FailCount() >= 1 && r.SuccessCount() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	r.Stop()

	if r.FailCount() < 1 {
		t.Fatalf("expected at least one failure from panic, got %d", r.FailCount())
	}
	if r.SuccessCount() < 1 {
		t.Fatalf("expected runner to continue after panic; successes=%d calls=%d", r.SuccessCount(), calls.Load())
	}
	if r.LastFailureAt().IsZero() {
		t.Fatal("expected LastFailureAt to be set")
	}
	if r.LastSuccessAt().IsZero() {
		t.Fatal("expected LastSuccessAt to be set")
	}
}

func TestRunnerStopWaitsForInFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var inFlight atomic.Bool

	r := NewRunner(time.Hour, 2*time.Second, func(ctx context.Context) error {
		inFlight.Store(true)
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			// Still mark completion path; Stop waits for job return.
		}
		inFlight.Store(false)
		return nil
	}, WithName("stop-test"))

	r.Start(context.Background())

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job never started")
	}

	stopped := make(chan struct{})
	go func() {
		// Unblock job shortly after Stop begins waiting.
		time.Sleep(50 * time.Millisecond)
		close(release)
		r.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return")
	}
	if inFlight.Load() {
		t.Fatal("Stop returned while job still marked in-flight")
	}
	// Second Stop is safe.
	r.Stop()
}

func TestRunnerTickTimeout(t *testing.T) {
	var sawDeadline atomic.Bool
	r := NewRunner(time.Hour, 40*time.Millisecond, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				sawDeadline.Store(true)
			}
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return errors.New("job did not observe timeout")
		}
	}, WithName("timeout-test"))

	r.Start(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sawDeadline.Load() && r.FailCount() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	r.Stop()

	if !sawDeadline.Load() {
		t.Fatal("job did not observe per-tick deadline")
	}
	if r.FailCount() < 1 {
		t.Fatalf("expected fail count >= 1, got %d", r.FailCount())
	}
	if r.SuccessCount() != 0 {
		t.Fatalf("expected no successes, got %d", r.SuccessCount())
	}
}

func TestRunnerStopBeforeStart(t *testing.T) {
	r := NewRunner(time.Second, time.Second, func(ctx context.Context) error {
		return nil
	})
	// Must not hang or panic.
	r.Stop()
}

func TestRunnerSuccessMetrics(t *testing.T) {
	r := NewRunner(15*time.Millisecond, 50*time.Millisecond, func(ctx context.Context) error {
		return nil
	}, WithName("metrics-test"))
	r.Start(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r.SuccessCount() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	r.Stop()
	if r.SuccessCount() < 2 {
		t.Fatalf("success count = %d, want >= 2", r.SuccessCount())
	}
	if r.FailCount() != 0 {
		t.Fatalf("fail count = %d, want 0", r.FailCount())
	}
}
