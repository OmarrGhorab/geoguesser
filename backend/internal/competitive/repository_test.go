package competitive_test

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func competitiveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping competitive repository integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if !db.Migrator().HasTable("competitive_seasons") ||
		!db.Migrator().HasTable("competitive_standings") ||
		!db.Migrator().HasTable("competitive_rating_changes") {
		t.Skip("migration 00019 competitive tables missing; run goose up for 00019_casual_ranked_team_modes.sql")
	}
	if !db.Migrator().HasColumn("matches", "progression_finalized_at") {
		t.Skip("migration 00019 match columns missing")
	}
	return db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec failed: %v\nsql=%s", err, sql)
	}
}

func seedActiveSeasonID(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	var row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := db.Raw(`SELECT id FROM competitive_seasons WHERE status = 'active' LIMIT 1`).Scan(&row).Error
	if err != nil || row.ID == uuid.Nil {
		t.Skip("no active competitive season seeded")
	}
	return row.ID
}

func seedUser(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	mustExec(t, db,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		id, id.String()+"@competitive.test", now, now)
	mustExec(t, db,
		`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, ?, 'en', ?, ?)`,
		id, "C"+id.String()[:8], now, now)
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM competitive_rating_changes WHERE user_id = ?`, id)
		_ = db.Exec(`DELETE FROM competitive_standings WHERE user_id = ?`, id)
		_ = db.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, id)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, id)
	})
	return id
}

func seedMap(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	mapID := uuid.New()
	now := time.Now().UTC()
	mustExec(t, db, `
		INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Competitive Test Map', 'public', 'free', 'mixed', 'active', ?, ?)
	`, mapID, "comp-test-"+mapID.String()[:8], now, now)
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM maps WHERE id = ?`, mapID)
	})
	return mapID
}

type rankedMatchFixture struct {
	MatchID  uuid.UUID
	GameID   uuid.UUID
	SeasonID uuid.UUID
	UserA    uuid.UUID
	UserB    uuid.UUID
	GPA      uuid.UUID
	GPB      uuid.UUID
}

