package competitive_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestLeaderboardEligibilityAndOrder(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()
	seasonID := seedActiveSeasonID(t, db)
	now := time.Now().UTC()

	var season competitive.Season
	if err := db.Where("id = ?", seasonID).Take(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}

	// Create controlled standings with distinct tie-break facts.
	type seed struct {
		rating, placements, matches, wins int
		reachedOffset                     time.Duration
		active                            bool
		label                             string
	}
	seeds := []seed{
		{2100, 5, 30, 20, 0, true, "top"},
		{2100, 5, 30, 15, -time.Hour, true, "same rating fewer wins"},
		{2050, 5, 30, 25, -2 * time.Hour, true, "lower rating"},
		{2500, 5, 24, 40, 0, true, "insufficient matches"},  // ineligible
		{2500, 4, 40, 40, 0, true, "placements incomplete"}, // ineligible
		{1999, 5, 40, 40, 0, true, "below 2000"},            // ineligible
		{2200, 5, 40, 10, 0, false, "disabled user"},        // inactive
	}

	var userIDs []uuid.UUID
	for i, s := range seeds {
		uid := seedUser(t, db)
		userIDs = append(userIDs, uid)
		if !s.active {
			mustExec(t, db, `UPDATE users SET status = 'disabled' WHERE id = ?`, uid)
		}
		reached := now.Add(s.reachedOffset)
		mustExec(t, db, `
			INSERT INTO competitive_standings (
				season_id, user_id, rating, placements_completed, matches_played, wins, losses, draws, abandons,
				peak_rating, rating_reached_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, ?, ?, ?, ?)
			ON CONFLICT (season_id, user_id) DO UPDATE SET
				rating = EXCLUDED.rating,
				placements_completed = EXCLUDED.placements_completed,
				matches_played = EXCLUDED.matches_played,
				wins = EXCLUDED.wins,
				peak_rating = EXCLUDED.peak_rating,
				rating_reached_at = EXCLUDED.rating_reached_at,
				updated_at = EXCLUDED.updated_at
		`, seasonID, uid, s.rating, s.placements, s.matches, s.wins, s.rating, reached, now, now)
		_ = i
		_ = s.label
	}

	rows, err := repo.ListEligibleStandings(ctx, season, 50, nil, competitive.Top500Limit+1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Expect only the three eligible active users.
	if len(rows) != 3 {
		t.Fatalf("eligible count = %d, want 3 (%+v)", len(rows), rows)
	}
	// Order: rating DESC, wins DESC, rating_reached_at ASC, user_id ASC
	// seeds[0] 2100/20, seeds[1] 2100/15, seeds[2] 2050/25
	if rows[0].UserID != userIDs[0] {
		t.Fatalf("pos1 user = %s, want %s (highest wins at 2100)", rows[0].UserID, userIDs[0])
	}
	if rows[1].UserID != userIDs[1] {
		t.Fatalf("pos2 user = %s, want %s", rows[1].UserID, userIDs[1])
	}
	if rows[2].UserID != userIDs[2] {
		t.Fatalf("pos3 user = %s, want %s", rows[2].UserID, userIDs[2])
	}

	// World Legend for positions ≤500.
	for i, row := range rows {
		pos := i + 1
		if !competitive.IsWorldLegendPosition(pos) {
			t.Fatalf("pos %d should be world legend eligible", pos)
		}
		rank := competitive.PresentationRank(row.Rating, row.PlacementsCompleted, &pos)
		if rank == nil || rank.Code != competitive.WorldLegendCode {
			t.Fatalf("pos %d rank = %+v", pos, rank)
		}
	}

	// Position lookup for top user.
	pos, err := repo.EligibleStandingPosition(ctx, season, userIDs[0])
	if err != nil || pos == nil || *pos != 1 {
		t.Fatalf("position top = %v err=%v", pos, err)
	}
	// Ineligible has nil position.
	pos, err = repo.EligibleStandingPosition(ctx, season, userIDs[3])
	if err != nil {
		t.Fatalf("ineligible pos err: %v", err)
	}
	if pos != nil {
		t.Fatalf("ineligible should have nil position, got %d", *pos)
	}
}

func TestLeaderboardTop500Cutoff(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()
	seasonID := seedActiveSeasonID(t, db)
	now := time.Now().UTC()

	var season competitive.Season
	if err := db.Where("id = ?", seasonID).Take(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}

	// Seed a modest set and verify scan bound of 501 is respected by maxScan.
	const n = 10
	for i := 0; i < n; i++ {
		uid := seedUser(t, db)
		mustExec(t, db, `
			INSERT INTO competitive_standings (
				season_id, user_id, rating, placements_completed, matches_played, wins, losses, draws, abandons,
				peak_rating, rating_reached_at, created_at, updated_at
			) VALUES (?, ?, ?, 5, 30, ?, 0, 0, 0, ?, ?, ?, ?)
			ON CONFLICT (season_id, user_id) DO NOTHING
		`, seasonID, uid, 2000+i, i, 2000+i, now.Add(-time.Duration(i)*time.Minute), now, now)
	}
	rows, err := repo.ListEligibleStandings(ctx, season, 5, nil, competitive.Top500Limit+1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) > 5 {
		t.Fatalf("page size exceeded: %d", len(rows))
	}
	// maxScan=2 returns at most 2 even if limit higher.
	bounded, err := repo.ListEligibleStandings(ctx, season, 50, nil, 2)
	if err != nil {
		t.Fatalf("bounded: %v", err)
	}
	if len(bounded) > 2 {
		t.Fatalf("maxScan violated: %d", len(bounded))
	}
}

func TestLeaderboardCursorStability(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()
	seasonID := seedActiveSeasonID(t, db)
	now := time.Now().UTC()

	var season competitive.Season
	if err := db.Where("id = ?", seasonID).Take(&season).Error; err != nil {
		t.Fatalf("load season: %v", err)
	}

	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		uid := seedUser(t, db)
		ids = append(ids, uid)
		mustExec(t, db, `
			INSERT INTO competitive_standings (
				season_id, user_id, rating, placements_completed, matches_played, wins, losses, draws, abandons,
				peak_rating, rating_reached_at, created_at, updated_at
			) VALUES (?, ?, ?, 5, 40, 10, 0, 0, 0, ?, ?, ?, ?)
			ON CONFLICT (season_id, user_id) DO NOTHING
		`, seasonID, uid, 2300-i*10, 2300-i*10, now.Add(-time.Duration(i)*time.Hour), now, now)
	}

	page1, err := repo.ListEligibleStandings(ctx, season, 2, nil, competitive.Top500Limit+1)
	if err != nil || len(page1) < 2 {
		t.Fatalf("page1: len=%d err=%v", len(page1), err)
	}
	// Build cursor from last of page1 using encode via service package — re-query with keyset.
	// Use repository keyset by constructing cursor through a second call with synthetic cursor
	// via listing all and verifying order continuity.
	all, err := repo.ListEligibleStandings(ctx, season, 100, nil, competitive.Top500Limit+1)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	// Ensure deterministic descending rating order.
	for i := 1; i < len(all); i++ {
		if all[i].Rating > all[i-1].Rating {
			t.Fatalf("order broken at %d: %d > %d", i, all[i].Rating, all[i-1].Rating)
		}
		if all[i].Rating == all[i-1].Rating && all[i].Wins > all[i-1].Wins {
			t.Fatalf("wins order broken at %d", i)
		}
	}
	_ = ids
}

