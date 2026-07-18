package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testAppDB opens PostgreSQL when DATABASE_URL is set; otherwise skips.
func testAppDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping casual/ranked migration integration tests")
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

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/app -> backend/migrations
	dir := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
	if _, err := os.Stat(filepath.Join(dir, "00019_casual_ranked_team_modes.sql")); err != nil {
		t.Fatalf("migration file missing under %s: %v", dir, err)
	}
	return dir
}

func migration00019Applied(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	return hasTable(t, db, "parties") &&
		hasTable(t, db, "competitive_seasons") &&
		hasColumn(t, db, "matches", "playlist") &&
		hasColumn(t, db, "guesses", "accuracy_score")
}

func hasColumn(t *testing.T, db *gorm.DB, table, column string) bool {
	t.Helper()
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = ?
			  AND column_name = ?
		)`, table, column).Scan(&exists).Error
	if err != nil {
		t.Fatalf("hasColumn %s.%s: %v", table, column, err)
	}
	return exists
}

func hasTable(t *testing.T, db *gorm.DB, table string) bool {
	t.Helper()
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = ?
		)`, table).Scan(&exists).Error
	if err != nil {
		t.Fatalf("hasTable %s: %v", table, err)
	}
	return exists
}

func hasIndex(t *testing.T, db *gorm.DB, indexName string) bool {
	t.Helper()
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = current_schema()
			  AND indexname = ?
		)`, indexName).Scan(&exists).Error
	if err != nil {
		t.Fatalf("hasIndex %s: %v", indexName, err)
	}
	return exists
}

func hasCheckConstraint(t *testing.T, db *gorm.DB, table, constraint string) bool {
	t.Helper()
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			WHERE n.nspname = current_schema()
			  AND t.relname = ?
			  AND c.conname = ?
			  AND c.contype = 'c'
		)`, table, constraint).Scan(&exists).Error
	if err != nil {
		t.Fatalf("hasCheckConstraint %s.%s: %v", table, constraint, err)
	}
	return exists
}

// extractGooseSection returns the SQL body for "+goose Up" or "+goose Down".
func extractGooseSection(content, direction string) string {
	marker := "-- +goose " + direction
	start := strings.Index(content, marker)
	if start < 0 {
		return ""
	}
	body := content[start+len(marker):]
	if direction == "Up" {
		if down := strings.Index(body, "-- +goose Down"); down >= 0 {
			body = body[:down]
		}
	}
	return strings.TrimSpace(body)
}

// execGooseSQL executes a goose SQL section inside a single transaction,
// honoring StatementBegin/End blocks and skipping goose directive lines.
func execGooseSQL(t *testing.T, db *gorm.DB, section string) error {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	stmts := parseGooseStatements(section)
	tx, err := sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec migration statement failed: %w\n--- sql ---\n%s", err, truncate(stmt, 400))
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration tx: %w", err)
	}
	return nil
}

func parseGooseStatements(section string) []string {
	lines := strings.Split(section, "\n")
	var (
		stmts        []string
		buf          strings.Builder
		inBlock      bool
		blockBuilder strings.Builder
	)
	flush := func() {
		s := strings.TrimSpace(buf.String())
		buf.Reset()
		if s != "" {
			stmts = append(stmts, s)
		}
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "-- +goose StatementBegin") {
			inBlock = true
			blockBuilder.Reset()
			continue
		}
		if strings.HasPrefix(trim, "-- +goose StatementEnd") {
			inBlock = false
			s := strings.TrimSpace(blockBuilder.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			blockBuilder.Reset()
			continue
		}
		if strings.HasPrefix(trim, "-- +goose") {
			continue
		}
		if inBlock {
			blockBuilder.WriteString(line)
			blockBuilder.WriteByte('\n')
			continue
		}
		// Strip full-line SQL comments outside blocks.
		if strings.HasPrefix(trim, "--") {
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			flush()
		}
	}
	flush()
	return stmts
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func ensureCasualRankedMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	if migration00019Applied(t, db) {
		return
	}
	// Prior ranked foundations must exist.
	if !hasTable(t, db, "matches") || !hasTable(t, db, "match_players") {
		t.Skip("matches table missing; run goose up through 00014+ first")
	}
	path := filepath.Join(migrationsDir(t), "00019_casual_ranked_team_modes.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	up := extractGooseSection(string(raw), "Up")
	if up == "" {
		t.Fatal("empty Up section in 00019 migration")
	}
	if err := execGooseSQL(t, db, up); err != nil {
		t.Fatalf("apply 00019 up: %v", err)
	}
	// Record goose version when the version table exists (best-effort).
	if hasTable(t, db, "goose_db_version") {
		_ = db.Exec(`
			INSERT INTO goose_db_version (version_id, is_applied, tstamp)
			SELECT 19, TRUE, now()
			WHERE NOT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = 19)
		`).Error
	}
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec failed: %v\nsql=%s", err, sql)
	}
}

