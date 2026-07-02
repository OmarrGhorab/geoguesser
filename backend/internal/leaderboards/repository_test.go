package leaderboards_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/leaderboards"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"gorm.io/gorm"
)

func setupLeaderboardsRepositoryTest(t *testing.T) (*leaderboards.Repository, *gorm.DB) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for leaderboard repository integration tests")
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("postgres connection failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	return leaderboards.NewRepository(db), db.WithContext(context.Background())
}

func TestRepositoryMaterializeCompletedGameRanksAndRetries(t *testing.T) {
	repo, db := setupLeaderboardsRepositoryTest(t)
	ctx := context.Background()

	mapID := seedLeaderboardMap(t, db)
	firstUser := seedLeaderboardUser(t, db, "active", "First")
	secondUser := seedLeaderboardUser(t, db, "active", "Second")
	disabledUser := seedLeaderboardUser(t, db, "disabled", "Hidden")
	now := time.Now().UTC().Truncate(time.Second)

	lowGameID := seedLeaderboardGame(t, db, firstUser, mapID, 1000, now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	highGameID := seedLeaderboardGame(t, db, firstUser, mapID, 2000, now.Add(-2*time.Hour), now.Add(-time.Hour))
	secondGameID := seedLeaderboardGame(t, db, secondUser, mapID, 1500, now.Add(-3*time.Hour), now.Add(-2*time.Hour))
	hiddenGameID := seedLeaderboardGame(t, db, disabledUser, mapID, 5000, now.Add(-3*time.Hour), now.Add(-2*time.Hour))

	for _, gameID := range []uuid.UUID{lowGameID, secondGameID, hiddenGameID, highGameID, lowGameID} {
		if _, err := repo.MaterializeCompletedGame(ctx, gameID); err != nil {
			t.Fatalf("MaterializeCompletedGame(%s) failed: %v", gameID, err)
		}
	}

	global, err := repo.EnsureGlobalLeaderboard(ctx)
	if err != nil {
		t.Fatalf("EnsureGlobalLeaderboard failed: %v", err)
	}
	entries, err := repo.ListGeneralEntries(ctx, global.ID, 10, "")
	if err != nil {
		t.Fatalf("ListGeneralEntries failed: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}
	if entries[0].UserID != firstUser || entries[0].Score != 2000 || entries[0].Rank != 1 || entries[0].GamesPlayed != 2 {
		t.Fatalf("first entry = %+v, want first user score 2000 rank 1 games_played 2", entries[0])
	}
	if entries[1].UserID != secondUser || entries[1].Score != 1500 || entries[1].Rank != 2 || entries[1].GamesPlayed != 1 {
		t.Fatalf("second entry = %+v, want second user score 1500 rank 2 games_played 1", entries[1])
	}
	for _, entry := range entries {
		if entry.UserID == disabledUser {
			t.Fatal("disabled user should not appear in leaderboard entries")
		}
	}

	svc := leaderboards.NewService(repo, nil, clock.Fixed(now), nil, 0, nil)
	firstPage, err := svc.GetGlobal(ctx, 1, "")
	if err != nil {
		t.Fatalf("GetGlobal first page failed: %v", err)
	}
	if len(firstPage.Data) != 1 || firstPage.Data[0].UserID != firstUser {
		t.Fatalf("first page = %+v, want first user only", firstPage.Data)
	}
	if firstPage.Page.NextCursor == nil {
		t.Fatal("expected first page next cursor")
	}
	thirdUser := seedLeaderboardUser(t, db, "active", "Third")
	newHighGameID := seedLeaderboardGame(t, db, thirdUser, mapID, 3000, now.Add(-30*time.Minute), now.Add(-15*time.Minute))
	if _, err := repo.MaterializeCompletedGame(ctx, newHighGameID); err != nil {
		t.Fatalf("MaterializeCompletedGame(new high score) failed: %v", err)
	}
	secondPage, err := svc.GetGlobal(ctx, 1, *firstPage.Page.NextCursor)
	if err != nil {
		t.Fatalf("GetGlobal second page failed: %v", err)
	}
	if len(secondPage.Data) != 1 || secondPage.Data[0].UserID != secondUser {
		t.Fatalf("second page after higher insert = %+v, want original second user", secondPage.Data)
	}
}

func seedLeaderboardUser(t *testing.T, db *gorm.DB, status string, namePrefix string) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	user := auth.User{
		Email:  "leaderboard-test-" + suffix + "@example.com",
		Status: status,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	profile := auth.UserProfile{
		UserID:      user.ID,
		DisplayName: namePrefix + " " + suffix[:8],
		Locale:      "en",
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create user profile failed: %v", err)
	}
	return user.ID
}

func seedLeaderboardMap(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	gameMap := maps.Map{
		Slug:       "leaderboard-test-" + suffix,
		Name:       "Leaderboard Test " + suffix,
		Visibility: "public",
		AccessTier: "free",
		Difficulty: "mixed",
		Status:     "active",
	}
	if err := db.Create(&gameMap).Error; err != nil {
		t.Fatalf("create map failed: %v", err)
	}
	location := locations.Location{
		Latitude:    30,
		Longitude:   31,
		CountryCode: "EG",
		Difficulty:  "easy",
		Provider:    "image",
		ProviderRef: fmt.Sprintf("leaderboard-test-%s", suffix),
		Status:      "active",
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("create location failed: %v", err)
	}
	link := maps.MapLocation{MapID: gameMap.ID, LocationID: location.ID, SelectionWeight: 1}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("create map location failed: %v", err)
	}
	return gameMap.ID
}

func seedLeaderboardGame(t *testing.T, db *gorm.DB, userID uuid.UUID, mapID uuid.UUID, totalScore int, startedAt time.Time, completedAt time.Time) uuid.UUID {
	t.Helper()
	game := games.Game{
		Mode:        games.GameModeSolo,
		Status:      games.GameStatusCompleted,
		MapID:       mapID,
		RoundCount:  3,
		TotalScore:  totalScore,
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
	}
	if err := db.Create(&game).Error; err != nil {
		t.Fatalf("create game failed: %v", err)
	}
	player := games.GamePlayer{
		GameID:      game.ID,
		UserID:      &userID,
		DisplayName: "player",
		Role:        games.PlayerRolePlayer,
		Status:      games.PlayerStatusActive,
		TotalScore:  totalScore,
	}
	if err := db.Create(&player).Error; err != nil {
		t.Fatalf("create game player failed: %v", err)
	}
	return game.ID
}
