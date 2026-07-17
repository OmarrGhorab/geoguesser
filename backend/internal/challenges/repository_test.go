package challenges

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
)

func TestGetDefaultActiveMapIDSkipsMapsWithoutEnoughLocations(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for repository integration test")
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	ctx := context.Background()
	eligibleMapID := uuid.New()
	ineligibleMapID := uuid.New()
	unplayableMapID := uuid.New()
	eligibleCreatedAt := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	ineligibleCreatedAt := eligibleCreatedAt.Add(time.Hour)
	unplayableCreatedAt := ineligibleCreatedAt.Add(time.Hour)

	insertMap := func(id uuid.UUID, slug string, createdAt time.Time) {
		t.Helper()
		if err := tx.WithContext(ctx).Exec(`
			INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
			VALUES (?, ?, ?, 'public', 'free', 'mixed', 'active', ?, ?)
		`, id, slug, slug, createdAt, createdAt).Error; err != nil {
			t.Fatalf("insert map: %v", err)
		}
	}
	insertMap(eligibleMapID, "eligible-"+eligibleMapID.String(), eligibleCreatedAt)
	insertMap(ineligibleMapID, "ineligible-"+ineligibleMapID.String(), ineligibleCreatedAt)
	insertMap(unplayableMapID, "unplayable-"+unplayableMapID.String(), unplayableCreatedAt)

	insertLocations := func(mapID uuid.UUID, count int, provider string) {
		t.Helper()
		for index := range count {
			locationID := uuid.New()
			providerRef := locationID.String()
			if err := tx.WithContext(ctx).Exec(`
				INSERT INTO locations (
					id, latitude, longitude, country_code, difficulty, provider,
					provider_ref, status, created_at, updated_at
				)
				VALUES (?, 0, 0, 'US', 'medium', ?, ?, 'active', ?, ?)
			`, locationID, provider, providerRef, eligibleCreatedAt, eligibleCreatedAt).Error; err != nil {
				t.Fatalf("insert location %d: %v", index, err)
			}
			if err := tx.WithContext(ctx).Exec(`
				INSERT INTO map_locations (map_id, location_id, selection_weight, created_at)
				VALUES (?, ?, 1, ?)
			`, mapID, locationID, eligibleCreatedAt).Error; err != nil {
				t.Fatalf("link location %d: %v", index, err)
			}
		}
	}
	insertLocations(eligibleMapID, DefaultRoundCount, "google_street_view")
	insertLocations(ineligibleMapID, DefaultRoundCount-1, "google_street_view")
	insertLocations(unplayableMapID, DefaultRoundCount, "test")

	repo := NewRepository(tx)
	got, err := repo.GetDefaultActiveMapID(ctx, DefaultRoundCount)
	if err != nil {
		t.Fatalf("GetDefaultActiveMapID() error = %v", err)
	}
	if got != eligibleMapID {
		t.Fatalf("GetDefaultActiveMapID() = %s, want eligible map %s", got, eligibleMapID)
	}
}