// scanUUID reads a UUID column that drivers may return as string or []byte.
func scanUUID(t *testing.T, db *gorm.DB, query string, args ...any) uuid.UUID {
	t.Helper()
	var raw string
	if err := db.Raw(query, args...).Scan(&raw).Error; err != nil {
		t.Fatalf("scanUUID: %v\nquery=%s", err, query)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("parse UUID %q: %v", raw, err)
	}
	return id
}

func seedLegacyRankedFixtures(t *testing.T, db *gorm.DB, withGuesses bool) (activeMatchID, completedMatchID uuid.UUID, userA, userB uuid.UUID) {
	t.Helper()
	now := time.Now().UTC()
	mapID := uuid.New()
	userA, userB = uuid.New(), uuid.New()
	activeMatchID = uuid.New()
	completedMatchID = uuid.New()
	activeGameID := uuid.New()
	completedGameID := uuid.New()
	locID := uuid.New()

	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userA, userA.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userB, userB.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'Alpha', 'en', ?, ?)`, userA, now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'Bravo', 'en', ?, ?)`, userB, now, now)
	mustExec(t, db, `INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Casual Ranked Migration Map', 'public', 'free', 'mixed', 'active', ?, ?)`,
		mapID, "crm-map-"+mapID.String()[:8], now, now)
	mustExec(t, db, `INSERT INTO locations (id, country_code, latitude, longitude, difficulty, provider, provider_ref, status, created_at, updated_at)
		VALUES (?, 'US', 1.0, 2.0, 'easy', 'test', ?, 'active', ?, ?)`,
		locID, "crm-loc-"+locID.String(), now, now)
	mustExec(t, db, `INSERT INTO map_locations (map_id, location_id, selection_weight, created_at) VALUES (?, ?, 1, ?)`,
		mapID, locID, now)

	// Active legacy ranked match (matched lifecycle).
	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, timer_seconds, scoring_version, total_score, started_at, created_at, updated_at)
		VALUES (?, 'ranked', 'active', ?, ?, 5, 60, 1, 0, ?, ?, ?)`,
		activeGameID, mapID, userA, now, now, now)
	gpA1 := uuid.New()
	gpB1 := uuid.New()
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at)
		VALUES (?, ?, ?, 'Alpha', 'player', 'active', 0, ?)`, gpA1, activeGameID, userA, now)
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at)
		VALUES (?, ?, ?, 'Bravo', 'player', 'active', 0, ?)`, gpB1, activeGameID, userB, now)
	mustExec(t, db, `INSERT INTO matches (id, formation_key, game_id, mode, status, matched_at, started_at, created_at, updated_at)
		VALUES (?, ?, ?, 'ranked_standard', 'active', ?, ?, ?, ?)`,
		activeMatchID, "crm-active-"+activeMatchID.String(), activeGameID, now.Add(-2*time.Minute), now.Add(-time.Minute), now, now)
	// Deterministic order: userB assigned earlier => team_slot 1; userA later => team_slot 2.
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at)
		VALUES (?, ?, ?, 'active', ?)`, activeMatchID, userB, gpB1, now.Add(-90*time.Second))
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at)
		VALUES (?, ?, ?, 'active', ?)`, activeMatchID, userA, gpA1, now.Add(-60*time.Second))

	// Completed legacy ranked match.
	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, timer_seconds, scoring_version, total_score, started_at, completed_at, created_at, updated_at)
		VALUES (?, 'ranked', 'completed', ?, ?, 5, 60, 1, 4200, ?, ?, ?, ?)`,
		completedGameID, mapID, userA, now.Add(-time.Hour), now.Add(-30*time.Minute), now, now)
	gpA2 := uuid.New()
	gpB2 := uuid.New()
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at)
		VALUES (?, ?, ?, 'Alpha', 'player', 'active', 2500, ?)`, gpA2, completedGameID, userA, now.Add(-time.Hour))
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at)
		VALUES (?, ?, ?, 'Bravo', 'player', 'active', 1700, ?)`, gpB2, completedGameID, userB, now.Add(-time.Hour))
	mustExec(t, db, `INSERT INTO matches (id, formation_key, game_id, mode, status, matched_at, started_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, 'ranked_standard', 'completed', ?, ?, ?, ?, ?)`,
		completedMatchID, "crm-done-"+completedMatchID.String(), completedGameID,
		now.Add(-2*time.Hour), now.Add(-time.Hour), now.Add(-30*time.Minute), now, now)
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at)
		VALUES (?, ?, ?, 'completed', ?, ?)`, completedMatchID, userA, gpA2, now.Add(-2*time.Hour), now.Add(-30*time.Minute))
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at)
		VALUES (?, ?, ?, 'completed', ?, ?)`, completedMatchID, userB, gpB2, now.Add(-2*time.Hour).Add(time.Second), now.Add(-30*time.Minute))

	if withGuesses {
		roundID := uuid.New()
		mustExec(t, db, `INSERT INTO rounds (id, game_id, location_id, round_number, status, starts_at, ends_at, revealed_at, created_at)
			VALUES (?, ?, ?, 1, 'completed', ?, ?, ?, ?)`,
			roundID, completedGameID, locID, now.Add(-50*time.Minute), now.Add(-49*time.Minute), now.Add(-49*time.Minute), now)
		mustExec(t, db, `INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, submitted_at, created_at)
			VALUES (?, ?, ?, 10.0, 20.0, 1000, 4321, ?, ?)`,
			uuid.New(), roundID, gpA2, now.Add(-49*time.Minute).Add(-10*time.Second), now)
		mustExec(t, db, `INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, submitted_at, created_at)
			VALUES (?, ?, ?, 11.0, 21.0, 2000, 2100, ?, ?)`,
			uuid.New(), roundID, gpB2, now.Add(-49*time.Minute).Add(-5*time.Second), now)
	}

	return activeMatchID, completedMatchID, userA, userB
}

