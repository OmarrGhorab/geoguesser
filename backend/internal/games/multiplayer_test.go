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

func TestIsMultiplayerModeIncludesRanked(t *testing.T) {
	t.Parallel()
	if !IsMultiplayerMode(GameModePrivateRoom) {
		t.Fatal("private_room should be multiplayer")
	}
	if !IsMultiplayerMode(GameModeRanked) {
		t.Fatal("ranked should be multiplayer")
	}
	if IsMultiplayerMode(GameModeSolo) {
		t.Fatal("solo should not be multiplayer")
	}
	if !IsMultiplayerMode(GameModeCasualDuo) {
		t.Fatal("casual_duo should be multiplayer")
	}
}

func TestSumTeamTotalsAndDecideResult(t *testing.T) {
	t.Parallel()
	one, two := TeamSlotOne, TeamSlotTwo
	players := []GamePlayer{
		{TeamSlot: &one, TotalScore: 10},
		{TeamSlot: &two, TotalScore: 10},
	}
	totals := SumTeamTotals(players)
	if totals.TeamOneScore != 10 || totals.TeamTwoScore != 10 {
		t.Fatalf("totals=%+v", totals)
	}
	result, winner := DecideTeamResult(totals.TeamOneScore, totals.TeamTwoScore)
	if result != ResultDraw || winner != nil {
		t.Fatalf("result=%s winner=%v", result, winner)
	}
}

func TestCanGuessBeforeStart_RankedCountdown(t *testing.T) {
	t.Parallel()
	starts := time.Date(2026, 7, 11, 12, 0, 5, 0, time.UTC)
	before := starts.Add(-time.Second)
	after := starts.Add(time.Second)

	if CanGuessBeforeStart(GameModeRanked, &starts, before) {
		t.Fatal("ranked must reject guesses before starts_at")
	}
	if !CanGuessBeforeStart(GameModeRanked, &starts, after) {
		t.Fatal("ranked must allow guesses after starts_at")
	}
	if !CanGuessBeforeStart(GameModePrivateRoom, &starts, before) {
		t.Fatal("private_room should allow early guesses relative to starts_at")
	}
	if !CanGuessBeforeStart(GameModeRanked, nil, before) {
		t.Fatal("nil starts_at should allow guesses")
	}
}
