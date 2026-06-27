package games_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"gorm.io/gorm"
)

func setupGamesRepositoryTest(t *testing.T) (*games.Repository, *gorm.DB) {
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

	return games.NewRepository(db), db.WithContext(context.Background())
}

func TestRepositorySetup(t *testing.T) {
	repo, _ := setupGamesRepositoryTest(t)
	if repo == nil {
		t.Fatal("repo should be created")
	}
}

func TestRepositorySoloGamePersistenceFlow(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 3)
	guest := "guest-" + uuid.NewString()

	game := &games.Game{Mode: games.GameModeSolo, Status: games.GameStatusPending, MapID: mapID, RoundCount: 3, ScoringVersion: games.ScoringVersionV1}
	player := &games.GamePlayer{GuestIdentityHash: &guest, DisplayName: "Guest", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	rounds := []games.Round{
		{LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending},
		{LocationID: locationIDs[1], RoundNumber: 2, Status: games.RoundStatusPending},
		{LocationID: locationIDs[2], RoundNumber: 3, Status: games.RoundStatusPending},
	}
	if err := repo.CreateGameBundle(ctx, game, player, rounds); err != nil {
		t.Fatalf("CreateGameBundle failed: %v", err)
	}
	if game.ID == uuid.Nil || player.ID == uuid.Nil {
		t.Fatal("created game and player should have ids")
	}

	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	started, err := repo.StartGame(ctx, game.ID, now, nil)
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}
	if started.Status != games.GameStatusActive || started.CurrentRoundNumber == nil || *started.CurrentRoundNumber != 1 {
		t.Fatalf("started game = %+v", started)
	}

	current, err := repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		t.Fatalf("GetCurrentRound failed: %v", err)
	}
	if current == nil || current.RoundNumber != 1 {
		t.Fatalf("current round = %+v", current)
	}

	key := "guess-" + uuid.NewString()
	saved, answer, completed, err := repo.SubmitGuessTx(ctx, game.ID, current.RoundID, player.ID, games.Guess{
		Latitude:       30.0444,
		Longitude:      31.2357,
		IdempotencyKey: &key,
	}, now.Add(time.Second))
	if err != nil {
		t.Fatalf("SubmitGuessTx failed: %v", err)
	}
	if saved == nil || answer == nil || completed {
		t.Fatalf("saved=%+v answer=%+v completed=%v", saved, answer, completed)
	}
	if saved.Score < 0 || saved.Score > 5000 {
		t.Fatalf("score = %d", saved.Score)
	}

	loadedGuess, err := repo.GetGuessByRoundPlayer(ctx, current.RoundID, player.ID)
	if err != nil {
		t.Fatalf("GetGuessByRoundPlayer failed: %v", err)
	}
	if loadedGuess == nil || loadedGuess.ID != saved.ID {
		t.Fatalf("loaded guess = %+v, want %s", loadedGuess, saved.ID)
	}
	replayGuess, err := repo.GetGuessByIdempotencyKey(ctx, player.ID, key)
	if err != nil {
		t.Fatalf("GetGuessByIdempotencyKey failed: %v", err)
	}
	if replayGuess == nil || replayGuess.ID != saved.ID {
		t.Fatalf("replay guess = %+v, want %s", replayGuess, saved.ID)
	}

	var refreshedPlayer games.GamePlayer
	if err := db.First(&refreshedPlayer, "id = ?", player.ID).Error; err != nil {
		t.Fatalf("load player failed: %v", err)
	}
	if refreshedPlayer.TotalScore != saved.Score {
		t.Fatalf("player total = %d, want %d", refreshedPlayer.TotalScore, saved.Score)
	}
}