func cleanupMigrationFixtures(t *testing.T, db *gorm.DB, matchIDs []uuid.UUID, userIDs []uuid.UUID) {
	t.Helper()
	for _, mid := range matchIDs {
		var gameIDRaw string
		_ = db.Raw(`SELECT game_id::text FROM matches WHERE id = ?`, mid).Scan(&gameIDRaw)
		_ = db.Exec(`DELETE FROM team_message_reports WHERE match_id = ?`, mid)
		_ = db.Exec(`DELETE FROM team_messages WHERE match_id = ?`, mid)
		_ = db.Exec(`DELETE FROM match_mutes WHERE match_id = ?`, mid)
		_ = db.Exec(`DELETE FROM competitive_rating_changes WHERE match_id = ?`, mid)
		_ = db.Exec(`UPDATE parties SET active_match_id = NULL WHERE active_match_id = ?`, mid)
		_ = db.Exec(`UPDATE match_players SET party_id = NULL WHERE match_id = ?`, mid)
		_ = db.Exec(`DELETE FROM match_players WHERE match_id = ?`, mid)
		_ = db.Exec(`DELETE FROM matches WHERE id = ?`, mid)
		if gameIDRaw != "" {
			if gameID, err := uuid.Parse(gameIDRaw); err == nil {
				_ = db.Exec(`DELETE FROM guesses WHERE game_player_id IN (SELECT id FROM game_players WHERE game_id = ?)`, gameID)
				_ = db.Exec(`DELETE FROM rounds WHERE game_id = ?`, gameID)
				_ = db.Exec(`DELETE FROM game_players WHERE game_id = ?`, gameID)
				_ = db.Exec(`DELETE FROM games WHERE id = ?`, gameID)
			}
		}
	}
	for _, uid := range userIDs {
		_ = db.Exec(`DELETE FROM competitive_standings WHERE user_id = ?`, uid)
		_ = db.Exec(`DELETE FROM party_invites WHERE inviter_user_id = ? OR invitee_user_id = ?`, uid, uid)
		_ = db.Exec(`DELETE FROM party_members WHERE user_id = ?`, uid)
		_ = db.Exec(`DELETE FROM parties WHERE leader_user_id = ?`, uid)
		_ = db.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, uid)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, uid)
	}
}

func TestCasualRankedMigration_SchemaTablesColumnsConstraints(t *testing.T) {
	db := testAppDB(t)
	ensureCasualRankedMigration(t, db)

	tables := []string{
		"parties",
		"party_members",
		"party_invites",
		"competitive_seasons",
		"competitive_standings",
		"competitive_rating_changes",
		"team_messages",
		"match_mutes",
		"team_message_reports",
	}
	for _, table := range tables {
		if !hasTable(t, db, table) {
			t.Fatalf("expected table %s to exist after migration 00019", table)
		}
	}

	matchCols := []string{
		"playlist", "format", "team_size", "season_id",
		"team_one_score", "team_two_score", "winner_team_slot", "result",
		"last_activity_at", "chat_access_until", "progression_finalized_at",
	}
	for _, col := range matchCols {
		if !hasColumn(t, db, "matches", col) {
			t.Fatalf("expected matches.%s", col)
		}
	}
	for _, col := range []string{"team_slot", "party_id", "abandoned_at", "abandon_reason"} {
		if !hasColumn(t, db, "match_players", col) {
			t.Fatalf("expected match_players.%s", col)
		}
	}
	if !hasColumn(t, db, "game_players", "team_slot") {
		t.Fatal("expected game_players.team_slot")
	}
	for _, col := range []string{"accuracy_score", "speed_bonus"} {
		if !hasColumn(t, db, "guesses", col) {
			t.Fatalf("expected guesses.%s", col)
		}
	}
	for _, table := range []string{"uploads", "files"} {
		for _, col := range []string{"purpose", "context_id", "sanitization_status", "raw_storage_key"} {
			if !hasColumn(t, db, table, col) {
				t.Fatalf("expected %s.%s", table, col)
			}
		}
	}

	indexes := []string{
		"parties_leader_status_idx",
		"parties_status_updated_at_idx",
		"parties_active_match_id_idx",
		"party_members_active_user_uidx",
		"party_members_party_status_joined_idx",
		"party_invites_pending_unique",
		"competitive_seasons_one_active_uidx",
		"competitive_standings_top500_idx",
		"matches_playlist_format_status_matched_idx",
		"matches_season_status_completed_idx",
		"matches_status_last_activity_idx",
		"match_players_match_team_status_user_idx",
		"game_players_game_team_status_id_idx",
		"team_messages_match_team_sequence_idx",
		"uploads_purpose_context_idx",
		"files_purpose_context_idx",
	}
	for _, idx := range indexes {
		if !hasIndex(t, db, idx) {
			t.Fatalf("expected index %s", idx)
		}
	}

	checks := []struct {
		table, name string
	}{
		{"matches", "matches_mode_check"},
		{"matches", "matches_playlist_check"},
		{"matches", "matches_team_size_format_check"},
		{"matches", "matches_season_playlist_check"},
		{"guesses", "guesses_score_range"},
		{"guesses", "guesses_accuracy_score_range"},
		{"guesses", "guesses_speed_bonus_range"},
		{"guesses", "guesses_score_components_check"},
		{"parties", "parties_format_check"},
		{"competitive_seasons", "competitive_seasons_status_check"},
	}
	for _, c := range checks {
		if !hasCheckConstraint(t, db, c.table, c.name) {
			t.Fatalf("expected check constraint %s on %s", c.name, c.table)
		}
	}
}

