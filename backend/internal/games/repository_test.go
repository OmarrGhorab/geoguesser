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
	"github.com/raven/geoguess/backend/internal/matchmaking"
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
	conflicting := games.Guess{RoundID: otherRound.ID, GamePlayerID: player.ID, Latitude: 3, Longitude: 3, DistanceMeters: 1, AccuracyScore: 1, Score: 1, IdempotencyKey: &key, SubmittedAt: time.Now().UTC()}
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

func TestRankedMultiplayerModeGates(t *testing.T) {
	if !games.IsMultiplayerMode(games.GameModeRanked) {
		t.Fatal("ranked must use multiplayer semantics")
	}
	if !games.IsMultiplayerMode(games.GameModePrivateRoom) {
		t.Fatal("private_room must use multiplayer semantics")
	}
	starts := time.Now().UTC().Add(time.Minute)
	if games.CanGuessBeforeStart(games.GameModeRanked, &starts, time.Now().UTC()) {
		t.Fatal("ranked must reject guesses before starts_at")
	}
	if !games.CanGuessBeforeStart(games.GameModePrivateRoom, &starts, time.Now().UTC()) {
		t.Fatal("private_room allows early guesses relative to starts_at")
	}
}

func TestCloseExpiredMultiplayerRoundAdvancesDeadline(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	_ = db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
		userA, userA.String()+"@example.test", now, now)
	_ = db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
		userB, userB.String()+"@example.test", now, now)

	timer := 60
	started := now.Add(-2 * time.Minute)
	game := &games.Game{
		Mode: games.GameModeRanked, Status: games.GameStatusActive, MapID: mapID,
		RoundCount: 2, TimerSeconds: &timer, ScoringVersion: games.ScoringVersionV1, StartedAt: &started,
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("create game: %v", err)
	}
	slotOne, slotTwo := games.TeamSlotOne, games.TeamSlotTwo
	pA := &games.GamePlayer{GameID: game.ID, UserID: &userA, DisplayName: "A", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, TeamSlot: &slotOne}
	pB := &games.GamePlayer{GameID: game.ID, UserID: &userB, DisplayName: "B", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, TeamSlot: &slotTwo}
	if err := db.Create(pA).Error; err != nil {
		t.Fatalf("player A: %v", err)
	}
	if err := db.Create(pB).Error; err != nil {
		t.Fatalf("player B: %v", err)
	}
	roundStart := now.Add(-2 * time.Minute)
	roundEnd := now.Add(-time.Minute)
	r1 := games.Round{GameID: game.ID, LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusActive, StartsAt: &roundStart, EndsAt: &roundEnd}
	r2 := games.Round{GameID: game.ID, LocationID: locationIDs[1], RoundNumber: 2, Status: games.RoundStatusPending}
	if err := db.Create(&r1).Error; err != nil {
		t.Fatalf("round1: %v", err)
	}
	if err := db.Create(&r2).Error; err != nil {
		t.Fatalf("round2: %v", err)
	}

	out, err := repo.CloseExpiredMultiplayerRound(ctx, game.ID, now, games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("close expired: %v", err)
	}
	if out == nil || !out.RoundCompleted {
		t.Fatalf("expected round completed via deadline, got %+v", out)
	}
	if out.NextRoundNumber == nil || *out.NextRoundNumber != 2 {
		t.Fatalf("expected next round 2, got %+v", out.NextRoundNumber)
	}
	current, err := repo.GetCurrentRound(ctx, game.ID)
	if err != nil || current == nil || current.RoundNumber != 2 {
		t.Fatalf("current after deadline = %+v err=%v", current, err)
	}
}

