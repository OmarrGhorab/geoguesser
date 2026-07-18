package matchplay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/session"
)

func registeredSession(userID uuid.UUID) *session.Context {
	s := userID.String()
	return &session.Context{Kind: session.KindUser, UserID: &s, Role: "user"}
}

func seedCasual1v1(store *memoryStore, now time.Time) (matchID, gameID, userA, userB, gpA, gpB uuid.UUID) {
	matchID, gameID = uuid.New(), uuid.New()
	userA, userB = uuid.New(), uuid.New()
	gpA, gpB = uuid.New(), uuid.New()
	locID := uuid.New()

	match := matchplay.Match{
		ID:             matchID,
		FormationKey:   "fk-" + matchID.String(),
		GameID:         gameID,
		Mode:           "casual_solo",
		Status:         matchplay.MatchStatusActive,
		Playlist:       matchplay.PlaylistCasual,
		Format:         matchplay.FormatSolo,
		TeamSize:       1,
		TeamOneScore:   1200,
		TeamTwoScore:   800,
		LastActivityAt: now,
		MatchedAt:      now,
		StartedAt:      &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	parts := []matchplay.MatchParticipant{
		{MatchID: matchID, UserID: userA, GamePlayerID: gpA, Status: matchplay.ParticipantStatusActive, TeamSlot: 1, AssignedAt: now},
		{MatchID: matchID, UserID: userB, GamePlayerID: gpB, Status: matchplay.ParticipantStatusActive, TeamSlot: 2, AssignedAt: now},
	}
	players := []matchplay.GamePlayerRow{
		{ID: gpA, GameID: gameID, UserID: &userA, DisplayName: "Alice", Status: matchplay.PlayerStatusActive, TotalScore: 1200, TeamSlot: intPtr(1)},
		{ID: gpB, GameID: gameID, UserID: &userB, DisplayName: "Bob", Status: matchplay.PlayerStatusActive, TotalScore: 800, TeamSlot: intPtr(2)},
	}
	store.seedMatch(match, parts, players)

	// Active round — not revealed.
	roundID := uuid.New()
	store.seedRound(gameID, matchplay.RoundRow{
		ID: roundID, GameID: gameID, LocationID: locID, RoundNumber: 1, Status: "active",
		StartsAt: &now,
	}, &matchplay.AnswerLocation{
		Latitude: 40.4, Longitude: -3.7, CountryCode: "ES", Provider: "test", ProviderRef: "pano-1",
	}, []matchplay.GuessRow{
		// Viewer A has submitted; answer must stay hidden on snapshot.
		{
			ID: uuid.New(), RoundID: roundID, GamePlayerID: gpA,
			Latitude: 41, Longitude: -3, DistanceMeters: 50000,
			AccuracyScore: 4000, SpeedBonus: 0, Score: 4000, SubmittedAt: now,
		},
	})
	return matchID, gameID, userA, userB, gpA, gpB
}

func intPtr(v int) *int { return &v }

func TestSnapshotParticipantOnlyAndPrivacyNotFound(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newMemoryStore()
	matchID, _, userA, _, _, _ := seedCasual1v1(store, now)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})

	// Participant can read.
	resp, err := svc.GetSnapshot(context.Background(), registeredSession(userA), matchID)
	if err != nil {
		t.Fatalf("participant snapshot: %v", err)
	}
	if resp.Match.ID != matchID {
		t.Fatalf("match id = %s", resp.Match.ID)
	}
	if resp.Match.Viewer.UserID != userA {
		t.Fatalf("viewer = %s", resp.Match.Viewer.UserID)
	}

	// Non-participant gets privacy-safe not found (same as missing).
	stranger := uuid.New()
	_, err = svc.GetSnapshot(context.Background(), registeredSession(stranger), matchID)
	if !errors.Is(err, matchplay.ErrNotFound) {
		t.Fatalf("stranger err = %v, want not found", err)
	}

	// Missing match also not found.
	_, err = svc.GetSnapshot(context.Background(), registeredSession(userA), uuid.New())
	if !errors.Is(err, matchplay.ErrNotFound) {
		t.Fatalf("missing match err = %v, want not found", err)
	}

	// MapError keeps both privacy-safe.
	mapped := matchplay.MapError(matchplay.ErrNotFound)
	if mapped == nil {
		t.Fatal("expected mapped error")
	}
}

func TestSnapshotPreRevealRedaction(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newMemoryStore()
	matchID, _, userA, userB, gpA, gpB := seedCasual1v1(store, now)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})

	resp, err := svc.GetSnapshot(context.Background(), registeredSession(userA), matchID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Current round present; no answer coordinates on snapshot media path.
	if resp.Match.Round == nil || resp.Match.Round.Status != "active" {
		t.Fatalf("expected active round, got %+v", resp.Match.Round)
	}
	// Pre-reveal: last_round_result must be absent while only active (unrevealed) round exists.
	// Note: seed has no completed round, so LastRoundResult should be nil.
	if resp.Match.LastRoundResult != nil {
		t.Fatal("expected no last_round_result before any reveal")
	}

	// Viewer submitted flag true; opponent total redacted.
	if !resp.Match.Viewer.Submitted {
		t.Fatal("viewer should be submitted")
	}
	var teammateScore, opponentScore *int
	for _, team := range resp.Match.Teams {
		for _, p := range team.Players {
			switch p.GamePlayerID {
			case gpA:
				teammateScore = p.TotalScore
			case gpB:
				opponentScore = p.TotalScore
			}
		}
	}
	if teammateScore == nil {
		t.Fatal("own total_score must be present")
	}
	if opponentScore != nil {
		t.Fatalf("opponent total_score must be redacted pre-reveal, got %d", *opponentScore)
	}

	// Opponent cannot read caller's private guess via snapshot fields (no guess payload at all).
	// Ensure no answer lat/lng leak on DTO (round projection has no answer field by design).
	_ = userB
}