func TestCasualRankedMigration_Season1Seed(t *testing.T) {
	db := testAppDB(t)
	ensureCasualRankedMigration(t, db)

	type seasonRow struct {
		Sequence         int
		Slug             string
		Name             string
		Status           string
		InitialRating    int
		EloK             int       `gorm:"column:elo_k"`
		ResetFactorBps   int       `gorm:"column:reset_factor_bps"`
		Top500MinMatches int       `gorm:"column:top500_min_matches"`
		StartsAt         time.Time `gorm:"column:starts_at"`
		EndsAt           time.Time `gorm:"column:ends_at"`
	}
	var season seasonRow
	err := db.Raw(`
		SELECT sequence, slug, name, status, initial_rating, elo_k, reset_factor_bps, top500_min_matches, starts_at, ends_at
		FROM competitive_seasons
		WHERE sequence = 1
	`).Scan(&season).Error
	if err != nil {
		t.Fatalf("load season 1: %v", err)
	}
	if season.Sequence != 1 || season.Slug != "season-1" || season.Name != "Season 1" || season.Status != "active" {
		t.Fatalf("unexpected season seed identity: %+v", season)
	}
	if season.InitialRating != 800 || season.EloK != 32 || season.ResetFactorBps != 5000 || season.Top500MinMatches != 25 {
		t.Fatalf("unexpected season constants: %+v", season)
	}
	duration := season.EndsAt.Sub(season.StartsAt)
	// 84 days with a small clock/tolerance window.
	if duration < 83*24*time.Hour || duration > 85*24*time.Hour {
		t.Fatalf("expected ~84 day season window, got %s", duration)
	}

	var activeCount int64
	if err := db.Raw(`SELECT COUNT(*) FROM competitive_seasons WHERE status = 'active'`).Scan(&activeCount).Error; err != nil {
		t.Fatalf("count active seasons: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly one active season, got %d", activeCount)
	}
}

func TestCasualRankedMigration_LegacyBackfillAndUpgradePreservation(t *testing.T) {
	db := testAppDB(t)

	// Path A: migration not yet applied — seed legacy rows first, then upgrade.
	if !migration00019Applied(t, db) {
		if !hasTable(t, db, "matches") {
			t.Skip("matches table missing; run goose up through 00014+ first")
		}
		activeID, completedID, userA, userB := seedLegacyRankedFixtures(t, db, true)
		t.Cleanup(func() {
			cleanupMigrationFixtures(t, db, []uuid.UUID{activeID, completedID}, []uuid.UUID{userA, userB})
		})
		ensureCasualRankedMigration(t, db)
		assertLegacyBackfill(t, db, activeID, completedID, userA, userB)
		return
	}

	// Path B: already upgraded — insert post-migration fixtures shaped like
	// preserved legacy ranked matches and assert constraints/preservation.
	activeID, completedID, userA, userB := seedPostMigrationRankedFixtures(t, db)
	t.Cleanup(func() {
		cleanupMigrationFixtures(t, db, []uuid.UUID{activeID, completedID}, []uuid.UUID{userA, userB})
	})
	assertLegacyBackfill(t, db, activeID, completedID, userA, userB)

	// Any pre-existing ranked_standard rows must remain coherently backfilled.
	var badCount int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM matches
		WHERE mode = 'ranked_standard'
		  AND (
			playlist IS DISTINCT FROM 'ranked'
			OR format IS DISTINCT FROM 'solo'
			OR team_size IS DISTINCT FROM 1
			OR season_id IS NULL
		  )
	`).Scan(&badCount).Error; err != nil {
		t.Fatalf("scan legacy match integrity: %v", err)
	}
	if badCount != 0 {
		t.Fatalf("found %d ranked_standard matches without proper backfill fields", badCount)
	}

	var badGuesses int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM guesses
		WHERE accuracy_score IS NULL
		   OR speed_bonus IS NULL
		   OR score <> accuracy_score + speed_bonus
	`).Scan(&badGuesses).Error; err != nil {
		t.Fatalf("scan guess integrity: %v", err)
	}
	if badGuesses != 0 {
		t.Fatalf("found %d guesses violating accuracy/bonus invariants", badGuesses)
	}
}