func TestSubmitMultiplayerGuessRequiresActiveMultiplayerGame(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 2)
	userID := uuid.New()
	now := time.Now().UTC()

	// Seed minimal registered user for FK if required by identity checks.
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
		userID, userID.String()+"@example.test", now, now).Error; err != nil {
		// Some environments may already satisfy identity without explicit users for guest-only games.
		_ = err
	}

	timer := 60
	game := &games.Game{
		Mode:           games.GameModeRanked,
		Status:         games.GameStatusActive,
		MapID:          mapID,
		RoundCount:     2,
		TimerSeconds:   &timer,
		ScoringVersion: games.ScoringVersionV1,
		StartedAt:      &now,
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("create ranked game: %v", err)
	}
	playerA := &games.GamePlayer{GameID: game.ID, UserID: &userID, DisplayName: "A", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	userB := uuid.New()
	_ = db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
		userB, userB.String()+"@example.test", now, now)
	playerB := &games.GamePlayer{GameID: game.ID, UserID: &userB, DisplayName: "B", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	if err := db.Create(playerA).Error; err != nil {
		t.Fatalf("create player A: %v", err)
	}
	if err := db.Create(playerB).Error; err != nil {
		t.Fatalf("create player B: %v", err)
	}
	starts := now.Add(-time.Second)
	ends := now.Add(time.Minute)
	round1 := games.Round{GameID: game.ID, LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusActive, StartsAt: &starts, EndsAt: &ends}
	round2 := games.Round{GameID: game.ID, LocationID: locationIDs[1], RoundNumber: 2, Status: games.RoundStatusPending}
	if err := db.Create(&round1).Error; err != nil {
		t.Fatalf("create round1: %v", err)
	}
	if err := db.Create(&round2).Error; err != nil {
		t.Fatalf("create round2: %v", err)
	}

	// Early-start gate: future starts_at rejects.
	future := now.Add(time.Minute)
	if err := db.Model(&games.Round{}).Where("id = ?", round1.ID).Update("starts_at", future).Error; err != nil {
		t.Fatalf("set future starts: %v", err)
	}
	_, _, err := repo.SubmitMultiplayerGuessTx(ctx, game.ID, round1.ID, playerA.ID, games.Guess{Latitude: 1, Longitude: 1}, now, games.MultiplayerTxHooks{})
	if err == nil {
		t.Fatal("expected early guess rejection")
	}

	// Restore starts and complete both guesses to advance round.
	if err := db.Model(&games.Round{}).Where("id = ?", round1.ID).Update("starts_at", starts).Error; err != nil {
		t.Fatalf("restore starts: %v", err)
	}
	if _, _, err := repo.SubmitMultiplayerGuessTx(ctx, game.ID, round1.ID, playerA.ID, games.Guess{Latitude: 1, Longitude: 1}, now, games.MultiplayerTxHooks{}); err != nil {
		t.Fatalf("guess A: %v", err)
	}
	out, _, err := repo.SubmitMultiplayerGuessTx(ctx, game.ID, round1.ID, playerB.ID, games.Guess{Latitude: 2, Longitude: 2}, now.Add(time.Second), games.MultiplayerTxHooks{})
	if err != nil {
		t.Fatalf("guess B: %v", err)
	}
	if out == nil || !out.RoundCompleted {
		t.Fatalf("expected round completion after both guesses, got %+v", out)
	}
}

