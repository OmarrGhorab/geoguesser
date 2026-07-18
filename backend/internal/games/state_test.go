package games_test

import (
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/games"
)

func TestIsMultiplayerModeIncludesMatchmadeModes(t *testing.T) {
	t.Parallel()

	multiplayer := []string{
		games.GameModePrivateRoom,
		games.GameModePartyLobby,
		games.GameModeRanked,
		games.GameModeCasualSolo,
		games.GameModeCasualDuo,
		games.GameModeCasualSquad,
		games.GameModeRankedSolo,
		games.GameModeRankedDuo,
		games.GameModeRankedSquad,
	}
	for _, mode := range multiplayer {
		if !games.IsMultiplayerMode(mode) {
			t.Fatalf("%q should be multiplayer", mode)
		}
	}
	for _, mode := range []string{games.GameModeSolo, games.GameModePractice, games.GameModeDaily, ""} {
		if games.IsMultiplayerMode(mode) {
			t.Fatalf("%q should not be multiplayer", mode)
		}
	}
}

func TestPartyLobbyPracticeModeHelpers(t *testing.T) {
	t.Parallel()

	if !games.IsPartyLobbyMode(games.GameModePartyLobby) || !games.IsPartyLobbyMode(games.GameModePrivateRoom) {
		t.Fatal("canonical and legacy hosted room modes must share party lobby semantics")
	}
	if !games.IsOpenEndedMode(games.GameModePractice) {
		t.Fatal("practice must be open ended")
	}
	for _, mode := range []string{games.GameModePartyLobby, games.GameModePrivateRoom, games.GameModePractice} {
		if !games.IsProgressionNeutralMode(mode) {
			t.Fatalf("%q must be progression neutral", mode)
		}
	}
	if games.IsProgressionNeutralMode(games.GameModeRankedSolo) {
		t.Fatal("ranked must not be progression neutral")
	}
}

func TestIsRankedAndCasualModeHelpers(t *testing.T) {
	t.Parallel()

	if !games.IsRankedMode(games.GameModeRanked) || !games.IsRankedMode(games.GameModeRankedDuo) {
		t.Fatal("ranked modes should be ranked")
	}
	if games.IsRankedMode(games.GameModeCasualSolo) || games.IsRankedMode(games.GameModePrivateRoom) {
		t.Fatal("non-ranked modes must not report ranked")
	}
	if !games.IsCasualMode(games.GameModeCasualSquad) {
		t.Fatal("casual_squad should be casual")
	}
	if games.IsCasualMode(games.GameModeRankedSolo) {
		t.Fatal("ranked_solo should not be casual")
	}
}

func TestCanGuessBeforeStartRankedCanonicalModes(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 7, 18, 12, 0, 5, 0, time.UTC)
	before := starts.Add(-time.Second)
	after := starts.Add(time.Second)

	for _, mode := range []string{games.GameModeRanked, games.GameModeRankedSolo, games.GameModeRankedDuo, games.GameModeRankedSquad} {
		if games.CanGuessBeforeStart(mode, &starts, before) {
			t.Fatalf("%q must reject guesses before starts_at", mode)
		}
		if !games.CanGuessBeforeStart(mode, &starts, after) {
			t.Fatalf("%q must allow guesses after starts_at", mode)
		}
	}

	for _, mode := range []string{games.GameModeCasualSolo, games.GameModeCasualDuo, games.GameModePrivateRoom} {
		if !games.CanGuessBeforeStart(mode, &starts, before) {
			t.Fatalf("%q should allow early guesses relative to starts_at", mode)
		}
	}
}

func TestDelayedRevealPolicy(t *testing.T) {
	t.Parallel()

	var policy games.DelayedRevealPolicy
	if !policy.MayRevealAnswer(games.GameModeSolo, games.RoundStatusActive, false) {
		t.Fatal("solo should always allow reveal policy true for non-multiplayer")
	}
	if policy.MayRevealAnswer(games.GameModeRankedDuo, games.RoundStatusActive, false) {
		t.Fatal("active multiplayer round must not reveal")
	}
	if !policy.MayRevealAnswer(games.GameModeCasualSquad, games.RoundStatusActive, true) {
		t.Fatal("roundCompleted should allow reveal")
	}
	if !policy.MayRevealAnswer(games.GameModeRanked, games.RoundStatusCompleted, false) {
		t.Fatal("completed round status should allow reveal")
	}
}
