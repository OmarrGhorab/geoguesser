package matchmaking_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping matchmaking repository integration tests")
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
	return db
}

func TestMatchModelTableNames(t *testing.T) {
	if (matchmaking.Match{}).TableName() != "matches" {
		t.Fatalf("Match table name")
	}
	if (matchmaking.MatchPlayer{}).TableName() != "match_players" {
		t.Fatalf("MatchPlayer table name")
	}
}

func TestMigrationIntegrityConstraints(t *testing.T) {
	db := testDB(t)

	// Verify expected tables exist after migration 00014.
	for _, table := range []string{"matches", "match_players"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist (run goose up with 00014_matchmaking_ranked.sql)", table)
		}
	}
}

func TestRepositoryFindActiveUserMissing(t *testing.T) {
	db := testDB(t)
	repo := matchmaking.NewRepository(db)
	user, err := repo.FindActiveUser(t.Context(), uuid.MustParse("00000000-0000-0000-0000-0000000000aa"))
	if err != nil {
		t.Fatalf("FindActiveUser: %v", err)
	}
	if user != nil {
		t.Fatalf("expected nil user, got %+v", user)
	}
}

func seedFormationFixtures(t *testing.T, db *gorm.DB) (mapID uuid.UUID, userA, userB uuid.UUID, locationIDs []uuid.UUID) {
	t.Helper()
	mapID = uuid.New()
	userA, userB = uuid.New(), uuid.New()
	now := time.Now().UTC()

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("seed exec failed: %v\nsql=%s", err, sql)
		}
	}

	mustExec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		userA, userA.String()+"@example.test", now, now)
	mustExec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		userB, userB.String()+"@example.test", now, now)
	mustExec(`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, 'Alpha', 'en', ?, ?)`, userA, now, now)
	mustExec(`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, 'Bravo', 'en', ?, ?)`, userB, now, now)
	mustExec(`INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Ranked Test Map', 'public', 'free', 'mixed', 'active', ?, ?)`,
		mapID, "ranked-test-"+mapID.String()[:8], now, now)

	locationIDs = make([]uuid.UUID, 5)
	for i := range locationIDs {
		locationIDs[i] = uuid.New()
		mustExec(`INSERT INTO locations (id, country_code, latitude, longitude, difficulty, provider, provider_ref, status, created_at, updated_at)
			VALUES (?, 'US', 0, 0, 'easy', 'test', ?, 'active', ?, ?)`,
			locationIDs[i], locationIDs[i].String(), now, now)
		mustExec(`INSERT INTO map_locations (map_id, location_id, selection_weight, created_at) VALUES (?, ?, 1, ?)`,
			mapID, locationIDs[i], now)
	}
	return mapID, userA, userB, locationIDs
}

func TestCreateFormationBundle_ExactlyTwoParticipantsAndReplay(t *testing.T) {
	db := testDB(t)
	repo := matchmaking.NewRepository(db)
	mapID, userA, userB, locationIDs := seedFormationFixtures(t, db)
	formationKey := "formation-" + uuid.NewString()
	matchedAt := time.Date(2026, 7, 11, 17, 0, 0, 0, time.UTC)

	first, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: formationKey,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("create formation: %v", err)
	}
	if first.Match.ID == uuid.Nil || first.Match.GameID == uuid.Nil {
		t.Fatalf("match incomplete: %+v", first.Match)
	}
	if first.Match.Status != matchmaking.MatchStatusActive {
		t.Fatalf("status = %q, want active (game starts active at formation)", first.Match.Status)
	}
	if first.Match.StartedAt == nil {
		t.Fatal("started_at required for active match")
	}
	var activeParticipants int64
	if err := db.Table("match_players").Where("match_id = ? AND status = ?", first.Match.ID, matchmaking.ParticipantStatusActive).Count(&activeParticipants).Error; err != nil {
		t.Fatalf("count active participants: %v", err)
	}
	if activeParticipants != 2 {
		t.Fatalf("active participants = %d, want 2", activeParticipants)
	}

	var participantCount int64
	if err := db.Table("match_players").Where("match_id = ?", first.Match.ID).Count(&participantCount).Error; err != nil {
		t.Fatalf("count participants: %v", err)
	}
	if participantCount != 2 {
		t.Fatalf("participants = %d, want 2", participantCount)
	}

	var gamePlayerCount int64
	if err := db.Table("game_players").Where("game_id = ?", first.Match.GameID).Count(&gamePlayerCount).Error; err != nil {
		t.Fatalf("count game players: %v", err)
	}
	if gamePlayerCount != 2 {
		t.Fatalf("game_players = %d, want 2", gamePlayerCount)
	}

	var roundCount int64
	if err := db.Table("rounds").Where("game_id = ?", first.Match.GameID).Count(&roundCount).Error; err != nil {
		t.Fatalf("count rounds: %v", err)
	}
	if roundCount != 5 {
		t.Fatalf("rounds = %d, want 5", roundCount)
	}

	// Formation key replay is idempotent.
	second, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: formationKey,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("replay formation: %v", err)
	}
	if second.Match.ID != first.Match.ID || second.Match.GameID != first.Match.GameID {
		t.Fatalf("replay diverged: first=%+v second=%+v", first.Match, second.Match)
	}

	// Partial active assignment uniqueness: third formation including userA must fail.
	userC := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		userC, userC.String()+"@example.test", now, now).Error; err != nil {
		t.Fatalf("seed userC: %v", err)
	}
	if err := db.Exec(`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, 'Charlie', 'en', ?, ?)`,
		userC, now, now).Error; err != nil {
		t.Fatalf("seed profile C: %v", err)
	}
	_, err = repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: "formation-" + uuid.NewString(),
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userC},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt.Add(time.Minute),
	})
	if err != matchmaking.ErrAlreadyAssigned {
		t.Fatalf("expected ErrAlreadyAssigned, got %v", err)
	}

	// Active assignment lookup works for both original players.
	for _, uid := range []uuid.UUID{userA, userB} {
		asg, err := repo.FindActiveAssignment(t.Context(), uid)
		if err != nil || asg == nil {
			t.Fatalf("assignment for %s: %+v err=%v", uid, asg, err)
		}
		if asg.MatchID != first.Match.ID || asg.GameID != first.Match.GameID {
			t.Fatalf("assignment mismatch: %+v", asg)
		}
	}
}

