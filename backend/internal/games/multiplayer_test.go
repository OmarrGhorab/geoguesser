package games

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
)

func TestRoundsFromSelectedPreservesOrder(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	rounds := roundsFromSelected(gameID, []maps.SelectedLocation{{ID: first}, {ID: second}}, 2)

	if len(rounds) != 2 {
		t.Fatalf("round count = %d", len(rounds))
	}
	if rounds[0].GameID != gameID || rounds[0].LocationID != first || rounds[0].RoundNumber != 1 || rounds[0].Status != RoundStatusPending {
		t.Fatalf("first round = %+v", rounds[0])
	}
	if rounds[1].LocationID != second || rounds[1].RoundNumber != 2 {
		t.Fatalf("second round = %+v", rounds[1])
	}
}

func TestMultiplayerRoundStateShape(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	playerID := uuid.New()
	state := MultiplayerRoundState{
		RoundID:            uuid.New(),
		RoundNumber:        1,
		Status:             RoundStatusActive,
		StartsAt:           &now,
		EndsAt:             &now,
		Provider:           "image",
		ProviderRef:        "https://example.test/pano.jpg",
		SubmittedCount:     1,
		EligibleCount:      2,
		SubmittedPlayerIDs: []uuid.UUID{playerID},
	}

	if state.SubmittedCount != 1 || state.EligibleCount != 2 || state.SubmittedPlayerIDs[0] != playerID {
		t.Fatalf("state = %+v", state)
	}
}