func TestMultiplayerLifecycleHooks_CommitAndRollback(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	if !db.Migrator().HasTable("matches") {
		t.Skip("matches table missing; run migration 00014")
	}
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	for _, u := range []uuid.UUID{userA, userB} {
		if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
			u, u.String()+"@example.test", now, now).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	timer := 60
	started := now.Add(-time.Minute)
	ends := now.Add(-time.Second)
	game := &games.Game{
		Mode: games.GameModeRanked, Status: games.GameStatusActive, MapID: mapID,
		RoundCount: 1, TimerSeconds: &timer, ScoringVersion: games.ScoringVersionV1, StartedAt: &started,
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("create game: %v", err)
	}
	pA := &games.GamePlayer{GameID: game.ID, UserID: &userA, DisplayName: "A", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	pB := &games.GamePlayer{GameID: game.ID, UserID: &userB, DisplayName: "B", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive}
	if err := db.Create(pA).Error; err != nil {
		t.Fatalf("player A: %v", err)
	}
	if err := db.Create(pB).Error; err != nil {
		t.Fatalf("player B: %v", err)
	}
	round := games.Round{GameID: game.ID, LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusActive, StartsAt: &started, EndsAt: &ends}
	if err := db.Create(&round).Error; err != nil {
		t.Fatalf("round: %v", err)
	}
	seasonID := activeSeasonID(t, db)
	match := &matchmaking.Match{
		FormationKey: "lifecycle-" + uuid.NewString(), GameID: game.ID, Mode: matchmaking.ModeRankedStandard,
		Playlist: matchmaking.PlaylistRanked, Format: matchmaking.FormatSolo, TeamSize: matchmaking.TeamSizeSolo, SeasonID: &seasonID,
		Status: matchmaking.MatchStatusActive, MatchedAt: now, StartedAt: &started, LastActivityAt: now,
	}
	if err := db.Create(match).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
	for i, player := range []*games.GamePlayer{pA, pB} {
		participant := &matchmaking.MatchPlayer{
			MatchID: match.ID, UserID: *player.UserID, GamePlayerID: player.ID,
			TeamSlot: i + 1, Status: matchmaking.ParticipantStatusActive, AssignedAt: now,
		}
		if err := db.Create(participant).Error; err != nil {
			t.Fatalf("create match player: %v", err)
		}
	}
	adapter := matchmaking.NewRankedLifecycleAdapter(matchmaking.NewRepository(db))

	// Failing lifecycle hook must roll back game, match, and participant completion.
	_, err := repo.CloseExpiredMultiplayerRound(ctx, game.ID, now, games.MultiplayerTxHooks{
		OnGameCompleted: func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
			if err := adapter.ApplyCompletedInTx(ctx, tx, gameID, at); err != nil {
				return err
			}
			return fmt.Errorf("hook boom")
		},
	})
	if err == nil {
		t.Fatal("expected completion hook failure")
	}
	var status string
	if err := db.Raw(`SELECT status FROM games WHERE id = ?`, game.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload game: %v", err)
	}
	if status != games.GameStatusActive {
		t.Fatalf("game status after failed hook = %q, want active (rolled back)", status)
	}
	var matchStatus string
	if err := db.Raw(`SELECT status FROM matches WHERE id = ?`, match.ID).Scan(&matchStatus).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	if matchStatus != matchmaking.MatchStatusActive {
		t.Fatalf("match status after failed hook = %q, want active", matchStatus)
	}
	var activeParticipants int64
	if err := db.Model(&matchmaking.MatchPlayer{}).Where("match_id = ? AND status = ?", match.ID, matchmaking.ParticipantStatusActive).Count(&activeParticipants).Error; err != nil {
		t.Fatalf("count active participants: %v", err)
	}
	if activeParticipants != 2 {
		t.Fatalf("active participants = %d, want 2 after rollback", activeParticipants)
	}

	// Successful lifecycle hook commits game, match, and participant completion.
	out, err := repo.CloseExpiredMultiplayerRound(ctx, game.ID, now, games.MultiplayerTxHooks{
		OnGameCompleted: func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
			return adapter.ApplyCompletedInTx(ctx, tx, gameID, at)
		},
	})
	if err != nil {
		t.Fatalf("close expired success: %v", err)
	}
	if out == nil || !out.GameCompleted {
		t.Fatalf("expected game completed, got %+v", out)
	}
	if err := db.Raw(`SELECT status FROM games WHERE id = ?`, game.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload completed game: %v", err)
	}
	if status != games.GameStatusCompleted {
		t.Fatalf("game status = %q, want completed", status)
	}
	if err := db.Raw(`SELECT status FROM matches WHERE id = ?`, match.ID).Scan(&matchStatus).Error; err != nil {
		t.Fatalf("reload completed match: %v", err)
	}
	if matchStatus != matchmaking.MatchStatusCompleted {
		t.Fatalf("match status = %q, want completed", matchStatus)
	}
	var completedParticipants int64
	if err := db.Model(&matchmaking.MatchPlayer{}).Where("match_id = ? AND status = ?", match.ID, matchmaking.ParticipantStatusCompleted).Count(&completedParticipants).Error; err != nil {
		t.Fatalf("count completed participants: %v", err)
	}
	if completedParticipants != 2 {
		t.Fatalf("completed participants = %d, want 2", completedParticipants)
	}
}

