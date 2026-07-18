package matchplay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
)

func seedCasualDuoWithParties(store *memoryStore, now time.Time) (matchID uuid.UUID, userA, userB uuid.UUID, partyA, partyB uuid.UUID) {
	matchID = uuid.New()
	gameID := uuid.New()
	userA, userB = uuid.New(), uuid.New()
	gpA, gpB := uuid.New(), uuid.New()
	partyA, partyB = uuid.New(), uuid.New()

	match := matchplay.Match{
		ID:             matchID,
		FormationKey:   "fk-duo-" + matchID.String(),
		GameID:         gameID,
		Mode:           "casual_duo",
		Status:         matchplay.MatchStatusActive,
		Playlist:       matchplay.PlaylistCasual,
		Format:         matchplay.FormatDuo,
		TeamSize:       2,
		LastActivityAt: now,
		MatchedAt:      now,
		StartedAt:      &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// Solo-style 1 player per team for simpler leave tests (team_size field may be 1 in 1v1).
	// For duo format with leave, use team_size 1 equivalent 1v1 with party IDs attached.
	match.TeamSize = 1
	match.Format = matchplay.FormatSolo
	match.Mode = "casual_solo"

	parts := []matchplay.MatchParticipant{
		{MatchID: matchID, UserID: userA, GamePlayerID: gpA, Status: matchplay.ParticipantStatusActive, TeamSlot: 1, PartyID: &partyA, AssignedAt: now},
		{MatchID: matchID, UserID: userB, GamePlayerID: gpB, Status: matchplay.ParticipantStatusActive, TeamSlot: 2, PartyID: &partyB, AssignedAt: now},
	}
	players := []matchplay.GamePlayerRow{
		{ID: gpA, GameID: gameID, UserID: &userA, DisplayName: "A", Status: matchplay.PlayerStatusActive, TotalScore: 0, TeamSlot: intPtr(1)},
		{ID: gpB, GameID: gameID, UserID: &userB, DisplayName: "B", Status: matchplay.PlayerStatusActive, TotalScore: 0, TeamSlot: intPtr(2)},
	}
	store.seedMatch(match, parts, players)
	return matchID, userA, userB, partyA, partyB
}

func TestRealtimeConnectionTransitionsPersistAndPublish(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userID, _, _, _ := seedCasualDuoWithParties(store, now)
	events := &fakeEvents{}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithEvents(events).
		WithClock(func() time.Time { return now })

	if err := svc.MarkRealtimeDisconnected(context.Background(), matchID, userID); err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if err := svc.MarkRealtimeConnected(context.Background(), matchID, userID); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	events.mu.Lock()
	defer events.mu.Unlock()
	if len(events.types) != 2 || events.types[0] != matchplay.EventMatchPlayerDisconnected || events.types[1] != matchplay.EventMatchPlayerReconnected {
		t.Fatalf("connection events = %v", events.types)
	}
}

func TestExplicitCasualLeaveForfeitAndPartyRestore(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, userB, partyA, partyB := seedCasualDuoWithParties(store, now)

	restorer := &fakePartyRestorer{}
	events := &fakeEvents{}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithPartyRestorer(restorer).
		WithEvents(events).
		WithClock(func() time.Time { return now })

	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("leave: %v", err)
	}

	// Match terminal with forfeit; no progression for casual.
	store.mu.Lock()
	match := store.matches[matchID]
	if match.Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s", match.Status)
	}
	if match.Result == nil || *match.Result != matchplay.MatchResultForfeit {
		t.Fatalf("result = %v", match.Result)
	}
	if match.WinnerTeamSlot == nil || *match.WinnerTeamSlot != 2 {
		t.Fatalf("winner = %v, want team 2", match.WinnerTeamSlot)
	}
	if match.ProgressionFinalizedAt != nil {
		t.Fatal("casual leave must not set progression_finalized_at")
	}
	// Abandoner marked.
	var abandoned bool
	for _, p := range store.participants[matchID] {
		if p.UserID == userA && p.AbandonedAt != nil && p.AbandonReason != nil &&
			*p.AbandonReason == matchplay.AbandonReasonExplicitLeave {
			abandoned = true
		}
	}
	if !abandoned {
		t.Fatal("expected abandoner marked with explicit_leave")
	}
	store.mu.Unlock()

	// Party restoration invoked for both parties.
	restorer.mu.Lock()
	if restorer.calls < 1 {
		t.Fatal("expected party restore")
	}
	if restorer.matchID != matchID {
		t.Fatalf("restore match = %s", restorer.matchID)
	}
	gotParties := map[uuid.UUID]bool{}
	for _, id := range restorer.lastIDs {
		gotParties[id] = true
	}
	if !gotParties[partyA] || !gotParties[partyB] {
		t.Fatalf("restored parties = %v, want %s and %s", restorer.lastIDs, partyA, partyB)
	}
	restorer.mu.Unlock()

	// Events published.
	events.mu.Lock()
	if len(events.types) == 0 {
		t.Fatal("expected forfeit/completed events")
	}
	events.mu.Unlock()

	// Terminal result shows casual no-progression.
	resp, err := svc.GetTerminalResult(context.Background(), registeredSession(userB), matchID)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if resp.Progression.Applied {
		t.Fatal("casual progression applied")
	}
}

