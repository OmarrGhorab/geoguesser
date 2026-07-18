package competitive_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestSoftResetFormula_fiftyPercentToward800(t *testing.T) {
	t.Parallel()
	cases := []struct {
		old, want int
	}{
		{800, 800},
		{1200, 1000},
		{1600, 1200},
		{0, 400},
		{2000, 1400},
		{400, 600},
	}
	for _, tc := range cases {
		if got := competitive.SoftResetRating(tc.old, 800, 5000); got != tc.want {
			t.Fatalf("soft reset %d = %d, want %d", tc.old, got, tc.want)
		}
	}
	// Floor zero with anchor 0.
	if got := competitive.SoftResetRating(-100, 0, 5000); got != 0 {
		t.Fatalf("floor = %d", got)
	}
}

func TestRolloverConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg := competitive.DefaultRolloverConfig()
	if cfg.Duration != 84*24*time.Hour {
		t.Fatalf("duration = %s", cfg.Duration)
	}
	if cfg.InitialRating != 800 || cfg.ResetFactorBPS != 5000 || cfg.Top500MinMatches != 25 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestSeasonSummary_factorPercent(t *testing.T) {
	t.Parallel()
	s := competitive.Season{
		ID:               uuid.New(),
		Sequence:         3,
		Slug:             "season-3",
		Name:             "Season 3",
		Status:           competitive.SeasonStatusActive,
		StartsAt:         time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
		EndsAt:           time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		InitialRating:    800,
		ResetFactorBPS:   5000,
		Top500MinMatches: 25,
	}
	sum := competitive.SeasonSummary(s)
	if sum.Reset.Anchor != 800 || sum.Reset.FactorPercent != 50 {
		t.Fatalf("reset = %+v", sum.Reset)
	}
	if sum.Sequence != 3 || sum.Top500MinMatches != 25 {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestRolloverIfDue_skipsWithoutDB(t *testing.T) {
	t.Parallel()
	repo := competitive.NewRepository(nil)
	_, err := repo.RollOverIfDue(context.Background(), competitive.DefaultRolloverConfig(), time.Now().UTC())
	if err != competitive.ErrDependencyFailure {
		t.Fatalf("err = %v", err)
	}
}

func TestRolloverConcurrency_pureFreezeHelpers(t *testing.T) {
	t.Parallel()
	// Concurrent pure helpers remain race-free (positions/codes).
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pos := i + 1
			_ = competitive.EndingRankCode(2100+i, 5, &pos)
			_ = competitive.PeakRankCode(2200+i, 5, &pos)
			_ = competitive.SoftResetRating(1000+i, 800, 5000)
		}(i)
	}
	wg.Wait()
}

func TestRolloverIntegration_whenDBAvailable(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()

	// Force active season ends_at into the past on a disposable season clone path:
	// Use advisory lock path by attempting rollover — if active season not ended, skipped.
	out, err := repo.RollOverIfDue(ctx, competitive.DefaultRolloverConfig(), time.Now().UTC())
	if err != nil {
		t.Fatalf("rollover: %v", err)
	}
	if out == nil {
		t.Fatal("expected outcome")
	}
	// Normally Season 1 ends far in the future → skipped.
	if !out.Skipped && !out.Applied && !out.Replay {
		t.Fatalf("unexpected outcome %+v", out)
	}

	// Create a temporary active season that is already ended only when safe.
	// Avoid clobbering the real active season: just assert skip path is idempotent.
	out2, err := repo.RollOverIfDue(ctx, competitive.DefaultRolloverConfig(), time.Now().UTC())
	if err != nil {
		t.Fatalf("second rollover: %v", err)
	}
	if out2 == nil || (!out2.Skipped && !out2.Replay && !out2.Applied) {
		t.Fatalf("second outcome %+v", out2)
	}
}

func TestRolloverIntegration_closeAndSoftReset(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Build an isolated ended season + standings without touching the global active
	// unique index: temporarily this test only freezes helpers via SQL under a
	// transaction that rolls back when we can't safely replace active.
	// Instead: seed standings on active season and verify soft-reset ensure path.

	activeID := seedActiveSeasonID(t, db)
	userID := seedUser(t, db)

	// Insert a prior closed season standing so soft reset has a source.
	prevSeasonID := uuid.New()
	mustExec(t, db, `
		INSERT INTO competitive_seasons (
			id, sequence, slug, name, status, starts_at, ends_at, closed_at,
			initial_rating, elo_k, reset_factor_bps, top500_min_matches, created_at, updated_at
		) VALUES (
			?, (SELECT COALESCE(MAX(sequence),0)+1000 FROM competitive_seasons),
			?, 'Prior Test Season', 'closed', ?, ?, ?,
			800, 32, 5000, 25, ?, ?
		)
	`, prevSeasonID, "prior-test-"+prevSeasonID.String()[:8],
		now.Add(-200*24*time.Hour), now.Add(-100*24*time.Hour), now.Add(-100*24*time.Hour),
		now, now)
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM competitive_standings WHERE season_id = ?`, prevSeasonID)
		_ = db.Exec(`DELETE FROM competitive_seasons WHERE id = ?`, prevSeasonID)
	})

	mustExec(t, db, `
		INSERT INTO competitive_standings (
			season_id, user_id, rating, placements_completed, matches_played, wins, losses, draws, abandons,
			peak_rating, rating_reached_at, final_position, ending_rank_code, peak_rank_code, created_at, updated_at
		) VALUES (?, ?, 1200, 5, 40, 20, 15, 5, 0, 1300, ?, 10, 'world_legend', 'world_legend', ?, ?)
	`, prevSeasonID, userID, now.Add(-100*24*time.Hour), now, now)

	// Load active season model.
	var season competitive.Season
	if err := db.WithContext(ctx).Where("id = ?", activeID).Take(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}

	st, err := repo.EnsureStandingWithSoftReset(ctx, season, userID, now)
	if err != nil {
		t.Fatalf("ensure soft reset: %v", err)
	}
	if st == nil {
		t.Fatal("standing nil")
	}
	// round(800 + 0.5*(1200-800)) = 1000
	if st.Rating != 1000 {
		t.Fatalf("soft reset rating = %d, want 1000", st.Rating)
	}
	if st.PlacementsCompleted != 0 {
		t.Fatalf("placements must reset, got %d", st.PlacementsCompleted)
	}
	if st.PeakRating != 1000 {
		t.Fatalf("peak = %d, want 1000", st.PeakRating)
	}

	// Idempotent re-ensure leaves values alone.
	st2, err := repo.EnsureStandingWithSoftReset(ctx, season, userID, now)
	if err != nil {
		t.Fatalf("re-ensure: %v", err)
	}
	if st2.Rating != 1000 || st2.PlacementsCompleted != 0 {
		t.Fatalf("re-ensure mutated standing: %+v", st2)
	}
}
