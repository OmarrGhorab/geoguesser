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

	// Isolate from leftover global entries left by other packages on a shared DATABASE_URL.
	global, err := repo.EnsureGlobalLeaderboard(ctx)
	if err != nil {
		t.Fatalf("EnsureGlobalLeaderboard failed: %v", err)
	}
	if err := db.Exec(`DELETE FROM leaderboard_entries WHERE leaderboard_id = ?`, global.ID).Error; err != nil {
		t.Fatalf("cleanup global leaderboard entries: %v", err)
	}

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

func TestRepositoryListFriendsEntriesCohortFiltering(t *testing.T) {
	repo, db := setupLeaderboardsRepositoryTest(t)
	ctx := context.Background()

	// Isolate global board for deterministic ranks.
	global, err := repo.EnsureGlobalLeaderboard(ctx)
	if err != nil {
		t.Fatalf("EnsureGlobalLeaderboard: %v", err)
	}
	if err := db.Exec(`DELETE FROM leaderboard_entries WHERE leaderboard_id = ?`, global.ID).Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	mapID := seedLeaderboardMap(t, db)
	viewer := seedLeaderboardUser(t, db, "active", "Viewer")
	friend := seedLeaderboardUser(t, db, "active", "Friend")
	pending := seedLeaderboardUser(t, db, "active", "Pending")
	blocked := seedLeaderboardUser(t, db, "active", "Blocked")
	stranger := seedLeaderboardUser(t, db, "active", "Stranger")
	inactiveFriend := seedLeaderboardUser(t, db, "active", "InactiveFriend")
	now := time.Now().UTC().Truncate(time.Second)

	// Friendships: accepted, pending, blocked.
	mustFriend := func(a, b uuid.UUID, status, requestedBy string) {
		t.Helper()
		ua, ub := a, b
		if bytesLessUUID(ub, ua) {
			ua, ub = ub, ua
		}
		req := a
		if requestedBy == "b" {
			req = b
		}
		acceptedAt := any(nil)
		var blockedBy any
		if status == "accepted" {
			acceptedAt = now
		}
		if status == "blocked" {
			blockedBy = a
		}
		if err := db.Exec(`
			INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, blocked_by_user_id, accepted_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New(), ua, ub, req, status, blockedBy, acceptedAt, now, now).Error; err != nil {
			t.Fatalf("insert friendship: %v", err)
		}
	}
	mustFriend(viewer, friend, "accepted", "a")
	mustFriend(viewer, pending, "pending", "a")
	mustFriend(viewer, blocked, "blocked", "a")
	mustFriend(viewer, inactiveFriend, "accepted", "a")
	if err := db.Exec(`UPDATE users SET status = 'disabled' WHERE id = ?`, inactiveFriend).Error; err != nil {
		t.Fatalf("disable friend: %v", err)
	}

	// Scores: stranger has highest global score but must be excluded from friends cohort.
	for _, item := range []struct {
		user  uuid.UUID
		score int
	}{
		{viewer, 100},
		{friend, 200},
		{pending, 500},
		{blocked, 600},
		{stranger, 999},
		{inactiveFriend, 800},
	} {
		gameID := seedLeaderboardGame(t, db, item.user, mapID, item.score, now.Add(-time.Hour), now)
		if _, err := repo.MaterializeCompletedGame(ctx, gameID); err != nil {
			t.Fatalf("materialize %s: %v", item.user, err)
		}
	}

	entries, err := repo.ListFriendsEntries(ctx, viewer, 20, "")
	if err != nil {
		t.Fatalf("ListFriendsEntries: %v", err)
	}
	ids := map[uuid.UUID]int{}
	for _, e := range entries {
		ids[e.UserID] = e.Rank
	}
	if _, ok := ids[viewer]; !ok {
		t.Fatal("viewer missing from friends cohort")
	}
	if _, ok := ids[friend]; !ok {
		t.Fatal("accepted friend missing from friends cohort")
	}
	for _, excluded := range []uuid.UUID{pending, blocked, stranger, inactiveFriend} {
		if _, ok := ids[excluded]; ok {
			t.Fatalf("excluded user %s appeared in friends cohort", excluded)
		}
	}
	// Friend (200) ranks above viewer (100) inside cohort.
	if ids[friend] != 1 || ids[viewer] != 2 {
		t.Fatalf("cohort ranks = friend %d viewer %d, want 1 and 2", ids[friend], ids[viewer])
	}
}

func bytesLessUUID(a, b uuid.UUID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
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