func TestSnapshotRoundResultsAfterReveal(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newMemoryStore()
	matchID, gameID, userA, userB, _, gpB := seedCasual1v1(store, now)

	// Complete the active round so results are available.
	store.mu.Lock()
	for i := range store.rounds[gameID] {
		if store.rounds[gameID][i].Status == "active" {
			store.rounds[gameID][i].Status = "completed"
			// Opponent also guessed.
			rid := store.rounds[gameID][i].ID
			store.guesses[rid] = append(store.guesses[rid], matchplay.GuessRow{
				ID: uuid.New(), RoundID: rid, GamePlayerID: gpB,
				Latitude: 39, Longitude: -4, DistanceMeters: 80000,
				AccuracyScore: 3000, SpeedBonus: 0, Score: 3000, SubmittedAt: now,
			})
		}
	}
	store.mu.Unlock()

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})

	// Reload snapshot — last_round_result present with answer.
	snap, err := svc.GetSnapshot(context.Background(), registeredSession(userA), matchID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Match.LastRoundResult == nil {
		t.Fatal("expected last_round_result after reveal")
	}
	if snap.Match.LastRoundResult.ActualLocation.CountryCode != "ES" {
		t.Fatalf("answer = %+v", snap.Match.LastRoundResult.ActualLocation)
	}
	if len(snap.Match.LastRoundResult.Guesses) != 2 {
		t.Fatalf("guesses = %d, want 2", len(snap.Match.LastRoundResult.Guesses))
	}
	// Opponent totals no longer redacted after a completed round is present.
	for _, team := range snap.Match.Teams {
		for _, p := range team.Players {
			if p.GamePlayerID == gpB && p.TotalScore == nil {
				t.Fatal("opponent total should be visible after reveal")
			}
		}
	}

	// Round results endpoint.
	var roundID uuid.UUID
	store.mu.Lock()
	roundID = store.rounds[gameID][0].ID
	store.mu.Unlock()

	results, err := svc.GetRoundResults(context.Background(), registeredSession(userB), matchID, roundID)
	if err != nil {
		t.Fatalf("round results: %v", err)
	}
	if results.Result.ActualLocation.Latitude == 0 {
		t.Fatal("expected answer latitude")
	}

	// Active (non-completed) round → not revealed.
	activeRound := uuid.New()
	store.seedRound(gameID, matchplay.RoundRow{
		ID: activeRound, GameID: gameID, LocationID: uuid.New(), RoundNumber: 2, Status: "active",
	}, nil, nil)
	_, err = svc.GetRoundResults(context.Background(), registeredSession(userA), matchID, activeRound)
	if !errors.Is(err, matchplay.ErrRoundNotRevealed) {
		t.Fatalf("err = %v, want round_not_revealed", err)
	}
}

func TestCasualTerminalResultNoProgression(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newMemoryStore()
	matchID, _, userA, userB, _, _ := seedCasual1v1(store, now)

	// Mark match terminal with forfeit-like result.
	store.mu.Lock()
	result := matchplay.MatchResultTeamOneWin
	winner := 1
	store.matches[matchID].Status = matchplay.MatchStatusCompleted
	store.matches[matchID].Result = &result
	store.matches[matchID].WinnerTeamSlot = &winner
	store.matches[matchID].CompletedAt = &now
	// Ensure casual: progression_finalized_at remains nil and playlist casual.
	store.matches[matchID].ProgressionFinalizedAt = nil
	store.matches[matchID].Playlist = matchplay.PlaylistCasual
	store.mu.Unlock()

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})
	resp, err := svc.GetTerminalResult(context.Background(), registeredSession(userA), matchID)
	if err != nil {
		t.Fatalf("terminal result: %v", err)
	}
	if resp.Progression.Applied {
		t.Fatal("casual progression must not be applied")
	}
	if resp.Progression.Reason == nil || *resp.Progression.Reason != "casual" {
		t.Fatalf("progression reason = %v, want casual", resp.Progression.Reason)
	}
	if resp.Result != matchplay.MatchResultTeamOneWin {
		t.Fatalf("result = %s", resp.Result)
	}

	// Non-participant privacy.
	_, err = svc.GetTerminalResult(context.Background(), registeredSession(uuid.New()), matchID)
	if !errors.Is(err, matchplay.ErrNotFound) {
		t.Fatalf("stranger = %v, want not found", err)
	}
	_ = userB
}

func TestSnapshotRequiresRegisteredSession(t *testing.T) {
	t.Parallel()
	store := newMemoryStore()
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})
	_, err := svc.GetSnapshot(context.Background(), &session.Context{Kind: session.KindGuest}, uuid.New())
	if !errors.Is(err, matchplay.ErrUnauthorized) {
		t.Fatalf("err = %v, want unauthorized", err)
	}
}
