package rooms

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
	standings, err := repo.ListPartyLobbyStandings(ctx, room.ID)
	if err != nil {
		t.Fatalf("ListPartyLobbyStandings failed: %v", err)
	}
	if len(standings) != 2 || standings[0].Placement != 1 || standings[1].Placement != 1 || !standings[0].Tied || !standings[1].Tied {
		t.Fatalf("standings = %+v", standings)
	}
}

func TestRepositoryPartyLobbyStandingsSupportsFiftyPlayersAndDeterministicTies(t *testing.T) {
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
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	location := locations.Location{Latitude: 30, Longitude: 31, CountryCode: "EG", Difficulty: "easy", Provider: "image", ProviderRef: "party-50-" + uuid.NewString(), Status: "active"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("create location: %v", err)
	}
	if err := db.Create(&maps.MapLocation{MapID: mapID, LocationID: location.ID, SelectionWeight: 1}).Error; err != nil {
		t.Fatalf("link location: %v", err)
	}
	hostGuest := "party-50-host-" + uuid.NewString()
	game := &games.Game{Mode: games.GameModePartyLobby, Status: games.GameStatusActive, MapID: mapID, RoundCount: 1, TimerSeconds: intPtr(180), ScoringVersion: games.ScoringVersionV1, StartedAt: &now}
	host := &games.GamePlayer{GuestIdentityHash: &hostGuest, DisplayName: "Player 00", Role: games.PlayerRoleHost, Status: games.PlayerStatusActive, JoinedAt: now}
	room := &Room{Code: "F" + uuid.NewString()[:5], Visibility: VisibilityPrivate, Status: StatusActive, MaxPlayers: 50, RoundCount: 1, TimerSeconds: intPtr(180), ExpiresAt: now.Add(time.Hour)}
	if err := repo.CreateRoomBundle(ctx, CreateRoomBundle{Room: room, Game: game, Player: host}); err != nil {
		t.Fatalf("create 50-player room: %v", err)
	}
	for i := 1; i < 50; i++ {
		guest := fmt.Sprintf("party-50-%02d-%s", i, uuid.NewString())
		if _, err := repo.JoinRoom(ctx, room.ID, ownerIdentity{guestHash: &guest}, fmt.Sprintf("Player %02d", i), now.Add(time.Duration(i)*time.Millisecond)); err != nil {
			t.Fatalf("join player %d: %v", i, err)
		}
	}
	round := games.Round{GameID: game.ID, LocationID: location.ID, RoundNumber: 1, Status: games.RoundStatusCompleted, StartsAt: &now, RevealedAt: timePtr(now.Add(time.Minute))}
	if err := db.Create(&round).Error; err != nil {
		t.Fatalf("create standings round: %v", err)
	}
	participants, err := repo.ListParticipants(ctx, room.ID)
	if err != nil || len(participants) != 50 {
		t.Fatalf("participants=%d err=%v", len(participants), err)
	}
	for i, participant := range participants {
		score := 4900 - i
		distance := 100 + i
		if i < 2 {
			score = 5000
			distance = 50
		}
		if err := db.Model(&games.GamePlayer{}).Where("id = ?", participant.GamePlayerID).Update("total_score", score).Error; err != nil {
			t.Fatalf("set score %d: %v", i, err)
		}
		guess := games.Guess{RoundID: round.ID, GamePlayerID: participant.GamePlayerID, Latitude: 30, Longitude: 31, DistanceMeters: distance, AccuracyScore: score, Score: score, SubmittedAt: now.Add(time.Second)}
		if err := db.Create(&guess).Error; err != nil {
			t.Fatalf("create guess %d: %v", i, err)
		}
	}

	standings, err := repo.ListPartyLobbyStandings(ctx, room.ID)
	if err != nil {
		t.Fatalf("list 50-player standings: %v", err)
	}
	if len(standings) != 50 {
		t.Fatalf("standings count=%d, want 50", len(standings))
	}
	if standings[0].Placement != 1 || standings[1].Placement != 1 || !standings[0].Tied || !standings[1].Tied || standings[2].Placement != 3 {
		t.Fatalf("tie placements: %+v", standings[:3])
	}
	reloaded, err := repo.ListPartyLobbyStandings(ctx, room.ID)
	if err != nil {
		t.Fatalf("reload standings: %v", err)
	}
	for i := range standings {
		if standings[i].PlayerID != reloaded[i].PlayerID || standings[i].Placement != reloaded[i].Placement {
			t.Fatalf("standing %d changed across reload: %+v != %+v", i, standings[i], reloaded[i])
		}
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

func intPtr(value int) *int              { return &value }
func timePtr(value time.Time) *time.Time { return &value }
