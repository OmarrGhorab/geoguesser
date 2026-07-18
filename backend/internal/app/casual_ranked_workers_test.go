package app_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/competitive"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/platform/workers"
)

// --- worker fakes ---

type workerClaimReconciler struct {
	calls atomic.Int64
}

func (f *workerClaimReconciler) ReconcileExpiredClaims(context.Context) {
	f.calls.Add(1)
}

type workerProgression struct {
	calls atomic.Int64
}

func (f *workerProgression) RetryPendingFinalizations(context.Context) (processed, applied, failed int, err error) {
	f.calls.Add(1)
	return 2, 1, 0, nil
}

type workerSeasonRoller struct {
	calls atomic.Int64
}

type workerRankedDeadlines struct {
	calls atomic.Int64
	limit atomic.Int64
}

func (f *workerRankedDeadlines) SweepExpiredRankedRounds(_ context.Context, limit int) error {
	f.calls.Add(1)
	f.limit.Store(int64(limit))
	return nil
}

func (f *workerSeasonRoller) RollOverIfDue(context.Context) (*competitive.RolloverOutcome, error) {
	f.calls.Add(1)
	return &competitive.RolloverOutcome{Skipped: true}, nil
}

type workerLifecycleStore struct {
	disconnect []matchplay.DisconnectCandidate
	inactive   []matchplay.InactivityCandidate
	forfeits   atomic.Int64
	closes     atomic.Int64
	batchLimit atomic.Int64
}

