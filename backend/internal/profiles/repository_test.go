package profiles_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/raven/geoguess/backend/internal/auth"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"github.com/raven/geoguess/backend/internal/profiles"
)

func setupProfilesRepositoryTest(t *testing.T) (*profiles.Repository, *gorm.DB) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for integration tests")
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

	return profiles.NewRepository(db), db.WithContext(context.Background())
}

func seedUser(t *testing.T, db *gorm.DB, status string) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	u := auth.User{
		Email:  "profile-test-" + suffix + "@example.com",
		Status: status,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	p := auth.UserProfile{
		UserID:      u.ID,
		DisplayName: "Test User " + suffix[:8],
		Locale:      "en",
	}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create user profile failed: %v", err)
	}
	return u.ID
}

func seedProfileMap(t *testing.T, db *gorm.DB, count int) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	suffix := uuid.NewString()
	m := maps.Map{
		Slug:       "profile-test-" + suffix,
		Name:       "Profile Test " + suffix,
		Visibility: "public",
		AccessTier: "free",
		Difficulty: "mixed",
		Status:     "active",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create map failed: %v", err)
	}
	ids := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		loc := locations.Location{
			Latitude:    float64(30 + i),
			Longitude:   float64(31 + i),
			CountryCode: "EG",
			Difficulty:  "easy",
			Provider:    "image",
			ProviderRef: fmt.Sprintf("profile-repo-test-%s-%d", suffix, i),
			Status:      "active",
		}
		if err := db.Create(&loc).Error; err != nil {
			t.Fatalf("create location failed: %v", err)
		}
		link := maps.MapLocation{MapID: m.ID, LocationID: loc.ID, SelectionWeight: 1}
		if err := db.Create(&link).Error; err != nil {
			t.Fatalf("create map location failed: %v", err)
		}
		ids[i] = loc.ID
	}
	return m.ID, ids
}

func seedCompletedGame(t *testing.T, db *gorm.DB, userID uuid.UUID, mapID uuid.UUID, totalScore int, completedAt time.Time) {
	t.Helper()
	g := games.Game{
		Mode:        games.GameModeSolo,
		Status:      games.GameStatusCompleted,
		MapID:       mapID,
		RoundCount:  3,
		TotalScore:  totalScore,
		CompletedAt: &completedAt,
	}
	if err := db.Create(&g).Error; err != nil {
		t.Fatalf("create game failed: %v", err)
	}
	player := games.GamePlayer{
		GameID:      g.ID,
		UserID:      &userID,
		DisplayName: "player",
		Role:        games.PlayerRolePlayer,
		Status:      games.PlayerStatusActive,
		TotalScore:  totalScore,
	}
	if err := db.Create(&player).Error; err != nil {
		t.Fatalf("create game player failed: %v", err)
	}
}

func TestRepositoryGetCurrentProfile(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	userID := seedUser(t, db, "active")

	profile, err := repo.GetCurrentProfile(ctx, userID)
	if err != nil {
		t.Fatalf("GetCurrentProfile failed: %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile, got nil")
	}
	if profile.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, profile.UserID)
	}
	if profile.Preferences == nil {
		t.Fatal("expected non-nil preferences map")
	}
}

func TestRepositoryGetCurrentProfileMissing(t *testing.T) {
	repo, _ := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	profile, err := repo.GetCurrentProfile(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetCurrentProfile failed: %v", err)
	}
	if profile != nil {
		t.Fatal("expected nil profile for missing user")
	}
}

func TestRepositoryUpdateProfilePartial(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	userID := seedUser(t, db, "active")

	newName := "Updated Name"
	prefs := map[string]any{"theme": "dark"}
	updated, err := repo.UpdateProfile(ctx, userID, profiles.ProfileUpdate{
		HasDisplayName: true,
		DisplayName:    &newName,
		HasPreferences: true,
		Preferences:    ptrToPtrMap(prefs),
	})
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.DisplayName != newName {
		t.Fatalf("expected display name %q, got %q", newName, updated.DisplayName)
	}
	if updated.Preferences["theme"] != "dark" {
		t.Fatalf("expected preferences to be updated, got %+v", updated.Preferences)
	}

	// A second update that only touches locale must not clobber display_name.
	newLocale := "ar"
	updated2, err := repo.UpdateProfile(ctx, userID, profiles.ProfileUpdate{
		HasLocale: true,
		Locale:    &newLocale,
	})
	if err != nil {
		t.Fatalf("second UpdateProfile failed: %v", err)
	}
	if updated2.DisplayName != newName {
		t.Fatalf("expected display name to remain %q, got %q", newName, updated2.DisplayName)
	}
	if updated2.Locale != newLocale {
		t.Fatalf("expected locale %q, got %q", newLocale, updated2.Locale)
	}
}