func TestLeaveIdempotentRetrySafety(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, _, _ := seedCasualDuoWithParties(store, now)
	restorer := &fakePartyRestorer{}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{}).
		WithPartyRestorer(restorer).
		WithClock(func() time.Time { return now })

	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("first leave: %v", err)
	}
	callsAfterFirst := store.leaveCalls
	restorer.mu.Lock()
	restoresAfterFirst := restorer.calls
	restorer.mu.Unlock()

	// Retry must succeed without error (idempotent 204 path).
	if err := svc.Leave(context.Background(), registeredSession(userA), matchID, matchplay.LeaveRequest{}); err != nil {
		t.Fatalf("retry leave: %v", err)
	}
	if store.leaveCalls <= callsAfterFirst {
		t.Fatal("expected leave store call on retry (idempotent path)")
	}
	// Winner slot unchanged.
	store.mu.Lock()
	match := store.matches[matchID]
	if match.WinnerTeamSlot == nil || *match.WinnerTeamSlot != 2 {
		t.Fatalf("winner mutated on retry: %v", match.WinnerTeamSlot)
	}
	store.mu.Unlock()

	// Party restore still safe to call again.
	restorer.mu.Lock()
	if restorer.calls < restoresAfterFirst {
		t.Fatal("restore calls decreased")
	}
	restorer.mu.Unlock()
}

func TestDisconnectGrace90Seconds(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, userB, _, _ := seedCasualDuoWithParties(store, base)

	// Mark userA disconnected 91s ago.
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

	restorer := &fakePartyRestorer{}
	clockNow := base
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
	}).WithPartyRestorer(restorer).WithClock(func() time.Time { return clockNow })

	if err := svc.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	store.mu.Lock()
	match := store.matches[matchID]
	if match.Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s after disconnect grace", match.Status)
	}
	if match.Result == nil || *match.Result != matchplay.MatchResultForfeit {
		t.Fatalf("result = %v", match.Result)
	}
	// No progression for casual disconnect forfeit.
	if match.ProgressionFinalizedAt != nil {
		t.Fatal("disconnect forfeit must not finalize progression")
	}
	store.mu.Unlock()

	if restorer.calls < 1 {
		t.Fatal("expected party restore after disconnect forfeit")
	}
	_ = userB
}

func TestDisconnectWithinGraceNotForfeited(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, _, _ := seedCasualDuoWithParties(store, base)

	// Disconnected only 30s ago — still within 90s grace.
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

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace: 90 * time.Second,
	}).WithClock(func() time.Time { return base })

	if err := svc.SweepDisconnectGrace(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	store.mu.Lock()
	if store.matches[matchID].Status != matchplay.MatchStatusActive {
		t.Fatalf("status = %s, want still active within grace", store.matches[matchID].Status)
	}
	store.mu.Unlock()
}

func TestCasualInactivityTenMinutes(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, userA, _, partyA, partyB := seedCasualDuoWithParties(store, base)

	// Stale activity 11 minutes ago.
	store.mu.Lock()
	store.matches[matchID].LastActivityAt = base.Add(-11 * time.Minute)
	store.mu.Unlock()

	restorer := &fakePartyRestorer{}
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		CasualInactivity: 10 * time.Minute,
	}).WithPartyRestorer(restorer).WithClock(func() time.Time { return base })

	if err := svc.SweepCasualInactivity(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	store.mu.Lock()
	match := store.matches[matchID]
	if match.Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s", match.Status)
	}
	if match.Result == nil || *match.Result != matchplay.MatchResultAbandoned {
		t.Fatalf("result = %v, want abandoned", match.Result)
	}
	if match.ProgressionFinalizedAt != nil {
		t.Fatal("inactivity close must not finalize progression")
	}
	store.mu.Unlock()

	restorer.mu.Lock()
	if restorer.calls < 1 {
		t.Fatal("expected party restore after inactivity")
	}
	got := map[uuid.UUID]bool{}
	for _, id := range restorer.lastIDs {
		got[id] = true
	}
	if !got[partyA] || !got[partyB] {
		t.Fatalf("parties = %v", restorer.lastIDs)
	}
	restorer.mu.Unlock()
	_ = userA
}

func TestInactivityWithinWindowKeepsMatch(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	matchID, _, _, _, _ := seedCasualDuoWithParties(store, base)
	store.mu.Lock()
	store.matches[matchID].LastActivityAt = base.Add(-5 * time.Minute)
	store.mu.Unlock()

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		CasualInactivity: 10 * time.Minute,
	}).WithClock(func() time.Time { return base })

	if err := svc.SweepCasualInactivity(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	store.mu.Lock()
	if store.matches[matchID].Status != matchplay.MatchStatusActive {
		t.Fatalf("status = %s, want active", store.matches[matchID].Status)
	}
	store.mu.Unlock()
}

func TestLeaveNonParticipantPrivacy(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	store := newMemoryStore()
	matchID, _, _, _, _ := seedCasualDuoWithParties(store, now)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})

	err := svc.Leave(context.Background(), registeredSession(uuid.New()), matchID, matchplay.LeaveRequest{})
	if !errors.Is(err, matchplay.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestRunLifecycleSweepCombinesWorkers(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()

	// One match past inactivity.
	matchID, _, _, _, _ := seedCasualDuoWithParties(store, base)
	store.mu.Lock()
	store.matches[matchID].LastActivityAt = base.Add(-20 * time.Minute)
	store.mu.Unlock()

	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{
		ReconnectGrace:   90 * time.Second,
		CasualInactivity: 10 * time.Minute,
	}).WithClock(func() time.Time { return base })

	if err := svc.RunLifecycleSweep(context.Background()); err != nil {
		t.Fatalf("lifecycle sweep: %v", err)
	}
	store.mu.Lock()
	if store.matches[matchID].Status != matchplay.MatchStatusCompleted {
		t.Fatalf("status = %s", store.matches[matchID].Status)
	}
	store.mu.Unlock()
}