func TestStartAttemptGamePreservesCompletedAttempt(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for repository integration test")
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	now := time.Now().UTC()
	challengeDate := time.Date(2098, 12, 30, 0, 0, 0, 0, time.UTC)
	resetEndsAt := challengeDate.Add(24 * time.Hour)
	mapRow := maps.Map{
		ID: uuid.New(), Slug: "completed-attempt-" + uuid.NewString(), Name: "Completed Attempt",
		Visibility: "public", AccessTier: "free", Difficulty: "mixed", Status: "active",
	}
	if err := tx.Create(&mapRow).Error; err != nil {
		t.Fatalf("insert map: %v", err)
	}
	settings, err := json.Marshal(SettingsSnapshot{RoundCount: DefaultRoundCount, MovementRules: "standard", ScoringVersion: DefaultScoringVersion})
	if err != nil {
		t.Fatalf("encode settings: %v", err)
	}
	challenge := Challenge{
		Type: TypeDaily, Seed: uuid.NewString(), ChallengeDate: &challengeDate,
		ResetStartsAt: &challengeDate, ResetEndsAt: &resetEndsAt, MapID: mapRow.ID,
		SettingsSnapshot: settings, Status: StatusActive,
	}
	if err := tx.Create(&challenge).Error; err != nil {
		t.Fatalf("insert challenge: %v", err)
	}
	guest := "completed-guest-" + uuid.NewString()
	selected := make([]maps.SelectedLocation, DefaultRoundCount)
	for index := range DefaultRoundCount {
		locationID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO locations (
				id, latitude, longitude, country_code, difficulty, provider,
				provider_ref, status, created_at, updated_at
			) VALUES (?, 0, 0, 'US', 'medium', 'test', ?, 'active', ?, ?)
		`, locationID, locationID.String(), now, now).Error; err != nil {
			t.Fatalf("insert location %d: %v", index, err)
		}
		selected[index] = maps.SelectedLocation{ID: locationID}
	}
	repo := NewRepository(tx)
	attempt, game, err := repo.CreateAttemptWithGame(
		context.Background(), challenge, ownerIdentity{guestHash: &guest, displayName: "Guest"},
		selected, SettingsSnapshot{RoundCount: DefaultRoundCount, MovementRules: "standard", ScoringVersion: DefaultScoringVersion}, now,
	)
	if err != nil {
		t.Fatalf("CreateAttemptWithGame() error = %v", err)
	}
	if game.Mode != games.GameModeDaily {
		t.Fatalf("game mode = %q, want %q", game.Mode, games.GameModeDaily)
	}
	locationIDs := make([]uuid.UUID, len(selected))
	for index := range selected {
		locationIDs[index] = selected[index].ID
	}
	playable, err := repo.AreLocationsPlayable(context.Background(), locationIDs)
	if err != nil {
		t.Fatalf("AreLocationsPlayable() error = %v", err)
	}
	if playable {
		t.Fatal("test-provider locations must not be treated as playable")
	}

	replacementMap := maps.Map{
		ID: uuid.New(), Slug: "replacement-" + uuid.NewString(), Name: "Playable Replacement",
		Visibility: "public", AccessTier: "free", Difficulty: "mixed", Status: "active",
	}
	if err := tx.Create(&replacementMap).Error; err != nil {
		t.Fatalf("insert replacement map: %v", err)
	}
	replacement := make([]maps.SelectedLocation, DefaultRoundCount)
	replacementIDs := make([]uuid.UUID, DefaultRoundCount)
	for index := range DefaultRoundCount {
		locationID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO locations (
				id, latitude, longitude, country_code, difficulty, provider,
				provider_ref, status, created_at, updated_at
			) VALUES (?, 0, 0, 'US', 'medium', 'google_street_view', ?, 'active', ?, ?)
		`, locationID, locationID.String(), now, now).Error; err != nil {
			t.Fatalf("insert replacement location %d: %v", index, err)
		}
		replacement[index] = maps.SelectedLocation{ID: locationID}
		replacementIDs[index] = locationID
	}
	playable, err = repo.AreLocationsPlayable(context.Background(), replacementIDs)
	if err != nil {
		t.Fatalf("AreLocationsPlayable(replacement) error = %v", err)
	}
	if !playable {
		t.Fatal("google street view locations should be playable")
	}
	repairedGame, err := repo.RepairUnplayedAttemptGame(context.Background(), attempt.ID, replacementMap.ID, replacement, now)
	if err != nil {
		t.Fatalf("RepairUnplayedAttemptGame() error = %v", err)
	}
	if repairedGame.MapID != replacementMap.ID {
		t.Fatalf("repaired map = %s, want %s", repairedGame.MapID, replacementMap.ID)
	}
	var repairedRounds []games.Round
	if err := tx.Where("game_id = ?", game.ID).Order("round_number ASC").Find(&repairedRounds).Error; err != nil {
		t.Fatalf("load repaired rounds: %v", err)
	}
	for index := range repairedRounds {
		if repairedRounds[index].LocationID != replacement[index].ID {
			t.Fatalf("round %d location = %s, want %s", index+1, repairedRounds[index].LocationID, replacement[index].ID)
		}
	}
	if err := tx.Model(&games.Game{}).Where("id = ?", game.ID).Updates(map[string]any{
		"status": games.GameStatusCompleted, "started_at": now, "completed_at": now,
	}).Error; err != nil {
		t.Fatalf("complete game: %v", err)
	}
	if err := tx.Model(&ChallengeAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{
		"status": AttemptStatusCompleted, "completed_at": now,
	}).Error; err != nil {
		t.Fatalf("complete attempt: %v", err)
	}

	gotAttempt, gotGame, err := repo.StartAttemptGame(context.Background(), attempt.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("StartAttemptGame() error = %v", err)
	}
	if gotAttempt.Status != AttemptStatusCompleted {
		t.Fatalf("attempt status = %q, want %q", gotAttempt.Status, AttemptStatusCompleted)
	}
	if gotGame.Status != games.GameStatusCompleted {
		t.Fatalf("game status = %q, want %q", gotGame.Status, games.GameStatusCompleted)
	}
}

