package rooms

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"gorm.io/gorm"
)

func TestRepositoryCreateJoinAndRejoinFlow(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for room repository integration tests")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("postgres connection failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	repo := NewRepository(db)
	mapID := seedRoomMap(t, db)
	hostGuest := "host-" + uuid.NewString()
	now := time.Now().UTC()
	game := &games.Game{Mode: games.GameModePrivateRoom, Status: games.GameStatusPending, MapID: mapID, RoundCount: 5, ScoringVersion: games.ScoringVersionV1}
	host := &games.GamePlayer{GuestIdentityHash: &hostGuest, DisplayName: "Host", Role: games.PlayerRoleHost, Status: games.PlayerStatusActive, JoinedAt: now}
	room := &Room{Code: "R" + uuid.NewString()[:6], Visibility: VisibilityPrivate, Status: StatusLobby, MaxPlayers: 2, RoundCount: 5, ExpiresAt: now.Add(time.Hour)}
	if err := repo.CreateRoomBundle(ctx, CreateRoomBundle{Room: room, Game: game, Player: host}); err != nil {
		t.Fatalf("CreateRoomBundle failed: %v", err)
	}

	playerGuest := "player-" + uuid.NewString()
	identity := ownerIdentity{guestHash: &playerGuest, displayName: "Player"}
	joined, err := repo.JoinRoom(ctx, room.ID, identity, "Player", now)
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}
	if joined.Player == nil || !joined.Joined {
		t.Fatalf("joined = %+v", joined)
	}
	rejoined, err := repo.JoinRoom(ctx, room.ID, identity, "Player", now)
	if err != nil {
		t.Fatalf("rejoin failed: %v", err)
	}
	if rejoined.Joined {
		t.Fatal("rejoin should not create a new participant")
	}
	participants, err := repo.ListParticipants(ctx, room.ID)
	if err != nil {
		t.Fatalf("ListParticipants failed: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("participants = %d, want 2", len(participants))
	}
}

func seedRoomMap(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	suffix := uuid.NewString()
	m := maps.Map{
		Slug:       "room-test-" + suffix,
		Name:       "Room Test " + suffix,
		Visibility: "public",
		AccessTier: "free",
		Difficulty: "mixed",
		Status:     "active",
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create map failed: %v", err)
	}
	return m.ID
}