func seedPostMigrationRankedFixtures(t *testing.T, db *gorm.DB) (activeMatchID, completedMatchID, userA, userB uuid.UUID) {
	t.Helper()
	now := time.Now().UTC()
	mapID := uuid.New()
	userA, userB = uuid.New(), uuid.New()
	activeMatchID = uuid.New()
	completedMatchID = uuid.New()
	activeGameID := uuid.New()
	completedGameID := uuid.New()
	locID := uuid.New()

	seasonID := scanUUID(t, db, `SELECT id FROM competitive_seasons WHERE sequence = 1`)

	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userA, userA.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userB, userB.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'Alpha', 'en', ?, ?)`, userA, now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'Bravo', 'en', ?, ?)`, userB, now, now)
	mustExec(t, db, `INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Post Migration Map', 'public', 'free', 'mixed', 'active', ?, ?)`,
		mapID, "crm-post-"+mapID.String()[:8], now, now)
	mustExec(t, db, `INSERT INTO locations (id, country_code, latitude, longitude, difficulty, provider, provider_ref, status, created_at, updated_at)
		VALUES (?, 'US', 1.0, 2.0, 'easy', 'test', ?, 'active', ?, ?)`,
		locID, "crm-post-loc-"+locID.String(), now, now)
	mustExec(t, db, `INSERT INTO map_locations (map_id, location_id, selection_weight, created_at) VALUES (?, ?, 1, ?)`,
		mapID, locID, now)

	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, timer_seconds, scoring_version, total_score, started_at, created_at, updated_at)
		VALUES (?, 'ranked', 'active', ?, ?, 5, 60, 1, 0, ?, ?, ?)`,
		activeGameID, mapID, userA, now, now, now)
	gpA1, gpB1 := uuid.New(), uuid.New()
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'Alpha', 'player', 'active', 0, ?, 2)`, gpA1, activeGameID, userA, now)
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'Bravo', 'player', 'active', 0, ?, 1)`, gpB1, activeGameID, userB, now)
	mustExec(t, db, `INSERT INTO matches (
		id, formation_key, game_id, mode, status, matched_at, started_at, created_at, updated_at,
		playlist, format, team_size, season_id, last_activity_at
	) VALUES (?, ?, ?, 'ranked_standard', 'active', ?, ?, ?, ?, 'ranked', 'solo', 1, ?, ?)`,
		activeMatchID, "crm-post-active-"+activeMatchID.String(), activeGameID,
		now.Add(-2*time.Minute), now.Add(-time.Minute), now, now, seasonID, now)
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, team_slot)
		VALUES (?, ?, ?, 'active', ?, 1)`, activeMatchID, userB, gpB1, now.Add(-90*time.Second))
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, team_slot)
		VALUES (?, ?, ?, 'active', ?, 2)`, activeMatchID, userA, gpA1, now.Add(-60*time.Second))

	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, timer_seconds, scoring_version, total_score, started_at, completed_at, created_at, updated_at)
		VALUES (?, 'ranked', 'completed', ?, ?, 5, 60, 1, 4200, ?, ?, ?, ?)`,
		completedGameID, mapID, userA, now.Add(-time.Hour), now.Add(-30*time.Minute), now, now)
	gpA2, gpB2 := uuid.New(), uuid.New()
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'Alpha', 'player', 'active', 2500, ?, 1)`, gpA2, completedGameID, userA, now.Add(-time.Hour))
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'Bravo', 'player', 'active', 1700, ?, 2)`, gpB2, completedGameID, userB, now.Add(-time.Hour))
	mustExec(t, db, `INSERT INTO matches (
		id, formation_key, game_id, mode, status, matched_at, started_at, completed_at, created_at, updated_at,
		playlist, format, team_size, season_id, last_activity_at, team_one_score, team_two_score, winner_team_slot, result
	) VALUES (?, ?, ?, 'ranked_standard', 'completed', ?, ?, ?, ?, ?, 'ranked', 'solo', 1, ?, ?, 2500, 1700, 1, 'team_one_win')`,
		completedMatchID, "crm-post-done-"+completedMatchID.String(), completedGameID,
		now.Add(-2*time.Hour), now.Add(-time.Hour), now.Add(-30*time.Minute), now, now, seasonID, now.Add(-30*time.Minute))
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at, team_slot)
		VALUES (?, ?, ?, 'completed', ?, ?, 1)`, completedMatchID, userA, gpA2, now.Add(-2*time.Hour), now.Add(-30*time.Minute))
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, completed_at, team_slot)
		VALUES (?, ?, ?, 'completed', ?, ?, 2)`, completedMatchID, userB, gpB2, now.Add(-2*time.Hour).Add(time.Second), now.Add(-30*time.Minute))

	roundID := uuid.New()
	mustExec(t, db, `INSERT INTO rounds (id, game_id, location_id, round_number, status, starts_at, ends_at, revealed_at, created_at)
		VALUES (?, ?, ?, 1, 'completed', ?, ?, ?, ?)`,
		roundID, completedGameID, locID, now.Add(-50*time.Minute), now.Add(-49*time.Minute), now.Add(-49*time.Minute), now)
	mustExec(t, db, `INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus, submitted_at, created_at)
		VALUES (?, ?, ?, 10.0, 20.0, 1000, 4321, 4321, 0, ?, ?)`,
		uuid.New(), roundID, gpA2, now.Add(-49*time.Minute).Add(-10*time.Second), now)
	mustExec(t, db, `INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus, submitted_at, created_at)
		VALUES (?, ?, ?, 11.0, 21.0, 2000, 2100, 2100, 0, ?, ?)`,
		uuid.New(), roundID, gpB2, now.Add(-49*time.Minute).Add(-5*time.Second), now)

	return activeMatchID, completedMatchID, userA, userB
}