func TestClaimMissionAwardsXPExactlyOnceAndRollsBackWithoutProfile(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for repository integration test")
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

	ctx := context.Background()
	now := time.Date(2097, 7, 17, 12, 0, 0, 0, time.UTC)
	mission := DefaultMissions(now, 0)[0]
	mission.ID = uuid.New()
	mission.Code += ":" + uuid.NewString()
	if err := tx.Create(&mission).Error; err != nil {
		t.Fatalf("insert mission: %v", err)
	}

	insertUser := func(withProfile bool) uuid.UUID {
		t.Helper()
		userID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO users (id, email, password_hash, status, created_at, updated_at)
			VALUES (?, ?, 'test-password-hash', 'active', ?, ?)
		`, userID, "mission-"+userID.String()+"@example.test", now, now).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
		if withProfile {
			if err := tx.Exec(`
				INSERT INTO user_profiles (
					user_id, display_name, experience_points, level, created_at, updated_at
				) VALUES (?, 'Mission Tester', 0, 1, ?, ?)
			`, userID, now, now).Error; err != nil {
				t.Fatalf("insert profile: %v", err)
			}
		}
		completedAt := now.Add(-time.Minute)
		progress := MissionProgress{
			MissionID: mission.ID, OwnerUserID: &userID,
			CurrentValue: mission.TargetValue, TargetValue: mission.TargetValue,
			Status: "completed", CompletedAt: &completedAt, UpdatedAt: completedAt,
		}
		if err := tx.Create(&progress).Error; err != nil {
			t.Fatalf("insert mission progress: %v", err)
		}
		return userID
	}

	repo := NewRepository(tx)
	userID := insertUser(true)
	owner := ownerIdentity{userID: &userID}
	for attempt := range 2 {
		if _, _, err := repo.ClaimMissionAndAwardXP(ctx, mission.ID, owner, now.Add(time.Duration(attempt)*time.Second)); err != nil {
			t.Fatalf("claim attempt %d: %v", attempt+1, err)
		}
	}
	var experiencePoints int64
	if err := tx.Raw("SELECT experience_points FROM user_profiles WHERE user_id = ?", userID).Scan(&experiencePoints).Error; err != nil {
		t.Fatalf("load experience points: %v", err)
	}
	if experiencePoints != int64(mission.RewardXP) {
		t.Fatalf("experience points = %d, want %d", experiencePoints, mission.RewardXP)
	}

	missingProfileUserID := insertUser(false)
	_, _, err = repo.ClaimMissionAndAwardXP(ctx, mission.ID, ownerIdentity{userID: &missingProfileUserID}, now)
	if err == nil {
		t.Fatal("claim without profile error = nil")
	}
	var claimedCount int64
	if err := tx.Raw(`
		SELECT COUNT(*) FROM mission_progress
		WHERE mission_id = ? AND owner_user_id = ? AND claimed_at IS NOT NULL
	`, mission.ID, missingProfileUserID).Scan(&claimedCount).Error; err != nil {
		t.Fatalf("load rolled back progress: %v", err)
	}
	if claimedCount != 0 {
		t.Fatalf("claimed progress rows = %d after rolled back XP award", claimedCount)
	}
}