func seedCompletedRanked1v1(t *testing.T, db *gorm.DB, seasonID uuid.UUID, result string, winnerSlot *int, abandonUser *uuid.UUID) rankedMatchFixture {
	t.Helper()
	now := time.Now().UTC()
	mapID := seedMap(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)
	gameID := uuid.New()
	matchID := uuid.New()
	gpA, gpB := uuid.New(), uuid.New()

	mustExec(t, db, `
		INSERT INTO games (id, mode, status, map_id, round_count, timer_seconds, scoring_version, total_score, started_at, completed_at, created_at, updated_at)
		VALUES (?, 'ranked', 'completed', ?, 5, 60, 1, 1800, ?, ?, ?, ?)
	`, gameID, mapID, now.Add(-9*time.Minute), now.Add(-time.Minute), now, now)

	mustExec(t, db, `
		INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'A', 'player', 'active', 1000, ?, 1)
	`, gpA, gameID, userA, now)
	mustExec(t, db, `
		INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'B', 'player', 'active', 800, ?, 2)
	`, gpB, gameID, userB, now)

	mustExec(t, db, `
		INSERT INTO matches (
			id, formation_key, game_id, mode, status, playlist, format, team_size,
			season_id, matched_at, started_at, completed_at, closed_at, result,
			winner_team_slot, team_one_score, team_two_score, last_activity_at,
			created_at, updated_at
		) VALUES (
			?, ?, ?, 'ranked_solo', 'completed', 'ranked', 'solo', 1,
			?, ?, ?, ?, ?, ?,
			?, 1000, 800, ?,
			?, ?
		)
	`, matchID, "comp-fk-"+matchID.String(), gameID, seasonID,
		now.Add(-10*time.Minute), now.Add(-9*time.Minute), now.Add(-time.Minute), nil,
		result, winnerSlot, now.Add(-time.Minute), now, now)

	mustExec(t, db, `
		INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at, team_slot)
		VALUES (?, ?, ?, 'completed', ?, ?, 1)
	`, matchID, userA, gpA, now.Add(-10*time.Minute), now.Add(-time.Minute))
	mustExec(t, db, `
		INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at, team_slot)
		VALUES (?, ?, ?, 'completed', ?, ?, 2)
	`, matchID, userB, gpB, now.Add(-10*time.Minute), now.Add(-time.Minute))

	if abandonUser != nil {
		mustExec(t, db, `
			UPDATE match_players
			SET abandoned_at = ?, abandon_reason = 'explicit_leave'
			WHERE match_id = ? AND user_id = ?
		`, now.Add(-2*time.Minute), matchID, *abandonUser)
	}

	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM competitive_rating_changes WHERE match_id = ?`, matchID)
		_ = db.Exec(`DELETE FROM match_players WHERE match_id = ?`, matchID)
		_ = db.Exec(`DELETE FROM matches WHERE id = ?`, matchID)
		_ = db.Exec(`DELETE FROM game_players WHERE game_id = ?`, gameID)
		_ = db.Exec(`DELETE FROM games WHERE id = ?`, gameID)
	})

	return rankedMatchFixture{
		MatchID:  matchID,
		GameID:   gameID,
		SeasonID: seasonID,
		UserA:    userA,
		UserB:    userB,
		GPA:      gpA,
		GPB:      gpB,
	}
}

func seedStanding(t *testing.T, db *gorm.DB, seasonID, userID uuid.UUID, rating, placements int) {
	t.Helper()
	now := time.Now().UTC()
	mustExec(t, db, `
		INSERT INTO competitive_standings (
			season_id, user_id, rating, placements_completed, matches_played,
			wins, losses, draws, abandons, peak_rating, rating_reached_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, 0, 0, 0, 0, 0, ?, ?, ?, ?)
		ON CONFLICT (season_id, user_id) DO UPDATE SET
			rating = EXCLUDED.rating,
			placements_completed = EXCLUDED.placements_completed,
			peak_rating = EXCLUDED.peak_rating,
			matches_played = 0,
			wins = 0,
			losses = 0,
			draws = 0,
			abandons = 0,
			last_match_id = NULL
	`, seasonID, userID, rating, placements, rating, now, now, now)
}

func TestFinalizeMatchProgression_equalRatedPlus16Minus16(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "team_one_win", intPtr(1), nil)
	seedStanding(t, db, seasonID, fx.UserA, 800, 5)
	seedStanding(t, db, seasonID, fx.UserB, 800, 5)

	out, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID:        fx.MatchID,
		EloK:           32,
		AbandonPenalty: 15,
		InitialRating:  800,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if out.Replay {
		t.Fatal("first finalize must not be replay")
	}
	if len(out.Changes) != 2 {
		t.Fatalf("changes = %d, want 2", len(out.Changes))
	}

	byUser := map[uuid.UUID]competitive.RatingChange{}
	for _, c := range out.Changes {
		byUser[c.UserID] = c
	}
	win := byUser[fx.UserA]
	loss := byUser[fx.UserB]
	if win.BaseDelta != 16 || win.TotalDelta != 16 || win.NewRating != 816 {
		t.Fatalf("winner change = %+v, want +16 → 816", win)
	}
	if loss.BaseDelta != -16 || loss.TotalDelta != -16 || loss.NewRating != 784 {
		t.Fatalf("loser change = %+v, want -16 → 784", loss)
	}
	if win.AbandonPenalty != 0 || loss.AbandonPenalty != 0 {
		t.Fatalf("unexpected abandon penalties")
	}

	// progression_finalized_at set
	var finalized *time.Time
	if err := db.Raw(`SELECT progression_finalized_at FROM matches WHERE id = ?`, fx.MatchID).Scan(&finalized).Error; err != nil {
		t.Fatalf("read finalized: %v", err)
	}
	if finalized == nil {
		t.Fatal("expected progression_finalized_at")
	}
}

func TestFinalizeMatchProgression_idempotentReplay(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "draw", nil, nil)
	seedStanding(t, db, seasonID, fx.UserA, 1000, 5)
	seedStanding(t, db, seasonID, fx.UserB, 1000, 5)

	first, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !second.Replay {
		t.Fatal("second call must be replay")
	}
	if len(first.Changes) != len(second.Changes) {
		t.Fatalf("change count mismatch %d vs %d", len(first.Changes), len(second.Changes))
	}

	var count int
	if err := db.Raw(`SELECT COUNT(*) FROM competitive_rating_changes WHERE match_id = ?`, fx.MatchID).Scan(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("rating changes count = %d, want 2 (exact-once)", count)
	}

	// Ratings must not double-apply.
	var ratingA int
	if err := db.Raw(`SELECT rating FROM competitive_standings WHERE season_id = ? AND user_id = ?`, seasonID, fx.UserA).
		Scan(&ratingA).Error; err != nil {
		t.Fatalf("rating: %v", err)
	}
	if ratingA != 1000 {
		t.Fatalf("draw rating = %d, want 1000 (no double apply)", ratingA)
	}
}

func TestFinalizeMatchProgression_abandonPenalty(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	// Prepare users first so we can mark abandoner.
	// seedCompletedRanked1v1 needs abandon user after create — pass pointer after.
	fx := seedCompletedRanked1v1(t, db, seasonID, "forfeit", intPtr(2), nil)
	// Mark userA abandoned (team 1 forfeits → team 2 wins).
	mustExec(t, db, `
		UPDATE match_players SET abandoned_at = now(), abandon_reason = 'explicit_leave'
		WHERE match_id = ? AND user_id = ?
	`, fx.MatchID, fx.UserA)

	seedStanding(t, db, seasonID, fx.UserA, 800, 5)
	seedStanding(t, db, seasonID, fx.UserB, 800, 5)

	out, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	byUser := map[uuid.UUID]competitive.RatingChange{}
	for _, c := range out.Changes {
		byUser[c.UserID] = c
	}
	quitter := byUser[fx.UserA]
	winner := byUser[fx.UserB]
	if quitter.BaseDelta != -16 {
		t.Fatalf("quitter base = %d, want -16", quitter.BaseDelta)
	}
	if quitter.AbandonPenalty != -15 {
		t.Fatalf("quitter abandon = %d, want -15", quitter.AbandonPenalty)
	}
	if quitter.TotalDelta != -31 || quitter.NewRating != 769 {
		t.Fatalf("quitter total/new = %d/%d, want -31/769", quitter.TotalDelta, quitter.NewRating)
	}
	if winner.BaseDelta != 16 || winner.AbandonPenalty != 0 {
		t.Fatalf("winner must get normal win only: %+v", winner)
	}
}

func TestFinalizeMatchProgression_floorZero(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "team_two_win", intPtr(2), nil)
	seedStanding(t, db, seasonID, fx.UserA, 5, 5)
	seedStanding(t, db, seasonID, fx.UserB, 5, 5)

	out, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	for _, c := range out.Changes {
		if c.UserID == fx.UserA {
			if c.NewRating != 0 {
				t.Fatalf("floored new rating = %d, want 0", c.NewRating)
			}
			if c.TotalDelta != -5 {
				t.Fatalf("floored total_delta = %d, want -5", c.TotalDelta)
			}
		}
	}
}

func TestFinalizeMatchProgression_concurrentExactOnce(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "team_one_win", intPtr(1), nil)
	seedStanding(t, db, seasonID, fx.UserA, 900, 5)
	seedStanding(t, db, seasonID, fx.UserB, 900, 5)

	const workers = 12
	var wg sync.WaitGroup
	results := make(chan *competitive.FinalizeOutcome, workers)
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
				MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
			})
			if err != nil {
				errs <- err
				return
			}
			results <- out
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent finalize error: %v", err)
	}

	var applied, replays int
	var firstChanges []competitive.RatingChange
	for out := range results {
		if out.Replay {
			replays++
		} else {
			applied++
		}
		if firstChanges == nil {
			firstChanges = out.Changes
		}
		if len(out.Changes) != 2 {
			t.Fatalf("changes = %d", len(out.Changes))
		}
	}
	if applied+replays != workers {
		t.Fatalf("applied=%d replays=%d workers=%d", applied, replays, workers)
	}
	// At most one non-replay (others may also report non-replay only if they
	// lost the race after writing — match lock ensures single writer).
	if applied > 1 {
		t.Fatalf("expected at most one applied writer, got %d", applied)
	}

	var count int
	if err := db.Raw(`SELECT COUNT(*) FROM competitive_rating_changes WHERE match_id = ?`, fx.MatchID).Scan(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("unique match/user changes violated: count=%d", count)
	}

	// Standings updated exactly once: +16 / -16 from 900.
	var ra, rb int
	if err := db.Raw(`SELECT rating FROM competitive_standings WHERE season_id = ? AND user_id = ?`, seasonID, fx.UserA).Scan(&ra).Error; err != nil {
		t.Fatalf("rating a: %v", err)
	}
	if err := db.Raw(`SELECT rating FROM competitive_standings WHERE season_id = ? AND user_id = ?`, seasonID, fx.UserB).Scan(&rb).Error; err != nil {
		t.Fatalf("rating b: %v", err)
	}
	if ra != 916 || rb != 884 {
		t.Fatalf("ratings A/B = %d/%d, want 916/884", ra, rb)
	}

	var finalized *time.Time
	if err := db.Raw(`SELECT progression_finalized_at FROM matches WHERE id = ?`, fx.MatchID).Scan(&finalized).Error; err != nil {
		t.Fatalf("finalized: %v", err)
	}
	if finalized == nil {
		t.Fatal("progression_finalized_at missing after concurrent finalization")
	}
}

func TestFinalizeMatchProgression_rollbackOnInvalidResult(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "forfeit", nil, nil) // forfeit without winner
	seedStanding(t, db, seasonID, fx.UserA, 800, 5)
	seedStanding(t, db, seasonID, fx.UserB, 800, 5)

	_, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error for forfeit without winner")
	}

	var count int
	if err := db.Raw(`SELECT COUNT(*) FROM competitive_rating_changes WHERE match_id = ?`, fx.MatchID).Scan(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback failed: rating changes = %d", count)
	}
	var finalized sql.NullTime
	if err := db.Raw(`SELECT progression_finalized_at FROM matches WHERE id = ?`, fx.MatchID).Scan(&finalized).Error; err != nil {
		t.Fatalf("finalized: %v", err)
	}
	if finalized.Valid {
		t.Fatal("rollback failed: progression_finalized_at set")
	}
	var rating int
	if err := db.Raw(`SELECT rating FROM competitive_standings WHERE season_id = ? AND user_id = ?`, seasonID, fx.UserA).
		Scan(&rating).Error; err != nil {
		t.Fatalf("rating: %v", err)
	}
	if rating != 800 {
		t.Fatalf("standing mutated on rollback: %d", rating)
	}
}

func TestEnsureStandings_stableOrderAndLazyCreate(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	// Create users in reverse order of UUID sort to prove stable ordering.
	u1 := seedUser(t, db)
	u2 := seedUser(t, db)
	// Ensure we pass unsorted.
	ids := []uuid.UUID{u2, u1, u2}
	rows, err := repo.EnsureStandings(context.Background(), seasonID, ids, 800, time.Now().UTC())
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if compareUUID(rows[0].UserID, rows[1].UserID) > 0 {
		t.Fatalf("standings not sorted: %s then %s", rows[0].UserID, rows[1].UserID)
	}
	for _, st := range rows {
		if st.Rating != 800 || st.PeakRating != 800 {
			t.Fatalf("lazy standing = %+v", st)
		}
	}
}

func TestListPendingRankedMatchIDs(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	seasonID := seedActiveSeasonID(t, db)
	fx := seedCompletedRanked1v1(t, db, seasonID, "team_one_win", intPtr(1), nil)

	ids, err := repo.ListPendingRankedMatchIDs(context.Background(), 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == fx.MatchID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pending match not listed")
	}

	// After finalization, should disappear.
	seedStanding(t, db, seasonID, fx.UserA, 800, 5)
	seedStanding(t, db, seasonID, fx.UserB, 800, 5)
	if _, err := repo.FinalizeMatchProgression(context.Background(), competitive.FinalizeInput{
		MatchID: fx.MatchID, EloK: 32, AbandonPenalty: 15, InitialRating: 800, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	ids, err = repo.ListPendingRankedMatchIDs(context.Background(), 100)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	for _, id := range ids {
		if id == fx.MatchID {
			t.Fatal("finalized match still pending")
		}
	}
}

func TestService_FinalizeMatchSymmetricAndPlacements(t *testing.T) {
	db := competitiveTestDB(t)
	repo := competitive.NewRepository(db)
	svc := competitive.NewService(repo, competitive.DefaultConfig(), nil)
	seasonID := seedActiveSeasonID(t, db)

	// 2v2-like with two users still works as 1v1 fixture; use duo by adding parties? Keep 1v1.
	fx := seedCompletedRanked1v1(t, db, seasonID, "team_one_win", intPtr(1), nil)
	// Placement users: hidden ranks.
	seedStanding(t, db, seasonID, fx.UserA, 800, 2)
	seedStanding(t, db, seasonID, fx.UserB, 800, 2)

	res, err := svc.FinalizeMatch(context.Background(), fx.MatchID)
	if err != nil {
		t.Fatalf("service finalize: %v", err)
	}
	if res.Replay || res.Pending {
		t.Fatalf("unexpected flags: %+v", res)
	}
	for _, dto := range res.ByUser {
		if dto.OldRating != nil || dto.NewRating != nil {
			t.Fatalf("placements must hide rating: %+v", dto)
		}
		if dto.Placement == nil || dto.Placement.Completed != 3 {
			t.Fatalf("placement progress = %+v, want completed 3", dto.Placement)
		}
	}

	// Matchmaking reader surface.
	snaps, err := svc.StandingsForUsers(context.Background(), seasonID, []uuid.UUID{fx.UserA, fx.UserB})
	if err != nil {
		t.Fatalf("standings: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("snaps = %d", len(snaps))
	}
	for _, s := range snaps {
		if s.RankCode != "" || s.NamedRankTier != 0 {
			t.Fatalf("hidden placement snapshot = %+v", s)
		}
		if s.PlacementsCompleted != 3 {
			t.Fatalf("placements_completed = %d, want 3", s.PlacementsCompleted)
		}
	}
}

func intPtr(v int) *int { return &v }

func compareUUID(a, b uuid.UUID) int {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