func assertLegacyBackfill(t *testing.T, db *gorm.DB, activeID, completedID, userA, userB uuid.UUID) {
	t.Helper()

	type matchRow struct {
		Mode     string
		Playlist string
		Format   string
		TeamSize int
		SeasonID string `gorm:"column:season_id"`
		Status   string
		LastAct  time.Time `gorm:"column:last_activity_at"`
	}
	for _, mid := range []uuid.UUID{activeID, completedID} {
		var m matchRow
		if err := db.Raw(`
			SELECT mode, playlist, format, team_size, season_id::text, status, last_activity_at
			FROM matches WHERE id = ?
		`, mid).Scan(&m).Error; err != nil {
			t.Fatalf("load match %s: %v", mid, err)
		}
		if m.Mode != "ranked_standard" || m.Playlist != "ranked" || m.Format != "solo" || m.TeamSize != 1 {
			t.Fatalf("match %s backfill mismatch: %+v", mid, m)
		}
		if _, err := uuid.Parse(m.SeasonID); err != nil {
			t.Fatalf("match %s missing/invalid season_id %q: %v", mid, m.SeasonID, err)
		}
		if m.LastAct.IsZero() {
			t.Fatalf("match %s missing last_activity_at", mid)
		}
	}

	// Active match: userB assigned first => team 1, userA => team 2.
	assertTeamSlot(t, db, activeID, userB, 1)
	assertTeamSlot(t, db, activeID, userA, 2)
	// Completed match: userA assigned first => team 1, userB => team 2.
	assertTeamSlot(t, db, completedID, userA, 1)
	assertTeamSlot(t, db, completedID, userB, 2)

	// game_players team slots copied from match_players for matchmade rows.
	var gpSlots int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM game_players gp
		JOIN match_players mp ON mp.game_player_id = gp.id
		WHERE mp.match_id IN (?, ?)
		  AND gp.team_slot IS DISTINCT FROM mp.team_slot
	`, activeID, completedID).Scan(&gpSlots).Error; err != nil {
		t.Fatalf("game_player team slot check: %v", err)
	}
	if gpSlots != 0 {
		t.Fatalf("game_players.team_slot mismatch count=%d", gpSlots)
	}

	type guessRow struct {
		Score         int
		AccuracyScore int `gorm:"column:accuracy_score"`
		SpeedBonus    int `gorm:"column:speed_bonus"`
	}
	var guesses []guessRow
	if err := db.Raw(`
		SELECT g.score, g.accuracy_score, g.speed_bonus
		FROM guesses g
		JOIN game_players gp ON gp.id = g.game_player_id
		JOIN match_players mp ON mp.game_player_id = gp.id
		WHERE mp.match_id = ?
		ORDER BY g.score DESC
	`, completedID).Scan(&guesses).Error; err != nil {
		t.Fatalf("load guesses: %v", err)
	}
	if len(guesses) != 2 {
		t.Fatalf("expected 2 guesses on completed match, got %d", len(guesses))
	}
	for _, g := range guesses {
		if g.AccuracyScore != g.Score || g.SpeedBonus != 0 {
			t.Fatalf("guess backfill mismatch: %+v", g)
		}
	}
}

func assertTeamSlot(t *testing.T, db *gorm.DB, matchID, userID uuid.UUID, want int) {
	t.Helper()
	var slot int
	if err := db.Raw(`SELECT team_slot FROM match_players WHERE match_id = ? AND user_id = ?`, matchID, userID).Scan(&slot).Error; err != nil {
		t.Fatalf("team_slot for match=%s user=%s: %v", matchID, userID, err)
	}
	if slot != want {
		t.Fatalf("match %s user %s team_slot=%d want %d", matchID, userID, slot, want)
	}
}

func TestCasualRankedMigration_ConstraintRejection(t *testing.T) {
	db := testAppDB(t)
	ensureCasualRankedMigration(t, db)

	now := time.Now().UTC()
	userID := uuid.New()
	mapID := uuid.New()
	gameID := uuid.New()
	matchID := uuid.New()
	locID := uuid.New()
	gpID := uuid.New()
	roundID := uuid.New()

	seasonID := scanUUID(t, db, `SELECT id FROM competitive_seasons WHERE sequence = 1`)

	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userID, userID.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'Constraint', 'en', ?, ?)`, userID, now, now)
	mustExec(t, db, `INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Constraint Map', 'public', 'free', 'mixed', 'active', ?, ?)`,
		mapID, "crm-c-"+mapID.String()[:8], now, now)
	mustExec(t, db, `INSERT INTO locations (id, country_code, latitude, longitude, difficulty, provider, provider_ref, status, created_at, updated_at)
		VALUES (?, 'US', 0, 0, 'easy', 'test', ?, 'active', ?, ?)`, locID, "crm-c-loc-"+locID.String(), now, now)
	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, timer_seconds, scoring_version, total_score, created_at, updated_at)
		VALUES (?, 'ranked', 'pending', ?, ?, 5, 60, 1, 0, ?, ?)`, gameID, mapID, userID, now, now)
	mustExec(t, db, `INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at, team_slot)
		VALUES (?, ?, ?, 'C', 'player', 'active', 0, ?, 1)`, gpID, gameID, userID, now)
	mustExec(t, db, `INSERT INTO matches (
		id, formation_key, game_id, mode, status, matched_at, created_at, updated_at,
		playlist, format, team_size, season_id, last_activity_at
	) VALUES (?, ?, ?, 'ranked_solo', 'matched', ?, ?, ?, 'ranked', 'solo', 1, ?, ?)`,
		matchID, "crm-constraint-"+matchID.String(), gameID, now, now, now, seasonID, now)
	mustExec(t, db, `INSERT INTO match_players (match_id, user_id, game_player_id, status, assigned_at, team_slot)
		VALUES (?, ?, ?, 'assigned', ?, 1)`, matchID, userID, gpID, now)
	mustExec(t, db, `INSERT INTO rounds (id, game_id, location_id, round_number, status, created_at)
		VALUES (?, ?, ?, 1, 'pending', ?)`, roundID, gameID, locID, now)

	t.Cleanup(func() {
		cleanupMigrationFixtures(t, db, []uuid.UUID{matchID}, []uuid.UUID{userID})
		_ = db.Exec(`DELETE FROM map_locations WHERE map_id = ?`, mapID)
		_ = db.Exec(`DELETE FROM locations WHERE id = ?`, locID)
		_ = db.Exec(`DELETE FROM maps WHERE id = ?`, mapID)
	})

	reject := func(name, q string, args ...any) {
		t.Helper()
		err := db.Exec(q, args...).Error
		if err == nil {
			t.Fatalf("%s: expected constraint rejection", name)
		}
	}

	// Invalid mode.
	reject("bad mode", `UPDATE matches SET mode = 'quick_play' WHERE id = ?`, matchID)

	// Ranked without season.
	reject("ranked missing season", `
		UPDATE matches SET season_id = NULL WHERE id = ?`, matchID)

	// Casual with season.
	reject("casual with season", `
		UPDATE matches
		SET mode = 'casual_solo', playlist = 'casual', format = 'solo', team_size = 1, season_id = ?
		WHERE id = ?`, seasonID, matchID)

	// Team size disagrees with format.
	reject("team size mismatch", `
		UPDATE matches SET format = 'duo', team_size = 1, mode = 'ranked_duo' WHERE id = ?`, matchID)

	// Invalid team slot.
	reject("bad team slot", `UPDATE match_players SET team_slot = 3 WHERE match_id = ?`, matchID)

	// Abandon reason without timestamp.
	reject("abandon lifecycle", `
		UPDATE match_players SET abandon_reason = 'explicit_leave', abandoned_at = NULL WHERE match_id = ?`, matchID)

	// Guess score > 5250.
	reject("score too high", `
		INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus)
		VALUES (?, ?, ?, 0, 0, 0, 5251, 5000, 251)`, uuid.New(), roundID, gpID)

	// Accuracy > 5000.
	reject("accuracy too high", `
		INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus)
		VALUES (?, ?, ?, 0, 0, 0, 5001, 5001, 0)`, uuid.New(), roundID, gpID)

	// score != accuracy + bonus.
	reject("score components", `
		INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus)
		VALUES (?, ?, ?, 0, 0, 0, 100, 90, 0)`, uuid.New(), roundID, gpID)

	// Valid ranked guess with bonus at cap boundary.
	mustExec(t, db, `
		INSERT INTO guesses (id, round_id, game_player_id, latitude, longitude, distance_meters, score, accuracy_score, speed_bonus)
		VALUES (?, ?, ?, 0, 0, 0, 5250, 5000, 250)`, uuid.New(), roundID, gpID)

	// Party capacity/format mismatch.
	reject("party capacity", `
		INSERT INTO parties (id, format, capacity, leader_user_id, status, version)
		VALUES (?, 'duo', 4, ?, 'forming', 0)`, uuid.New(), userID)

	// Second active season rejected.
	reject("second active season", `
		INSERT INTO competitive_seasons (sequence, slug, name, status, starts_at, ends_at, initial_rating, elo_k, reset_factor_bps, top500_min_matches)
		VALUES (999001, 'season-test-dup', 'Dup', 'active', now(), now() + interval '84 days', 800, 32, 5000, 25)`)

	// Upload team_chat requires context.
	reject("upload team_chat context", `
		INSERT INTO uploads (id, owner_user_id, file_name, content_type, size_bytes, storage_key, status, expires_at, purpose, sanitization_status)
		VALUES (?, ?, 'x.jpg', 'image/jpeg', 10, ?, 'pending', now() + interval '1 hour', 'team_chat', 'pending')`,
		uuid.New(), userID, "raw-"+uuid.NewString())
}

func TestCasualRankedMigration_GuardedDown(t *testing.T) {
	db := testAppDB(t)
	ensureCasualRankedMigration(t, db)

	path := filepath.Join(migrationsDir(t), "00019_casual_ranked_team_modes.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	down := extractGooseSection(string(raw), "Down")
	if down == "" {
		t.Fatal("empty Down section")
	}

	// Irreversible path: non-legacy mode or party/competitive/chat rows.
	now := time.Now().UTC()
	userID := uuid.New()
	mapID := uuid.New()
	gameID := uuid.New()
	matchID := uuid.New()
	mustExec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, 'x', 'user', 'active', ?, ?)`, userID, userID.String()+"@example.test", now, now)
	mustExec(t, db, `INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at)
		VALUES (?, 'DownGuard', 'en', ?, ?)`, userID, now, now)
	mustExec(t, db, `INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Down Guard Map', 'public', 'free', 'mixed', 'active', ?, ?)`,
		mapID, "crm-down-"+mapID.String()[:8], now, now)
	mustExec(t, db, `INSERT INTO games (id, mode, status, map_id, created_by_user_id, round_count, scoring_version, total_score, created_at, updated_at)
		VALUES (?, 'solo', 'pending', ?, ?, 5, 1, 0, ?, ?)`, gameID, mapID, userID, now, now)
	mustExec(t, db, `INSERT INTO matches (
		id, formation_key, game_id, mode, status, matched_at, created_at, updated_at,
		playlist, format, team_size, season_id, last_activity_at
	) VALUES (?, ?, ?, 'casual_solo', 'matched', ?, ?, ?, 'casual', 'solo', 1, NULL, ?)`,
		matchID, "crm-down-"+matchID.String(), gameID, now, now, now, now)

	t.Cleanup(func() {
		cleanupMigrationFixtures(t, db, []uuid.UUID{matchID}, []uuid.UUID{userID})
		_ = db.Exec(`DELETE FROM maps WHERE id = ?`, mapID)
	})

	err = execGooseSQL(t, db, down)
	if err == nil {
		t.Fatal("expected guarded down to fail while non-legacy mode data exists")
	}
	if !strings.Contains(err.Error(), "irreversible") {
		t.Fatalf("expected irreversible error, got: %v", err)
	}

	// Schema must still be present after failed down.
	if !hasTable(t, db, "parties") || !hasColumn(t, db, "matches", "playlist") {
		t.Fatal("guarded down partially applied schema changes; expected full abort")
	}
}

