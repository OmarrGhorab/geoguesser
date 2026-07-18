package matchplay_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
)

func seedRankedDuo(store *memoryStore, now time.Time) (matchID uuid.UUID, userA, userB, userC, userD uuid.UUID, partyA, partyB uuid.UUID) {
	matchID = uuid.New()
	gameID := uuid.New()
	userA, userB = uuid.New(), uuid.New()
	userC, userD = uuid.New(), uuid.New()
	gpA, gpB, gpC, gpD := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	partyA, partyB = uuid.New(), uuid.New()

	match := matchplay.Match{
		ID:             matchID,
		FormationKey:   "fk-ranked-duo-" + matchID.String(),
		GameID:         gameID,
		Mode:           "ranked_duo",
		Status:         matchplay.MatchStatusActive,
		Playlist:       matchplay.PlaylistRanked,
		Format:         matchplay.FormatDuo,
		TeamSize:       2,
		TeamOneScore:   5000,
		TeamTwoScore:   4800,
		LastActivityAt: now,
		MatchedAt:      now,
		StartedAt:      &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	parts := []matchplay.MatchParticipant{
		{MatchID: matchID, UserID: userA, GamePlayerID: gpA, Status: matchplay.ParticipantStatusActive, TeamSlot: 1, PartyID: &partyA, AssignedAt: now},
		{MatchID: matchID, UserID: userB, GamePlayerID: gpB, Status: matchplay.ParticipantStatusActive, TeamSlot: 1, PartyID: &partyA, AssignedAt: now},
		{MatchID: matchID, UserID: userC, GamePlayerID: gpC, Status: matchplay.ParticipantStatusActive, TeamSlot: 2, PartyID: &partyB, AssignedAt: now},
		{MatchID: matchID, UserID: userD, GamePlayerID: gpD, Status: matchplay.ParticipantStatusActive, TeamSlot: 2, PartyID: &partyB, AssignedAt: now},
	}
	players := []matchplay.GamePlayerRow{
		{ID: gpA, GameID: gameID, UserID: &userA, DisplayName: "A", Status: matchplay.PlayerStatusActive, TotalScore: 2500, TeamSlot: intPtr(1)},
		{ID: gpB, GameID: gameID, UserID: &userB, DisplayName: "B", Status: matchplay.PlayerStatusActive, TotalScore: 2500, TeamSlot: intPtr(1)},
		{ID: gpC, GameID: gameID, UserID: &userC, DisplayName: "C", Status: matchplay.PlayerStatusActive, TotalScore: 2400, TeamSlot: intPtr(2)},
		{ID: gpD, GameID: gameID, UserID: &userD, DisplayName: "D", Status: matchplay.PlayerStatusActive, TotalScore: 2400, TeamSlot: intPtr(2)},
	}
	store.seedMatch(match, parts, players)
	return matchID, userA, userB, userC, userD, partyA, partyB
}

func TestRankedExplicitLeave_TeamForfeitQuitterOnlyOpponentWin(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, userB, userC, userD, _, _ := seedRankedDuo(store, now)

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithClock(func() time.Time { return now })

	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("leave: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	match := store.matches[matchID]
	if match.Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s", match.Status)
	}
	if match.Result == nil || *match.Result != matchplay.MatchResultForfeit {
		t.Fatalf("result = %v, want forfeit", match.Result)
	}
	// Opponent team (slot 2) wins.
	if match.WinnerTeamSlot == nil || *match.WinnerTeamSlot != 2 {
		t.Fatalf("winner = %v, want team 2 (normal opponent win)", match.WinnerTeamSlot)
	}
	// Ranked progression must stay pending for competitive finalization (+ quitter -15).
	if match.ProgressionFinalizedAt != nil {
		t.Fatal("ranked abandon must leave progression_finalized_at null (pending retry)")
	}

	var abandoners int
	for _, p := range store.participants[matchID] {
		if p.AbandonedAt != nil {
			abandoners++
			if p.UserID != userA {
				t.Fatalf("non-quitter %s marked abandoned", p.UserID)
			}
			if p.AbandonReason == nil || *p.AbandonReason != matchplay.AbandonReasonExplicitLeave {
				t.Fatalf("abandon reason = %v", p.AbandonReason)
			}
		}
		// Teammate and opponents complete without abandon facts.
		if p.UserID == userB || p.UserID == userC || p.UserID == userD {
			if p.AbandonedAt != nil {
				t.Fatalf("user %s must not have abandon facts", p.UserID)
			}
			if p.Status != matchplay.ParticipantStatusCompleted {
				t.Fatalf("user %s status = %s, want completed", p.UserID, p.Status)
			}
		}
	}
	if abandoners != 1 {
		t.Fatalf("abandoners = %d, want 1 (quitter-only)", abandoners)
	}
}

func TestRankedLeave_TerminalReplaySafety(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, _, _, _, _ := seedRankedDuo(store, now)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithClock(func() time.Time { return now })

	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("first leave: %v", err)
	}
	store.mu.Lock()
	firstWinner := *store.matches[matchID].WinnerTeamSlot
	firstResult := *store.matches[matchID].Result
	store.mu.Unlock()

	// Replay must succeed without mutating terminal facts.
	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("replay leave: %v", err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	match := store.matches[matchID]
	if match.WinnerTeamSlot == nil || *match.WinnerTeamSlot != firstWinner {
		t.Fatalf("winner mutated on replay: %v", match.WinnerTeamSlot)
	}
	if match.Result == nil || *match.Result != firstResult {
		t.Fatalf("result mutated on replay: %v", match.Result)
	}
	// Still only one abandoner.
	n := 0
	for _, p := range store.participants[matchID] {
		if p.AbandonedAt != nil {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("abandoners after replay = %d", n)
	}
}

func TestRankedDisconnect_GraceThenForfeit(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, userC, _, _, _ := seedRankedDuo(store, base)

	// Disconnect past 90s grace.
	disconnectedAt := base.Add(-91 * time.Second)
	store.mu.Lock()
	players := store.players[store.matches[matchID].GameID]
	for i := range players {
		if players[i].UserID != nil && *players[i].UserID == userA {
			players[i].Status = matchplay.PlayerStatusDisconnected
			players[i].LeftAt = &disconnectedAt
		}
	}
	store.players[store.matches[matchID].GameID] = players
	store.mu.Unlock()

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
	}).WithClock(func() time.Time { return base })

	if err := svc.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	match := store.matches[matchID]
	if match.Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s", match.Status)
	}
	if match.Result == nil || *match.Result != matchplay.MatchResultForfeit {
		t.Fatalf("result = %v", match.Result)
	}
	if match.WinnerTeamSlot == nil || *match.WinnerTeamSlot != 2 {
		t.Fatalf("winner = %v, want opponent team 2", match.WinnerTeamSlot)
	}
	if match.ProgressionFinalizedAt != nil {
		t.Fatal("disconnect forfeit must leave progression pending")
	}
	// Quitter-only abandon.
	for _, p := range store.participants[matchID] {
		if p.UserID == userA {
			if p.AbandonedAt == nil || p.AbandonReason == nil ||
				*p.AbandonReason != matchplay.AbandonReasonDisconnectTimeout {
				t.Fatalf("quitter abandon facts missing: %+v", p)
			}
		} else if p.AbandonedAt != nil {
			t.Fatalf("non-quitter %s abandoned", p.UserID)
		}
	}
	_ = userC
}

