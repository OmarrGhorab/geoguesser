package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMigration00020DeclaresModesIdempotencyAndProtectedDown(t *testing.T) {
	path := filepath.Join(migrationsDir(t), "00020_party_lobby_practice_modes.sql")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(contents)
	for _, required := range []string{
		"'party_lobby'", "'practice'", "'private_room'",
		"creation_idempotency_key", "rounds_game_creation_idempotency_uidx",
		"games_round_count_range", "mode = 'practice' AND round_count >= 1",
		"irreversible while party_lobby or practice games exist",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}

func TestMigration00020AllowsOpenEndedPracticeButKeepsFixedModesBounded(t *testing.T) {
	db := testAppDB(t)
	if !hasColumn(t, db, "rounds", "creation_idempotency_key") {
		t.Skip("migration 00020 is not applied")
	}
	now := time.Now().UTC()
	mapID := uuid.New()
	practiceID := uuid.New()
	slug := "practice-constraint-" + mapID.String()[:8]
	mustExec(t, db, `INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Practice Constraint', 'public', 'free', 'mixed', 'active', ?, ?)`, mapID, slug, now, now)
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM games WHERE id = ?`, practiceID).Error
		_ = db.Exec(`DELETE FROM maps WHERE id = ?`, mapID).Error
	})

	if err := db.Exec(`INSERT INTO games (id, mode, status, map_id, round_count, scoring_version, total_score, created_at, updated_at)
		VALUES (?, 'practice', 'active', ?, 1000, 1, 0, ?, ?)`, practiceID, mapID, now, now).Error; err != nil {
		t.Fatalf("Practice round_count=1000 should satisfy migration constraint: %v", err)
	}
	if err := db.Exec(`INSERT INTO games (id, mode, status, map_id, round_count, scoring_version, total_score, created_at, updated_at)
		VALUES (?, 'solo', 'active', ?, 11, 1, 0, ?, ?)`, uuid.New(), mapID, now, now).Error; err == nil {
		t.Fatal("solo round_count=11 should remain rejected")
	}
	if err := db.Exec(`INSERT INTO games (id, mode, status, map_id, round_count, scoring_version, total_score, created_at, updated_at)
		VALUES (?, 'unknown_mode', 'active', ?, 1, 1, 0, ?, ?)`, uuid.New(), mapID, now, now).Error; err == nil {
		t.Fatal("unknown game mode should remain rejected")
	}
}