func TestCasualRankedMigration_ReversibleDownWhenEmpty(t *testing.T) {
	db := testAppDB(t)
	if !migration00019Applied(t, db) {
		// Apply only if the shared DB is still pre-00019; otherwise this test
		// would destroy shared schema used by other packages.
		ensureCasualRankedMigration(t, db)
	}

	// Only run the full down/up cycle when the database has no irreversible rows.
	var irreversible bool
	checks := []string{
		`SELECT EXISTS (SELECT 1 FROM parties LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM competitive_standings LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM competitive_rating_changes LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM team_messages LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM match_mutes LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM team_message_reports LIMIT 1)`,
		`SELECT EXISTS (SELECT 1 FROM matches WHERE mode <> 'ranked_standard' LIMIT 1)`,
	}
	for _, q := range checks {
		var exists bool
		if err := db.Raw(q).Scan(&exists).Error; err != nil {
			t.Fatalf("irreversibility probe failed: %v", err)
		}
		if exists {
			irreversible = true
			break
		}
	}
	if irreversible {
		t.Skip("shared database has irreversible 00019 data; skipping full down/up cycle")
	}

	path := filepath.Join(migrationsDir(t), "00019_casual_ranked_team_modes.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	down := extractGooseSection(string(raw), "Down")
	up := extractGooseSection(string(raw), "Up")

	if err := execGooseSQL(t, db, down); err != nil {
		t.Fatalf("expected clean down to succeed: %v", err)
	}
	if hasTable(t, db, "parties") || hasColumn(t, db, "matches", "playlist") {
		t.Fatal("down did not remove 00019 schema")
	}
	// Re-apply so the shared DB remains usable for other tests in this package.
	if err := execGooseSQL(t, db, up); err != nil {
		t.Fatalf("re-apply up after clean down: %v", err)
	}
	if !migration00019Applied(t, db) {
		t.Fatal("expected schema restored after re-apply")
	}
}
