package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/rooms"
)

func BenchmarkPartyLobbyFiftyPlayerSnapshot(b *testing.B) {
	standings := make([]rooms.PartyLobbyStanding, 50)
	for i := range standings {
		standings[i] = rooms.PartyLobbyStanding{Placement: i + 1, PlayerID: uuid.New(), DisplayName: "Player", TotalScore: 25000 - i}
	}
	payload := rooms.RoomResponse{Room: rooms.RoomDTO{ID: uuid.New(), Code: "ABC123", Mode: games.GameModePartyLobby, Standings: standings}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPracticeHundredRoundHistoryPage(b *testing.B) {
	items := make([]games.PracticeRoundHistoryItem, 100)
	for i := range items {
		items[i] = games.PracticeRoundHistoryItem{Round: games.RoundDTO{ID: uuid.New(), RoundNumber: i + 1, Status: games.RoundStatusCompleted, StartsAt: timePtr(time.Now())}}
	}
	payload := games.PracticeHistoryResponse{Items: items, HasMore: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func timePtr(value time.Time) *time.Time { return &value }