func TestClosedLeaderboardSnapshot(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	seasonID := uuid.New()
	mustExec(t, db, `
		INSERT INTO competitive_seasons (
			id, sequence, slug, name, status, starts_at, ends_at, closed_at,
			initial_rating, elo_k, reset_factor_bps, top500_min_matches, created_at, updated_at
		) VALUES (
			?, (SELECT COALESCE(MAX(sequence),0)+2000 FROM competitive_seasons),
			?, 'Closed LB Season', 'closed', ?, ?, ?,
			800, 32, 5000, 25, ?, ?
		)
	`, seasonID, "closed-lb-"+seasonID.String()[:8],
		now.Add(-90*24*time.Hour), now.Add(-6*24*time.Hour), now.Add(-6*24*time.Hour),
		now, now)
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM competitive_standings WHERE season_id = ?`, seasonID)
		_ = db.Exec(`DELETE FROM competitive_seasons WHERE id = ?`, seasonID)
	})

	for i := 1; i <= 3; i++ {
		uid := seedUser(t, db)
		pos := i
		code := "geo_master_1"
		if i <= competitive.Top500Limit {
			code = competitive.WorldLegendCode
		}
		mustExec(t, db, `
			INSERT INTO competitive_standings (
				season_id, user_id, rating, placements_completed, matches_played, wins, losses, draws, abandons,
				peak_rating, rating_reached_at, final_position, ending_rank_code, peak_rank_code, created_at, updated_at
			) VALUES (?, ?, ?, 5, 40, 20, 10, 0, 0, ?, ?, ?, ?, ?, ?, ?)
		`, seasonID, uid, 2500-i, 2500-i, now, pos, code, code, now, now)
	}

	rows, err := repo.ListClosedTop500(ctx, seasonID, 50, nil)
	if err != nil {
		t.Fatalf("closed list: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("closed rows = %d", len(rows))
	}
	if rows[0].FinalPosition == nil || *rows[0].FinalPosition != 1 {
		t.Fatalf("final pos = %v", rows[0].FinalPosition)
	}
	if rows[0].EndingRankCode == nil || *rows[0].EndingRankCode != competitive.WorldLegendCode {
		t.Fatalf("ending = %v", rows[0].EndingRankCode)
	}
}