func (s *workerLifecycleStore) LoadSnapshotBundle(context.Context, uuid.UUID) (*matchplay.SnapshotBundle, error) {
	return nil, nil
}
func (s *workerLifecycleStore) LoadRoundResultBundle(context.Context, uuid.UUID, uuid.UUID) (*matchplay.RoundResultBundle, error) {
	return nil, nil
}
func (s *workerLifecycleStore) LoadTerminalResultBundle(context.Context, uuid.UUID) (*matchplay.TerminalResultBundle, error) {
	return nil, nil
}
func (s *workerLifecycleStore) FindParticipant(context.Context, uuid.UUID, uuid.UUID) (*matchplay.MatchParticipant, error) {
	return nil, nil
}
func (s *workerLifecycleStore) ExplicitLeaveTx(context.Context, uuid.UUID, uuid.UUID, time.Time) (*matchplay.LeaveOutcome, error) {
	return nil, matchplay.ErrNotFound
}
func (s *workerLifecycleStore) ForfeitDisconnectTx(_ context.Context, matchID, userID uuid.UUID, now time.Time) (*matchplay.LeaveOutcome, error) {
	s.forfeits.Add(1)
	return &matchplay.LeaveOutcome{
		Match: matchplay.Match{
			ID: matchID, Mode: "casual_solo", Status: matchplay.MatchStatusCompleted,
			Playlist: matchplay.PlaylistCasual, Format: matchplay.FormatSolo,
			UpdatedAt: now,
		},
		AlreadyTerminal: false,
	}, nil
}
func (s *workerLifecycleStore) CloseInactiveCasualTx(_ context.Context, matchID uuid.UUID, now time.Time) (*matchplay.LeaveOutcome, error) {
	s.closes.Add(1)
	return &matchplay.LeaveOutcome{
		Match: matchplay.Match{
			ID: matchID, Mode: "casual_solo", Status: matchplay.MatchStatusCompleted,
			Playlist: matchplay.PlaylistCasual, Format: matchplay.FormatSolo,
			UpdatedAt: now,
		},
		AlreadyTerminal: false,
	}, nil
}
func (s *workerLifecycleStore) ListDisconnectCandidates(_ context.Context, _ time.Time, _ time.Duration, limit int) ([]matchplay.DisconnectCandidate, error) {
	s.batchLimit.Store(int64(limit))
	if limit < len(s.disconnect) {
		return s.disconnect[:limit], nil
	}
	return s.disconnect, nil
}
func (s *workerLifecycleStore) ListInactiveCasualMatches(_ context.Context, _ time.Time, limit int) ([]matchplay.InactivityCandidate, error) {
	if limit < len(s.inactive) {
		return s.inactive[:limit], nil
	}
	return s.inactive, nil
}
func (s *workerLifecycleStore) TouchActivity(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type workerPresence struct {
	windows map[string]bool
}

func (p *workerPresence) key(matchID, userID uuid.UUID) string {
	return matchID.String() + "|" + userID.String()
}
func (p *workerPresence) HasReconnectWindow(_ context.Context, matchID, userID uuid.UUID) (bool, error) {
	if p == nil || p.windows == nil {
		return false, nil
	}
	return p.windows[p.key(matchID, userID)], nil
}
func (p *workerPresence) ClearPresence(context.Context, uuid.UUID, uuid.UUID) error { return nil }

// --- tests ---

func TestWorkers_ClaimRecoveryRunner(t *testing.T) {
	t.Parallel()
	fake := &workerClaimReconciler{}
	runner := matchmaking.NewClaimSweepRunner(fake, matchmaking.ClaimSweepConfig{
		Interval: 15 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.Start(ctx)
	waitWorker(t, func() bool { return fake.calls.Load() >= 1 && runner.SuccessCount() >= 1 })
	runner.Stop()
	if fake.calls.Load() < 1 {
		t.Fatalf("claim recovery calls = %d", fake.calls.Load())
	}
}

func TestWorkers_ProgressionRetryAndSeasonRollover(t *testing.T) {
	t.Parallel()
	prog := &workerProgression{}
	progRunner := competitive.NewProgressionRetryRunner(prog, competitive.ProgressionWorkerConfig{
		Interval: 15 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, nil)

	roll := &workerSeasonRoller{}
	obs := &competitive.RolloverWorkerObservations{}
	rollRunner := competitive.NewSeasonRolloverRunner(roll, competitive.RolloverWorkerConfig{
		Interval: 15 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, nil, obs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	progRunner.Start(ctx)
	rollRunner.Start(ctx)
	waitWorker(t, func() bool {
		return prog.calls.Load() >= 1 && roll.calls.Load() >= 1 &&
			progRunner.SuccessCount() >= 1 && rollRunner.SuccessCount() >= 1
	})
	progRunner.Stop()
	rollRunner.Stop()

	if prog.calls.Load() < 1 {
		t.Fatal("progression retry never ran")
	}
	if roll.calls.Load() < 1 {
		t.Fatal("season rollover never ran")
	}
	_, _, outcome, _ := obs.Snapshot()
	if outcome != "skipped" && outcome != "applied" && outcome != "replay" {
		t.Fatalf("rollover outcome = %q", outcome)
	}
}

func TestWorkers_ReconnectForfeitAndCasualInactivity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	matchA, userA := uuid.New(), uuid.New()
	matchB := uuid.New()

	store := &workerLifecycleStore{
		disconnect: []matchplay.DisconnectCandidate{{MatchID: matchA, UserID: userA}},
		inactive:   []matchplay.InactivityCandidate{{MatchID: matchB}},
	}
	// No reconnect window → forfeit proceeds.
	presence := &workerPresence{windows: map[string]bool{}}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace:   90 * time.Second,
		CasualInactivity: 10 * time.Minute,
		SweepBatchSize:   50,
	}).WithClock(func() time.Time { return now })
	// Attach presence via constructor options if available.
	svc = svc.WithPresence(presence)

	if err := svc.RunLifecycleSweep(context.Background()); err != nil {
		t.Fatalf("lifecycle sweep: %v", err)
	}
	if store.forfeits.Load() < 1 {
		t.Fatal("expected disconnect forfeit")
	}
	if store.closes.Load() < 1 {
		t.Fatal("expected casual inactivity close")
	}

	// Reconnect window still open → skip forfeit.
	store2 := &workerLifecycleStore{
		disconnect: []matchplay.DisconnectCandidate{{MatchID: matchA, UserID: userA}},
	}
	presence2 := &workerPresence{windows: map[string]bool{
		matchA.String() + "|" + userA.String(): true,
	}}
	svc2 := matchplay.NewService(store2, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
		SweepBatchSize: 50,
	}).WithClock(func() time.Time { return now }).WithPresence(presence2)
	if err := svc2.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if store2.forfeits.Load() != 0 {
		t.Fatalf("forfeits = %d, want 0 while reconnect window open", store2.forfeits.Load())
	}
}

func TestWorkers_RoundDeadlinesPurePolicy(t *testing.T) {
	t.Parallel()
	// Ranked rounds have shared 60s deadlines; Casual never has ends_at.
	timer := games.RankedRoundTimerSeconds
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	rankedEnd := games.NextRoundEndsAt(games.GameModeRankedSolo, &timer, now)
	if rankedEnd == nil {
		t.Fatal("ranked must have deadline")
	}
	if !rankedEnd.Equal(now.Add(60 * time.Second)) {
		t.Fatalf("ranked ends_at = %v", rankedEnd)
	}
	if games.NextRoundEndsAt(games.GameModeCasualSolo, &timer, now) != nil {
		t.Fatal("casual must not receive ends_at even if timer provided")
	}
	if games.CasualHasDeadline(games.GameModeCasualDuo) {
		t.Fatal("casual has no scoring deadline")
	}
	if !games.IsRankedMode(games.GameModeRankedSquad) {
		t.Fatal("ranked squad should use deadline scoring")
	}

	deadline := &workerRankedDeadlines{}
	runner := games.NewRankedDeadlineRunner(deadline, games.RankedDeadlineWorkerConfig{
		Interval:  15 * time.Millisecond,
		Timeout:   50 * time.Millisecond,
		BatchSize: 17,
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.Start(ctx)
	waitWorker(t, func() bool { return deadline.calls.Load() >= 1 && runner.SuccessCount() >= 1 })
	runner.Stop()
	if deadline.limit.Load() != 17 {
		t.Fatalf("deadline batch limit = %d, want 17", deadline.limit.Load())
	}
}

func TestWorkers_RetentionCleanupBoundedBatch(t *testing.T) {
	t.Parallel()
	// Cleaner defaults batch size and workers.Runner graceful stop.
	store := &redactCleanupStore{} // empty candidates
	cleaner := matchplay.NewCleaner(store, &alwaysFailDeleter{}, nil, nil, matchplay.CleanupConfig{
		BatchSize: 0, // force default
	})
	stats, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if stats.MessagesDeleted != 0 {
		t.Fatalf("empty cleanup deleted = %d", stats.MessagesDeleted)
	}

	// Lifecycle sweep respects SweepBatchSize bound.
	candidates := make([]matchplay.DisconnectCandidate, 0, 20)
	for i := 0; i < 20; i++ {
		candidates = append(candidates, matchplay.DisconnectCandidate{MatchID: uuid.New(), UserID: uuid.New()})
	}
	storeLC := &workerLifecycleStore{disconnect: candidates}
	svc := matchplay.NewService(storeLC, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
		SweepBatchSize: 5,
	}).WithClock(func() time.Time { return time.Now().UTC() })
	if err := svc.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if storeLC.batchLimit.Load() != 5 {
		t.Fatalf("batch limit passed = %d, want 5", storeLC.batchLimit.Load())
	}
	if storeLC.forfeits.Load() != 5 {
		t.Fatalf("forfeits = %d, want bounded 5", storeLC.forfeits.Load())
	}
}

func TestWorkers_LifecycleRunnerAndGracefulShutdown(t *testing.T) {
	t.Parallel()
	store := &workerLifecycleStore{}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace:   90 * time.Second,
		CasualInactivity: 10 * time.Minute,
		SweepBatchSize:   10,
	}).WithClock(func() time.Time { return time.Now().UTC() })

	runner := matchplay.NewLifecycleRunner(svc, matchplay.LifecycleWorkerConfig{
		Interval: 15 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
	}, nil)

	// Cleanup runner also starts/stops cleanly.
	cleaner := matchplay.NewCleaner(&redactCleanupStore{}, nil, nil, nil, matchplay.CleanupConfig{BatchSize: 10})
	cleanupRunner := matchplay.NewCleanupRunner(cleaner, matchplay.CleanupWorkerConfig{
		Interval: 15 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	runner.Start(ctx)
	cleanupRunner.Start(ctx)
	waitWorker(t, func() bool {
		return runner.SuccessCount() >= 1 && cleanupRunner.SuccessCount() >= 1
	})

	// Graceful shutdown waits for in-flight ticks (runner.Stop).
	stopped := make(chan struct{})
	go func() {
		cancel()
		runner.Stop()
		cleanupRunner.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("workers did not stop gracefully")
	}

	// Double-stop is safe.
	runner.Stop()
	cleanupRunner.Stop()
}

func TestWorkers_GenericRunnerGracefulShutdown(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	var inFlight atomic.Bool

	r := workers.NewRunner(time.Hour, 2*time.Second, func(ctx context.Context) error {
		inFlight.Store(true)
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		inFlight.Store(false)
		return nil
	}, workers.WithName("graceful-test"))

	r.Start(context.Background())
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job never started")
	}

	done := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		close(release)
		r.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop hung")
	}
	if inFlight.Load() {
		t.Fatal("Stop returned while job still in-flight")
	}
}

func waitWorker(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("worker condition not met in time")
}
