package games_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/rooms"
	"gorm.io/gorm"
)

func TestPartyLobbyLifecycleIsAtomicAndCompletesOnAllSubmissions(t *testing.T) {
	gamesRepo, db := setupGamesRepositoryTest(t)
	roomsRepo := rooms.NewRepository(db)
	ctx := context.Background()
	mapID, locationIDs := seedGameMap(t, db, 1)
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	timer := 180
	hostGuest := "party-atomic-host-" + uuid.NewString()
	game := &games.Game{Mode: games.GameModePartyLobby, Status: games.GameStatusPending, MapID: mapID, RoundCount: 1, TimerSeconds: &timer, ScoringVersion: games.ScoringVersionV1}
	host := &games.GamePlayer{GuestIdentityHash: &hostGuest, DisplayName: "Host", Role: games.PlayerRoleHost, Status: games.PlayerStatusActive, JoinedAt: now}
	room := &rooms.Room{Code: "A" + uuid.NewString()[:5], Visibility: rooms.VisibilityPrivate, Status: rooms.StatusLobby, MaxPlayers: 2, RoundCount: 1, TimerSeconds: &timer, ExpiresAt: now.Add(time.Hour)}
	if err := roomsRepo.CreateRoomBundle(ctx, rooms.CreateRoomBundle{Room: room, Game: game, Player: host}); err != nil {
		t.Fatalf("create Party Lobby: %v", err)
	}
	guestHash := "party-atomic-guest-" + uuid.NewString()
	guest := &games.GamePlayer{GameID: game.ID, GuestIdentityHash: &guestHash, DisplayName: "Guest", Role: games.PlayerRolePlayer, Status: games.PlayerStatusActive, JoinedAt: now.Add(time.Millisecond)}
	if err := db.Create(guest).Error; err != nil {
		t.Fatalf("create guest player: %v", err)
	}
	if err := db.Create(&rooms.RoomPlayer{RoomID: room.ID, GamePlayerID: guest.ID, Status: rooms.ParticipantStatusJoined, JoinedAt: guest.JoinedAt}).Error; err != nil {
		t.Fatalf("create guest membership: %v", err)
	}

	rounds := []games.Round{{GameID: game.ID, LocationID: locationIDs[0], RoundNumber: 1, Status: games.RoundStatusPending}}
	_, rollbackErr := gamesRepo.StartPrivateRoomGame(ctx, game.ID, rounds, now, &timer, games.MultiplayerTxHooks{
		OnMatchActive: func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
			if err := roomsRepo.ApplyRoomStartedInTx(ctx, tx, gameID, at); err != nil {
				return err
			}
			return errors.New("force lifecycle rollback")
		},
	})
	if rollbackErr == nil {
		t.Fatal("expected forced lifecycle rollback")
	}
	assertGameAndRoomStatus(t, db, game.ID, room.ID, games.GameStatusPending, rooms.StatusLobby)

	hooks := games.MultiplayerTxHooks{
		OnMatchActive:   roomsRepo.ApplyRoomStartedInTx,
		OnGameCompleted: roomsRepo.ApplyRoomCompletedInTx,
	}
	started, err := gamesRepo.StartPrivateRoomGame(ctx, game.ID, rounds, now, &timer, hooks)
	if err != nil {
		t.Fatalf("start Party Lobby atomically: %v", err)
	}
	if started.CurrentRound.EndsAt == nil || !started.CurrentRound.EndsAt.Equal(now.Add(180*time.Second)) {
		t.Fatalf("Party Lobby deadline=%v", started.CurrentRound.EndsAt)
	}
	assertGameAndRoomStatus(t, db, game.ID, room.ID, games.GameStatusActive, rooms.StatusActive)

	first, _, err := gamesRepo.SubmitMultiplayerGuessTx(ctx, game.ID, started.CurrentRound.ID, host.ID, games.Guess{Latitude: 30, Longitude: 31}, now.Add(time.Second), hooks)
	if err != nil || first.RoundCompleted {
		t.Fatalf("first Party Lobby guess=%+v err=%v", first, err)
	}
	second, _, err := gamesRepo.SubmitMultiplayerGuessTx(ctx, game.ID, started.CurrentRound.ID, guest.ID, games.Guess{Latitude: 30, Longitude: 31}, now.Add(2*time.Second), hooks)
	if err != nil || !second.RoundCompleted || !second.GameCompleted {
		t.Fatalf("second Party Lobby guess=%+v err=%v", second, err)
	}
	assertGameAndRoomStatus(t, db, game.ID, room.ID, games.GameStatusCompleted, rooms.StatusCompleted)
}

func assertGameAndRoomStatus(t *testing.T, db *gorm.DB, gameID, roomID uuid.UUID, wantGame, wantRoom string) {
	t.Helper()
	var gameStatus string
	if err := db.Raw(`SELECT status FROM games WHERE id = ?`, gameID).Scan(&gameStatus).Error; err != nil {
		t.Fatalf("load game status: %v", err)
	}
	var roomStatus string
	if err := db.Raw(`SELECT status FROM rooms WHERE id = ?`, roomID).Scan(&roomStatus).Error; err != nil {
		t.Fatalf("load room status: %v", err)
	}
	if gameStatus != wantGame || roomStatus != wantRoom {
		t.Fatalf("game status=%q want=%q room status=%q want=%q", gameStatus, wantGame, roomStatus, wantRoom)
	}
}
