package games_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/platform/clock"
)

func TestTimedMultiplayerDeadlineWorkerClosesPartyLobbyAndZeroFillsMissingGuesses(t *testing.T) {
	repo, db := setupGamesRepositoryTest(t)
	ctx := context.Background()
	mapID, locations := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	timer := 180
	hostHash := "party-worker-host-" + uuid.NewString()
	game := &games.Game{Mode: games.GameModePartyLobby, Status: games.GameStatusPending, MapID: mapID, RoundCount: 1, TimerSeconds: &timer, ScoringVersion: games.ScoringVersionV1}
	host := &games.GamePlayer{GuestIdentityHash: &hostHash, DisplayName: "Host", Role: games.PlayerRoleHost, Status: games.PlayerStatusActive, JoinedAt: now}
	if err := repo.CreateGameBundle(ctx, game, host, nil); err != nil {
		t.Fatalf("create Party Lobby worker game: %v", err)
	}
	guestHash := "party-worker-guest-" + uuid.NewString()
	guest := &games.GamePlayer{GameID: game.ID, GuestIdentityHash: &guestHash, DisplayName: "Guest", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, JoinedAt: now}
	if err := db.Create(guest).Error; err != nil {
		t.Fatalf("create worker guest: %v", err)
	}
	rounds := []games.Round{{GameID: game.ID, LocationID: locations[0], RoundNumber: 1, Status: games.RoundStatusPending}}
	if _, err := repo.StartPrivateRoomGame(ctx, game.ID, rounds, now, &timer, games.MultiplayerTxHooks{}); err != nil {
		t.Fatalf("start Party Lobby worker game: %v", err)
	}

	svc := games.NewService(repo, nil, clock.Fixed(now.Add(181*time.Second)), slog.Default())
	if err := svc.SweepExpiredTimedMultiplayerRounds(ctx, 10); err != nil {
		t.Fatalf("sweep Party Lobby deadline: %v", err)
	}
	var stored games.Game
	if err := db.First(&stored, "id = ?", game.ID).Error; err != nil {
		t.Fatalf("reload worker game: %v", err)
	}
	if stored.Status != games.GameStatusCompleted {
		t.Fatalf("worker game status=%q", stored.Status)
	}
	var timedOut int64
	if err := db.Model(&games.Guess{}).Where("game_player_id IN ? AND timed_out = true AND score = 0", []uuid.UUID{host.ID, guest.ID}).Count(&timedOut).Error; err != nil {
		t.Fatalf("count zero-filled guesses: %v", err)
	}
	if timedOut != 2 {
		t.Fatalf("zero-filled guesses=%d, want 2", timedOut)
	}
}
