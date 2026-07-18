package matchmaking_test

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func teamFormationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping team formation integration tests")
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
	// Require migration 00019 columns.
	if !db.Migrator().HasColumn(&matchmaking.Match{}, "playlist") ||
		!db.Migrator().HasColumn(&matchmaking.Match{}, "team_size") ||
		!db.Migrator().HasColumn(&matchmaking.MatchPlayer{}, "team_slot") {
		t.Skip("migration 00019 columns missing; run goose up for 00019_casual_ranked_team_modes.sql")
	}
	return db
}

func seedUsers(t *testing.T, db *gorm.DB, n int) []uuid.UUID {
	t.Helper()
	now := time.Now().UTC()
	ids := make([]uuid.UUID, n)
	for i := 0; i < n; i++ {
		ids[i] = uuid.New()
		if err := db.Exec(
			`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
			ids[i], ids[i].String()+"@example.test", now, now,
		).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if err := db.Exec(
			`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, ?, 'en', ?, ?)`,
			ids[i], "Player"+ids[i].String()[:8], now, now,
		).Error; err != nil {
			t.Fatalf("seed profile: %v", err)
		}
	}
	return ids
}

func seedMapLocations(t *testing.T, db *gorm.DB, count int) (mapID uuid.UUID, locationIDs []uuid.UUID) {
	t.Helper()
	now := time.Now().UTC()
	mapID = uuid.New()
	if err := db.Exec(`
		INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, 'Team Formation Map', 'public', 'free', 'mixed', 'active', ?, ?)
	`, mapID, "team-form-"+mapID.String()[:8], now, now).Error; err != nil {
		t.Fatalf("seed map: %v", err)
	}
	locationIDs = make([]uuid.UUID, count)
	for i := range locationIDs {
		locationIDs[i] = uuid.New()
		if err := db.Exec(`
			INSERT INTO locations (id, country_code, latitude, longitude, difficulty, provider, provider_ref, status, created_at, updated_at)
			VALUES (?, 'US', 0, 0, 'easy', 'test', ?, 'active', ?, ?)
		`, locationIDs[i], locationIDs[i].String(), now, now).Error; err != nil {
			t.Fatalf("seed location: %v", err)
		}
		if err := db.Exec(
			`INSERT INTO map_locations (map_id, location_id, selection_weight, created_at) VALUES (?, ?, 1, ?)`,
			mapID, locationIDs[i], now,
		).Error; err != nil {
			t.Fatalf("seed map_location: %v", err)
		}
	}
	return mapID, locationIDs
}

func timerPtr(v int) *int { return &v }

func TestTeamFormation_1v1CardinalityAndSlots(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	users := seedUsers(t, db, 2)
	mapID, locs := seedMapLocations(t, db, 5)
	matchedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	key := "team-1v1-" + uuid.NewString()

	result, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: key,
		Mode:         matchmaking.ModeRankedSolo,
		Playlist:     matchmaking.PlaylistRanked,
		Format:       matchmaking.FormatSolo,
		TeamSize:     1,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		StartDelay:   5 * time.Second,
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("CreateTeamFormationBundle: %v", err)
	}
	if result.Match.Playlist != matchmaking.PlaylistRanked || result.Match.Format != matchmaking.FormatSolo || result.Match.TeamSize != 1 {
		t.Fatalf("match parts = playlist=%s format=%s size=%d", result.Match.Playlist, result.Match.Format, result.Match.TeamSize)
	}
	if result.Match.SeasonID == nil {
		t.Fatal("ranked match requires season_id")
	}
	if result.Match.TeamOneScore != 0 || result.Match.TeamTwoScore != 0 {
		t.Fatalf("team scores must start at 0")
	}

	var slots []struct {
		UserID   uuid.UUID
		TeamSlot int
	}
	if err := db.Table("match_players").Select("user_id, team_slot").Where("match_id = ?", result.Match.ID).Order("team_slot ASC, user_id ASC").Scan(&slots).Error; err != nil {
		t.Fatalf("load slots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("participants = %d, want 2", len(slots))
	}
	if slots[0].TeamSlot != 1 || slots[1].TeamSlot != 2 {
		t.Fatalf("slots = %+v, want 1 then 2", slots)
	}

	var gpSlots []int
	if err := db.Table("game_players").Select("team_slot").Where("game_id = ?", result.Match.GameID).Order("team_slot ASC").Scan(&gpSlots).Error; err != nil {
		t.Fatalf("game player slots: %v", err)
	}
	if len(gpSlots) != 2 || gpSlots[0] != 1 || gpSlots[1] != 2 {
		t.Fatalf("game_players team_slot = %v, want [1 2]", gpSlots)
	}
}

func TestTeamFormation_2v2And4v4Cardinality(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	mapID, locs := seedMapLocations(t, db, 5)
	matchedAt := time.Now().UTC()

	cases := []struct {
		name     string
		mode     string
		format   string
		teamSize int
	}{
		{"2v2", matchmaking.ModeRankedDuo, matchmaking.FormatDuo, 2},
		{"4v4", matchmaking.ModeRankedSquad, matchmaking.FormatSquad, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			users := seedUsers(t, db, tc.teamSize*2)
			teamOne := users[:tc.teamSize]
			teamTwo := users[tc.teamSize:]
			result, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
				FormationKey: "team-" + tc.name + "-" + uuid.NewString(),
				Mode:         tc.mode,
				Playlist:     matchmaking.PlaylistRanked,
				Format:       tc.format,
				TeamSize:     tc.teamSize,
				MapID:        mapID,
				RoundCount:   5,
				TimerSeconds: timerPtr(60),
				StartDelay:   3 * time.Second,
				TeamOne:      teamOne,
				TeamTwo:      teamTwo,
				LocationIDs:  locs,
				MatchedAt:    matchedAt,
			})
			if err != nil {
				t.Fatalf("form: %v", err)
			}
			if result.Match.TeamSize != tc.teamSize || result.Match.Format != tc.format {
				t.Fatalf("match = size=%d format=%s", result.Match.TeamSize, result.Match.Format)
			}
			var count int64
			if err := db.Table("match_players").Where("match_id = ?", result.Match.ID).Count(&count).Error; err != nil {
				t.Fatalf("count: %v", err)
			}
			if int(count) != tc.teamSize*2 {
				t.Fatalf("participants = %d, want %d", count, tc.teamSize*2)
			}
			var slot1, slot2 int64
			_ = db.Table("match_players").Where("match_id = ? AND team_slot = 1", result.Match.ID).Count(&slot1).Error
			_ = db.Table("match_players").Where("match_id = ? AND team_slot = 2", result.Match.ID).Count(&slot2).Error
			if int(slot1) != tc.teamSize || int(slot2) != tc.teamSize {
				t.Fatalf("slot counts slot1=%d slot2=%d want %d each", slot1, slot2, tc.teamSize)
			}
		})
	}
}

func TestTeamFormation_DistinctUsersAndLocationsRequired(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	users := seedUsers(t, db, 2)
	mapID, locs := seedMapLocations(t, db, 5)
	matchedAt := time.Now().UTC()

	// Duplicate user across teams.
	_, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: "dup-user-" + uuid.NewString(),
		Mode:         matchmaking.ModeRankedSolo,
		TeamSize:     1,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[0]},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if !errors.Is(err, matchmaking.ErrInvalidRequest) {
		t.Fatalf("duplicate user err = %v, want ErrInvalidRequest", err)
	}

	// Unequal roster sizes.
	users4 := seedUsers(t, db, 3)
	_, err = repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: "unequal-" + uuid.NewString(),
		Mode:         matchmaking.ModeRankedDuo,
		TeamSize:     2,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users4[0], users4[1]},
		TeamTwo:      []uuid.UUID{users4[2]},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if !errors.Is(err, matchmaking.ErrInvalidRequest) {
		t.Fatalf("unequal roster err = %v, want ErrInvalidRequest", err)
	}

	// Duplicate location IDs.
	dupLocs := []uuid.UUID{locs[0], locs[0], locs[1], locs[2], locs[3]}
	_, err = repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: "dup-loc-" + uuid.NewString(),
		Mode:         matchmaking.ModeRankedSolo,
		TeamSize:     1,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  dupLocs,
		MatchedAt:    matchedAt,
	})
	if !errors.Is(err, matchmaking.ErrInvalidRequest) {
		t.Fatalf("duplicate locations err = %v, want ErrInvalidRequest", err)
	}
}

func TestTeamFormation_TransactionRollbackOnFailure(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	users := seedUsers(t, db, 2)
	mapID, locs := seedMapLocations(t, db, 5)
	matchedAt := time.Now().UTC()

	// Inactive user forces ErrAccountIneligible after locks — nothing durable should remain.
	if err := db.Exec(`UPDATE users SET status = 'disabled' WHERE id = ?`, users[1]).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	key := "rollback-" + uuid.NewString()
	_, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: key,
		Mode:         matchmaking.ModeRankedSolo,
		TeamSize:     1,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if !errors.Is(err, matchmaking.ErrAccountIneligible) {
		t.Fatalf("err = %v, want ErrAccountIneligible", err)
	}
	var matchCount int64
	if err := db.Table("matches").Where("formation_key = ?", key).Count(&matchCount).Error; err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if matchCount != 0 {
		t.Fatalf("match rows after rollback = %d, want 0", matchCount)
	}
	// No leftover active assignment for the eligible user.
	asg, err := repo.FindActiveAssignment(t.Context(), users[0])
	if err != nil {
		t.Fatalf("assignment: %v", err)
	}
	if asg != nil {
		t.Fatalf("unexpected assignment after rollback: %+v", asg)
	}
}

func TestTeamFormation_LegacyAliasRecoveryAndIdempotentKeys(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	mapID, userA, userB, locs := seedFormationFixtures(t, db)
	matchedAt := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)
	key := "legacy-alias-" + uuid.NewString()

	first, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: key,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("legacy create: %v", err)
	}
	if first.Match.Mode != matchmaking.ModeRankedStandard {
		t.Fatalf("mode = %q, want ranked_standard", first.Match.Mode)
	}
	if first.Match.Playlist != matchmaking.PlaylistRanked || first.Match.Format != matchmaking.FormatSolo || first.Match.TeamSize != 1 {
		t.Fatalf("legacy parts incomplete: %+v", first.Match)
	}
	// Team slots assigned for legacy pair path.
	var slots []int
	if err := db.Table("match_players").Select("team_slot").Where("match_id = ?", first.Match.ID).Order("team_slot").Scan(&slots).Error; err != nil {
		t.Fatalf("slots: %v", err)
	}
	if len(slots) != 2 || slots[0] != 1 || slots[1] != 2 {
		t.Fatalf("legacy slots = %v", slots)
	}

	// Formation key lookup recovery.
	found, err := repo.FindMatchByFormationKey(t.Context(), key)
	if err != nil || found == nil || found.ID != first.Match.ID {
		t.Fatalf("FindMatchByFormationKey: found=%+v err=%v", found, err)
	}

	// Duplicate formation key is idempotent.
	second, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: key,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locs,
		MatchedAt:    matchedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if second.Match.ID != first.Match.ID || second.Match.GameID != first.Match.GameID {
		t.Fatalf("idempotent replay diverged: first=%+v second=%+v", first.Match, second.Match)
	}

	// Team formation key idempotency for ranked_solo.
	teamKey := "team-idem-" + uuid.NewString()
	users := seedUsers(t, db, 2)
	mapID2, locs2 := seedMapLocations(t, db, 5)
	a, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: teamKey,
		Mode:         matchmaking.ModeRankedSolo,
		TeamSize:     1,
		MapID:        mapID2,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  locs2,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("team first: %v", err)
	}
	b, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: teamKey,
		Mode:         matchmaking.ModeRankedSolo,
		TeamSize:     1,
		MapID:        mapID2,
		RoundCount:   5,
		TimerSeconds: timerPtr(60),
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  locs2,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("team replay: %v", err)
	}
	if a.Match.ID != b.Match.ID {
		t.Fatalf("team key not idempotent")
	}
}

func TestTeamFormation_CasualNullTimer(t *testing.T) {
	db := teamFormationDB(t)
	repo := matchmaking.NewRepository(db)
	users := seedUsers(t, db, 2)
	mapID, locs := seedMapLocations(t, db, 5)
	matchedAt := time.Now().UTC()

	result, err := repo.CreateTeamFormationBundle(t.Context(), matchmaking.TeamFormationInput{
		FormationKey: "casual-1v1-" + uuid.NewString(),
		Mode:         matchmaking.ModeCasualSolo,
		Playlist:     matchmaking.PlaylistCasual,
		Format:       matchmaking.FormatSolo,
		TeamSize:     1,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: nil,
		StartDelay:   0,
		TeamOne:      []uuid.UUID{users[0]},
		TeamTwo:      []uuid.UUID{users[1]},
		LocationIDs:  locs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		// games.mode check may still be pre-expansion on some environments.
		if isGamesModeCheckError(err) {
			t.Skipf("games_mode_check does not yet accept casual_*: %v", err)
		}
		t.Fatalf("casual form: %v", err)
	}
	if result.Match.Playlist != matchmaking.PlaylistCasual || result.Match.SeasonID != nil {
		t.Fatalf("casual match fields = playlist=%s season=%v", result.Match.Playlist, result.Match.SeasonID)
	}
	var timer *int
	if err := db.Table("games").Select("timer_seconds").Where("id = ?", result.Match.GameID).Scan(&timer).Error; err != nil {
		t.Fatalf("timer: %v", err)
	}
	if timer != nil {
		t.Fatalf("casual timer_seconds = %v, want NULL", *timer)
	}
	var endsAt sql.NullTime
	if err := db.Table("rounds").Select("ends_at").Where("game_id = ? AND round_number = 1", result.Match.GameID).Scan(&endsAt).Error; err != nil {
		t.Fatalf("ends_at: %v", err)
	}
	if endsAt.Valid {
		t.Fatalf("casual first round ends_at = %v, want NULL", endsAt.Time)
	}
}

func isGamesModeCheckError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "games_mode") || strings.Contains(msg, "violates check constraint")
}