func TestRepositoryGuessUniquenessAndIdempotencyConstraints(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	guest := "guest-" + uuid.NewString()
	game := &games.Game{Mode: games.GameModeSolo, Status: games.GameStatusPending, MapID: mapID, RoundCount: 2, ScoringVersion: games.ScoringVersionV1}
	player := &games.GamePlayer{GuestIdentityHash: &guest, DisplayName: "Guest", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	rounds := []games.Round{
		{LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending},
		{LocationID: locationIDs[1], RoundNumber: 2, Status: games.RoundStatusPending},
	}
	if err := repo.CreateGameBundle(ctx, game, player, rounds); err != nil {
		t.Fatalf("CreateGameBundle failed: %v", err)
	}
	if _, err := repo.StartGame(ctx, game.ID, time.Now().UTC(), nil); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}
	current, err := repo.GetCurrentRound(ctx, game.ID)
	if err != nil {
		t.Fatalf("GetCurrentRound failed: %v", err)
	}
	key := "same-key-" + uuid.NewString()
	if _, _, _, err := repo.SubmitGuessTx(ctx, game.ID, current.RoundID, player.ID, games.Guess{Latitude: 1, Longitude: 1, IdempotencyKey: &key}, time.Now().UTC()); err != nil {
		t.Fatalf("first SubmitGuessTx failed: %v", err)
	}
	if _, _, _, err := repo.SubmitGuessTx(ctx, game.ID, current.RoundID, player.ID, games.Guess{Latitude: 2, Longitude: 2}, time.Now().UTC()); err == nil {
		t.Fatal("second guess for same round should fail")
	}

	var distinctLocations int64
	if err := db.Model(&games.Round{}).Where("game_id = ?", game.ID).Distinct("location_id").Count(&distinctLocations).Error; err != nil {
		t.Fatalf("count locations failed: %v", err)
	}
	if distinctLocations != int64(len(locationIDs)) {
		t.Fatalf("distinct locations = %d, want %d", distinctLocations, len(locationIDs))
	}

	otherRound := games.Round{GameID: game.ID, LocationID: locationIDs[1], RoundNumber: 99, Status: games.RoundStatusCompleted}
	if err := db.Create(&otherRound).Error; err != nil {
		t.Fatalf("create other round failed: %v", err)
	}
	conflicting := games.Guess{RoundID: otherRound.ID, GamePlayerID: player.ID, Latitude: 3, Longitude: 3, DistanceMeters: 1, Score: 1, IdempotencyKey: &key, SubmittedAt: time.Now().UTC()}
	if err := db.Create(&conflicting).Error; err == nil {
		t.Fatal("duplicate idempotency key for same player should fail")
	}
}

func TestRepositoryLoadResultsBatchedShape(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	guest := "guest-" + uuid.NewString()
	game := &games.Game{Mode: games.GameModeSolo, Status: games.GameStatusPending, MapID: mapID, RoundCount: 2, ScoringVersion: games.ScoringVersionV1}
	player := &games.GamePlayer{GuestIdentityHash: &guest, DisplayName: "Guest", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	rounds := []games.Round{
		{LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending},
		{LocationID: locationIDs[1], RoundNumber: 2, Status: games.RoundStatusPending},
	}
	if err := repo.CreateGameBundle(ctx, game, player, rounds); err != nil {
		t.Fatalf("CreateGameBundle failed: %v", err)
	}
	now := time.Now().UTC()
	if _, err := repo.StartGame(ctx, game.ID, now, nil); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}
	for i := 0; i < 2; i++ {
		current, err := repo.GetCurrentRound(ctx, game.ID)
		if err != nil {
			t.Fatalf("GetCurrentRound failed: %v", err)
		}
		if _, _, _, err := repo.SubmitGuessTx(ctx, game.ID, current.RoundID, player.ID, games.Guess{Latitude: float64(i), Longitude: float64(i)}, now.Add(time.Duration(i+1)*time.Second)); err != nil {
			t.Fatalf("SubmitGuessTx round %d failed: %v", i+1, err)
		}
	}

	loadedGame, players, results, err := repo.LoadResults(ctx, game.ID)
	if err != nil {
		t.Fatalf("LoadResults failed: %v", err)
	}
	if loadedGame == nil || loadedGame.Status != games.GameStatusCompleted {
		t.Fatalf("loaded game = %+v", loadedGame)
	}
	if len(players) != 1 || len(results) != 2 {
		t.Fatalf("players=%d results=%d", len(players), len(results))
	}
	for _, result := range results {
		if len(result.Guesses) != 1 {
			t.Fatalf("round %d guesses = %d, want 1", result.RoundNumber, len(result.Guesses))
		}
	}
}

func seedGameMap(t *testing.T, db *gorm.DB, count int) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	suffix := uuid.NewString()
	m := maps.Map{
		Slug:       "solo-game-" + suffix,
		Name:       "Solo Game " + suffix,
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
			ProviderRef: fmt.Sprintf("repo-test-%s-%d", suffix, i),
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