func TestCancelMultiplayerGameTx_HookCommitAndRollback(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	if !db.Migrator().HasTable("matches") {
		t.Skip("matches table missing; run migration 00014")
	}
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	for _, u := range []uuid.UUID{userA, userB} {
		_ = db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?) ON CONFLICT DO NOTHING`,
			u, u.String()+"@example.test", now, now)
	}
	timer := 60
	started := now
	game := &games.Game{
		Mode: games.GameModeRanked, Status: games.GameStatusActive, MapID: mapID,
		RoundCount: 1, TimerSeconds: &timer, ScoringVersion: games.ScoringVersionV1, StartedAt: &started,
	}
	if err := db.Create(game).Error; err != nil {
		t.Fatalf("create game: %v", err)
	}
	for _, name := range []struct {
		user uuid.UUID
		n    string
		slot int
	}{{userA, "A", games.TeamSlotOne}, {userB, "B", games.TeamSlotTwo}} {
		p := &games.GamePlayer{GameID: game.ID, UserID: &name.user, DisplayName: name.n, Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, TeamSlot: &name.slot}
		if err := db.Create(p).Error; err != nil {
			t.Fatalf("player: %v", err)
		}
	}
	ends := now.Add(time.Minute)
	r := games.Round{GameID: game.ID, LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusActive, StartsAt: &started, EndsAt: &ends}
	if err := db.Create(&r).Error; err != nil {
		t.Fatalf("round: %v", err)
	}
	seasonID := activeSeasonID(t, db)
	match := &matchmaking.Match{
		FormationKey: "cancel-" + uuid.NewString(), GameID: game.ID, Mode: matchmaking.ModeRankedStandard,
		Playlist: matchmaking.PlaylistRanked, Format: matchmaking.FormatSolo, TeamSize: matchmaking.TeamSizeSolo, SeasonID: &seasonID,
		Status: matchmaking.MatchStatusActive, MatchedAt: now, StartedAt: &started, LastActivityAt: now,
	}
	if err := db.Create(match).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
	var players []games.GamePlayer
	if err := db.Where("game_id = ?", game.ID).Find(&players).Error; err != nil {
		t.Fatalf("load game players: %v", err)
	}
	for i := range players {
		participant := &matchmaking.MatchPlayer{
			MatchID: match.ID, UserID: *players[i].UserID, GamePlayerID: players[i].ID,
			TeamSlot: *players[i].TeamSlot, Status: matchmaking.ParticipantStatusActive, AssignedAt: now,
		}
		if err := db.Create(participant).Error; err != nil {
			t.Fatalf("create match player: %v", err)
		}
	}
	adapter := matchmaking.NewRankedLifecycleAdapter(matchmaking.NewRepository(db))

	// Failing lifecycle hook rolls back game, match, and participants.
	err := repo.CancelMultiplayerGameTx(ctx, game.ID, now, games.GameStatusAbandoned, "", games.MultiplayerTxHooks{
		OnGameCancelled: func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, code string) error {
			if err := adapter.ApplyCancelledInTx(ctx, tx, gameID, at, code); err != nil {
				return err
			}
			return fmt.Errorf("cancel hook boom")
		},
	})
	if err == nil {
		t.Fatal("expected cancel hook failure")
	}
	var status string
	if err := db.Raw(`SELECT status FROM games WHERE id = ?`, game.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if status != games.GameStatusActive {
		t.Fatalf("status after failed cancel = %q, want active", status)
	}
	var matchStatus string
	if err := db.Raw(`SELECT status FROM matches WHERE id = ?`, match.ID).Scan(&matchStatus).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	if matchStatus != matchmaking.MatchStatusActive {
		t.Fatalf("match status after rollback = %q, want active", matchStatus)
	}

	// Startup failure commits a failed-to-start match, failed participants, and cancelled game/rounds.
	if err := repo.CancelMultiplayerGameTx(ctx, game.ID, now, games.GameStatusCancelled, "startup_timeout", games.MultiplayerTxHooks{
		OnGameCancelled: func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, code string) error {
			return adapter.ApplyCancelledInTx(ctx, tx, gameID, at, code)
		},
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := db.Raw(`SELECT status FROM games WHERE id = ?`, game.ID).Scan(&status).Error; err != nil {
		t.Fatalf("reload abandoned: %v", err)
	}
	if status != games.GameStatusCancelled {
		t.Fatalf("status = %q, want cancelled", status)
	}
	var roundStatus string
	if err := db.Raw(`SELECT status FROM rounds WHERE id = ?`, r.ID).Scan(&roundStatus).Error; err != nil {
		t.Fatalf("reload round: %v", err)
	}
	if roundStatus != games.RoundStatusCancelled {
		t.Fatalf("round status = %q, want cancelled", roundStatus)
	}
	if err := db.Raw(`SELECT status FROM matches WHERE id = ?`, match.ID).Scan(&matchStatus).Error; err != nil {
		t.Fatalf("reload failed match: %v", err)
	}
	if matchStatus != matchmaking.MatchStatusFailedToStart {
		t.Fatalf("match status = %q, want failed_to_start", matchStatus)
	}
	var failureCode string
	if err := db.Raw(`SELECT failure_code FROM matches WHERE id = ?`, match.ID).Scan(&failureCode).Error; err != nil {
		t.Fatalf("load failure code: %v", err)
	}
	if failureCode != "startup_timeout" {
		t.Fatalf("failure code = %q, want startup_timeout", failureCode)
	}
	var failedParticipants int64
	if err := db.Model(&matchmaking.MatchPlayer{}).Where("match_id = ? AND status = ?", match.ID, matchmaking.ParticipantStatusFailed).Count(&failedParticipants).Error; err != nil {
		t.Fatalf("count failed participants: %v", err)
	}
	if failedParticipants != 2 {
		t.Fatalf("failed participants = %d, want 2", failedParticipants)
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

func activeSeasonID(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	var row struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := db.Table("competitive_seasons").Select("id").Where("status = ?", "active").Order("sequence ASC").Limit(1).Scan(&row).Error; err != nil {
		t.Fatalf("load active season: %v", err)
	}
	if row.ID == uuid.Nil {
		t.Fatal("active competitive season is required")
	}
	return row.ID
}