func TestRankedDisconnect_WithinGraceNotForfeited_ReconnectRecovery(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, _, _, _, _ := seedRankedDuo(store, base)

	// Disconnected only 30s ago — within 90s grace (reconnect recovery window).
	disconnectedAt := base.Add(-30 * time.Second)
	store.mu.Lock()
	players := store.players[store.matches[matchID].GameID]
	for i := range players {
		if players[i].UserID != nil && *players[i].UserID == userA {
			players[i].Status = matchplay.PlayerStatusDisconnected
			players[i].LeftAt = &disconnectedAt
		}
	}
	store.players[store.matches[matchID].GameID] = players
	store.mu.Unlock()

	// Presence still has reconnect window → do not forfeit.
	presence := &fakePresence{windows: map[string]bool{
		matchID.String() + ":" + userA.String(): true,
	}}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
	}).WithPresence(presence).WithClock(func() time.Time { return base })

	if err := svc.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	store.mu.Lock()
	if store.matches[matchID].Status != matchplay.MatchStatusActive {
		t.Fatalf("status = %s, want still active during reconnect grace", store.matches[matchID].Status)
	}
	store.mu.Unlock()
}

func TestRankedTerminalResult_ProgressionPending(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, userC, _, _, _ := seedRankedDuo(store, now)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithClock(func() time.Time { return now })

	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("leave: %v", err)
	}

	// Winner reads terminal result → progression still pending (202 path).
	_, err := svc.GetTerminalResult(context.Background(), registeredSession(userC), matchID)
	if err != matchplay.ErrProgressionPending {
		t.Fatalf("err = %v, want ErrProgressionPending", err)
	}
}