func TestTransitionMatch_LifecycleAndCompletedEligibility(t *testing.T) {
	db := testDB(t)
	repo := matchmaking.NewRepository(db)
	mapID, userA, userB, locationIDs := seedFormationFixtures(t, db)
	formationKey := "formation-life-" + uuid.NewString()
	matchedAt := time.Date(2026, 7, 11, 18, 0, 0, 0, time.UTC)

	formed, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: formationKey,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Formation already activates the match with the ranked game.
	if formed.Match.Status != matchmaking.MatchStatusActive {
		t.Fatalf("formed status = %q, want active", formed.Match.Status)
	}
	// Idempotent start replay: already active treats matched->active as applied.
	if err := repo.TransitionMatch(t.Context(), formed.Match.ID, matchmaking.MatchStatusMatched, matchmaking.MatchStatusActive, matchedAt); err != nil {
		t.Fatalf("active start replay: %v", err)
	}

	completedAt := matchedAt.Add(10 * time.Minute)
	// Completing match without completed game must not appear in rated results.
	if err := repo.TransitionMatch(t.Context(), formed.Match.ID, matchmaking.MatchStatusActive, matchmaking.MatchStatusCompleted, completedAt); err != nil {
		t.Fatalf("active->completed: %v", err)
	}
	results, err := repo.ListCompletedRankedResults(t.Context(), 10)
	if err != nil {
		t.Fatalf("list results: %v", err)
	}
	for _, m := range results {
		if m.ID == formed.Match.ID {
			t.Fatal("completed match with non-completed game must not qualify")
		}
	}

	// Mark game completed; now it qualifies.
	if err := db.Exec(`UPDATE games SET status = 'completed', completed_at = ? WHERE id = ?`, completedAt, formed.Match.GameID).Error; err != nil {
		t.Fatalf("complete game: %v", err)
	}
	results, err = repo.ListCompletedRankedResults(t.Context(), 10)
	if err != nil {
		t.Fatalf("list results after game complete: %v", err)
	}
	found := false
	for _, m := range results {
		if m.ID == formed.Match.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected completed match+game in ranked results")
	}

	// Terminal immutability: further transition fails (not idempotent to different status).
	if err := repo.TransitionMatch(t.Context(), formed.Match.ID, matchmaking.MatchStatusCompleted, matchmaking.MatchStatusCancelled, completedAt.Add(time.Minute)); err == nil {
		t.Fatal("expected terminal transition rejection")
	}
}

func TestMatchPlayersRetentionRestrictsDeletes(t *testing.T) {
	db := testDB(t)
	if !db.Migrator().HasTable("match_players") {
		t.Skip("match_players missing")
	}
	var restrictCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.referential_constraints rc
		  ON rc.constraint_name = tc.constraint_name
		 AND rc.constraint_schema = tc.constraint_schema
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = 'match_players'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND rc.delete_rule = 'RESTRICT'
	`).Scan(&restrictCount).Error; err != nil {
		t.Fatalf("query constraints: %v", err)
	}
	// user_id, game_player_id, match_id should all restrict deletes.
	if restrictCount < 3 {
		t.Fatalf("expected >=3 RESTRICT FKs on match_players, got %d", restrictCount)
	}

	var matchesRestrict int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.referential_constraints rc
		  ON rc.constraint_name = tc.constraint_name
		 AND rc.constraint_schema = tc.constraint_schema
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = 'matches'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND rc.delete_rule = 'RESTRICT'
	`).Scan(&matchesRestrict).Error; err != nil {
		t.Fatalf("query matches constraints: %v", err)
	}
	if matchesRestrict < 1 {
		t.Fatalf("expected RESTRICT FK from matches.game_id, got %d", matchesRestrict)
	}
}