func TestRepositoryGetPublicProfileHidesInactiveUsers(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	activeID := seedUser(t, db, "active")
	disabledID := seedUser(t, db, "disabled")

	active, err := repo.GetPublicProfile(ctx, activeID)
	if err != nil {
		t.Fatalf("GetPublicProfile failed: %v", err)
	}
	if active == nil {
		t.Fatal("expected active user's public profile to be visible")
	}

	disabled, err := repo.GetPublicProfile(ctx, disabledID)
	if err != nil {
		t.Fatalf("GetPublicProfile failed: %v", err)
	}
	if disabled != nil {
		t.Fatal("expected disabled user's public profile to be hidden")
	}

	missing, err := repo.GetPublicProfile(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetPublicProfile failed: %v", err)
	}
	if missing != nil {
		t.Fatal("expected missing user's public profile to be hidden")
	}
}

func TestRepositoryGetStatsOnlyCountsCompletedGames(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	userID := seedUser(t, db, "active")
	mapID, _ := seedProfileMap(t, db, 1)

	now := time.Now().UTC()
	seedCompletedGame(t, db, userID, mapID, 100, now.Add(-2*time.Hour))
	seedCompletedGame(t, db, userID, mapID, 200, now.Add(-1*time.Hour))

	// A pending game should not count toward stats.
	pending := games.Game{Mode: games.GameModeSolo, Status: games.GameStatusPending, MapID: mapID, RoundCount: 3}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatalf("create pending game failed: %v", err)
	}
	pendingPlayer := games.GamePlayer{GameID: pending.ID, UserID: &userID, DisplayName: "player", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, TotalScore: 999}
	if err := db.Create(&pendingPlayer).Error; err != nil {
		t.Fatalf("create pending game player failed: %v", err)
	}

	stats, err := repo.GetStats(ctx, userID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.GamesPlayed != 2 {
		t.Fatalf("expected 2 completed games, got %d", stats.GamesPlayed)
	}
	if stats.TotalScore != 300 {
		t.Fatalf("expected total score 300, got %d", stats.TotalScore)
	}
	if stats.BestScore != 200 {
		t.Fatalf("expected best score 200, got %d", stats.BestScore)
	}
	if stats.LastPlayedAt == nil {
		t.Fatal("expected last played at to be set")
	}
}

func TestRepositoryListGameHistoryPagination(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	userID := seedUser(t, db, "active")
	mapID, _ := seedProfileMap(t, db, 1)

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		seedCompletedGame(t, db, userID, mapID, 50*(i+1), now.Add(-time.Duration(i)*time.Hour))
	}

	page, err := repo.ListGameHistory(ctx, userID, 2, "")
	if err != nil {
		t.Fatalf("ListGameHistory failed: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items on first page, got %d", len(page.Items))
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor for first page")
	}

	page2, err := repo.ListGameHistory(ctx, userID, 2, *page.NextCursor)
	if err != nil {
		t.Fatalf("ListGameHistory second page failed: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Fatalf("expected 1 item on second page, got %d", len(page2.Items))
	}
	if page2.NextCursor != nil {
		t.Fatal("expected no next cursor on final page")
	}
}

func TestRepositoryListGameHistoryInvalidCursor(t *testing.T) {
	repo, db := setupProfilesRepositoryTest(t)
	ctx := context.Background()

	userID := seedUser(t, db, "active")

	_, err := repo.ListGameHistory(ctx, userID, 10, "not-a-valid-cursor")
	if err != profiles.ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func ptrToPtrMap(m map[string]any) **map[string]any {
	p := &m
	return &p
}
