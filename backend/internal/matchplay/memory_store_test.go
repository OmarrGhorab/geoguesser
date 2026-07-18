package matchplay_test

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
)

// memoryStore is an in-memory Store for unit tests (no DATABASE_URL required).
type memoryStore struct {
	mu           sync.Mutex
	matches      map[uuid.UUID]*matchplay.Match
	participants map[uuid.UUID][]matchplay.MatchParticipant // matchID -> parts
	players      map[uuid.UUID][]matchplay.GamePlayerRow    // gameID -> players
	rounds       map[uuid.UUID][]matchplay.RoundRow         // gameID -> rounds
	guesses      map[uuid.UUID][]matchplay.GuessRow         // roundID -> guesses
	answers      map[uuid.UUID]matchplay.AnswerLocation     // locationID -> answer
	leaveCalls   int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		matches:      map[uuid.UUID]*matchplay.Match{},
		participants: map[uuid.UUID][]matchplay.MatchParticipant{},
		players:      map[uuid.UUID][]matchplay.GamePlayerRow{},
		rounds:       map[uuid.UUID][]matchplay.RoundRow{},
		guesses:      map[uuid.UUID][]matchplay.GuessRow{},
		answers:      map[uuid.UUID]matchplay.AnswerLocation{},
	}
}

func (m *memoryStore) seedMatch(match matchplay.Match, parts []matchplay.MatchParticipant, players []matchplay.GamePlayerRow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := match
	m.matches[match.ID] = &cp
	m.participants[match.ID] = append([]matchplay.MatchParticipant{}, parts...)
	m.players[match.GameID] = append([]matchplay.GamePlayerRow{}, players...)
}

func (m *memoryStore) seedRound(gameID uuid.UUID, round matchplay.RoundRow, answer *matchplay.AnswerLocation, guesses []matchplay.GuessRow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rounds[gameID] = append(m.rounds[gameID], round)
	if answer != nil {
		m.answers[round.LocationID] = *answer
	}
	if len(guesses) > 0 {
		m.guesses[round.ID] = append([]matchplay.GuessRow{}, guesses...)
	}
}

func (m *memoryStore) LoadSnapshotBundle(_ context.Context, matchID uuid.UUID) (*matchplay.SnapshotBundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok {
		return nil, nil
	}
	parts := m.participants[matchID]
	players := m.players[match.GameID]
	bundle := &matchplay.SnapshotBundle{
		Match:        *match,
		Participants: append([]matchplay.MatchParticipant{}, parts...),
		Players:      append([]matchplay.GamePlayerRow{}, players...),
		SubmittedIDs: map[uuid.UUID]bool{},
	}
	for _, r := range m.rounds[match.GameID] {
		if r.Status == "active" {
			rr := r
			bundle.CurrentRound = &rr
			for _, g := range m.guesses[r.ID] {
				bundle.SubmittedIDs[g.GamePlayerID] = true
			}
			eligible := 0
			for _, p := range players {
				if p.Status == matchplay.PlayerStatusActive {
					eligible++
				}
			}
			bundle.EligibleCount = eligible
		}
		if r.Status == "completed" {
			if bundle.LastCompletedRound == nil || r.RoundNumber > bundle.LastCompletedRound.RoundNumber {
				rr := r
				bundle.LastCompletedRound = &rr
				bundle.LastRoundGuesses = append([]matchplay.GuessRow{}, m.guesses[r.ID]...)
				if ans, ok := m.answers[r.LocationID]; ok {
					a := ans
					bundle.LastRoundAnswer = &a
				}
			}
		}
	}
	return bundle, nil
}

func (m *memoryStore) LoadRoundResultBundle(_ context.Context, matchID, roundID uuid.UUID) (*matchplay.RoundResultBundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok {
		return nil, nil
	}
	var round *matchplay.RoundRow
	for i := range m.rounds[match.GameID] {
		if m.rounds[match.GameID][i].ID == roundID {
			round = &m.rounds[match.GameID][i]
			break
		}
	}
	if round == nil {
		return nil, nil
	}
	if round.Status != "completed" {
		return nil, matchplay.ErrRoundNotRevealed
	}
	ans, ok := m.answers[round.LocationID]
	if !ok {
		return nil, matchplay.ErrUnavailable
	}
	return &matchplay.RoundResultBundle{
		Match:        *match,
		Participants: append([]matchplay.MatchParticipant{}, m.participants[matchID]...),
		Players:      append([]matchplay.GamePlayerRow{}, m.players[match.GameID]...),
		Round:        *round,
		Guesses:      append([]matchplay.GuessRow{}, m.guesses[round.ID]...),
		Answer:       ans,
		TeamOneScore: match.TeamOneScore,
		TeamTwoScore: match.TeamTwoScore,
	}, nil
}

func (m *memoryStore) LoadTerminalResultBundle(_ context.Context, matchID uuid.UUID) (*matchplay.TerminalResultBundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok {
		return nil, nil
	}
	if !matchplay.IsTerminalMatch(match.Status) {
		return nil, matchplay.ErrMatchNotActive
	}
	out := &matchplay.TerminalResultBundle{
		Match:        *match,
		Participants: append([]matchplay.MatchParticipant{}, m.participants[matchID]...),
		Players:      append([]matchplay.GamePlayerRow{}, m.players[match.GameID]...),
	}
	for _, r := range m.rounds[match.GameID] {
		if r.Status != "completed" {
			continue
		}
		ans, ok := m.answers[r.LocationID]
		if !ok {
			continue
		}
		out.Rounds = append(out.Rounds, matchplay.RoundResultBundle{
			Match:        *match,
			Participants: out.Participants,
			Players:      out.Players,
			Round:        r,
			Guesses:      append([]matchplay.GuessRow{}, m.guesses[r.ID]...),
			Answer:       ans,
			TeamOneScore: match.TeamOneScore,
			TeamTwoScore: match.TeamTwoScore,
		})
	}
	return out, nil
}

func (m *memoryStore) FindParticipant(_ context.Context, matchID, userID uuid.UUID) (*matchplay.MatchParticipant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.participants[matchID] {
		if p.UserID == userID {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memoryStore) MarkConnectionState(_ context.Context, matchID, userID uuid.UUID, connected bool, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok || matchplay.IsTerminalMatch(match.Status) {
		return false, nil
	}
	for _, part := range m.participants[matchID] {
		if part.UserID != userID || !matchplay.IsActiveParticipant(part.Status) || part.AbandonedAt != nil {
			continue
		}
		players := m.players[match.GameID]
		for i := range players {
			if players[i].ID != part.GamePlayerID {
				continue
			}
			if connected {
				if players[i].Status != matchplay.PlayerStatusDisconnected {
					return false, nil
				}
				players[i].Status = matchplay.PlayerStatusActive
				players[i].LeftAt = nil
			} else {
				if players[i].Status != matchplay.PlayerStatusActive {
					return false, nil
				}
				players[i].Status = matchplay.PlayerStatusDisconnected
				at = at.UTC()
				players[i].LeftAt = &at
			}
			m.players[match.GameID] = players
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryStore) ExplicitLeaveTx(_ context.Context, matchID, userID uuid.UUID, now time.Time) (*matchplay.LeaveOutcome, error) {
	return m.forfeit(matchID, userID, now, matchplay.AbandonReasonExplicitLeave)
}

func (m *memoryStore) ForfeitDisconnectTx(_ context.Context, matchID, userID uuid.UUID, now time.Time) (*matchplay.LeaveOutcome, error) {
	return m.forfeit(matchID, userID, now, matchplay.AbandonReasonDisconnectTimeout)
}

func (m *memoryStore) forfeit(matchID, userID uuid.UUID, now time.Time, reason string) (*matchplay.LeaveOutcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaveCalls++
	match, ok := m.matches[matchID]
	if !ok {
		return nil, matchplay.ErrNotFound
	}
	parts := m.participants[matchID]
	var part *matchplay.MatchParticipant
	idx := -1
	for i := range parts {
		if parts[i].UserID == userID {
			part = &parts[i]
			idx = i
			break
		}
	}
	if part == nil {
		return nil, matchplay.ErrNotFound
	}
	if matchplay.IsTerminalMatch(match.Status) {
		return &matchplay.LeaveOutcome{
			Match:            *match,
			AlreadyTerminal:  true,
			RestoredPartyIDs: partyIDs(parts),
		}, nil
	}
	if part.AbandonedAt != nil {
		return &matchplay.LeaveOutcome{Match: *match}, nil
	}
	now = now.UTC()
	reasonCopy := reason
	parts[idx].AbandonedAt = &now
	parts[idx].AbandonReason = &reasonCopy
	parts[idx].Status = matchplay.ParticipantStatusCompleted
	parts[idx].CompletedAt = &now
	m.participants[matchID] = parts

	winner := matchplay.WinningTeamForForfeit(part.TeamSlot)
	result := matchplay.MatchResultForfeit
	chatUntil := now.Add(matchplay.DefaultChatAccessWindow)
	match.Status = matchplay.MatchStatusCompleted
	match.Result = &result
	match.WinnerTeamSlot = &winner
	match.CompletedAt = &now
	match.ClosedAt = &now
	match.ChatAccessUntil = &chatUntil
	match.LastActivityAt = now
	// Progression never finalized here (casual neutral; ranked pending competitive).
	match.ProgressionFinalizedAt = nil

	// Mark game player left.
	players := m.players[match.GameID]
	for i := range players {
		if players[i].ID == part.GamePlayerID {
			players[i].Status = matchplay.PlayerStatusLeft
			players[i].LeftAt = &now
		}
	}
	m.players[match.GameID] = players

	// Complete remaining non-abandoner participants without abandon facts.
	for i := range parts {
		if parts[i].UserID == userID {
			continue
		}
		if parts[i].Status == matchplay.ParticipantStatusAssigned || parts[i].Status == matchplay.ParticipantStatusActive {
			parts[i].Status = matchplay.ParticipantStatusCompleted
			parts[i].CompletedAt = &now
			parts[i].ClosedAt = &now
		}
	}
	m.participants[matchID] = parts

	return &matchplay.LeaveOutcome{
		Match:            *match,
		AbandonedUserIDs: []uuid.UUID{userID},
		RestoredPartyIDs: partyIDs(parts),
	}, nil
}

func (m *memoryStore) CloseInactiveCasualTx(_ context.Context, matchID uuid.UUID, now time.Time) (*matchplay.LeaveOutcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok {
		return nil, matchplay.ErrNotFound
	}
	parts := m.participants[matchID]
	if matchplay.IsTerminalMatch(match.Status) {
		return &matchplay.LeaveOutcome{
			Match:            *match,
			AlreadyTerminal:  true,
			RestoredPartyIDs: partyIDs(parts),
		}, nil
	}
	if !matchplay.IsCasualMatch(*match) {
		return nil, matchplay.ErrInvalidLeave
	}
	now = now.UTC()
	result := matchplay.MatchResultAbandoned
	chatUntil := now.Add(matchplay.DefaultChatAccessWindow)
	match.Status = matchplay.MatchStatusCompleted
	match.Result = &result
	match.CompletedAt = &now
	match.ClosedAt = &now
	match.ChatAccessUntil = &chatUntil
	match.LastActivityAt = now
	match.ProgressionFinalizedAt = nil
	return &matchplay.LeaveOutcome{
		Match:            *match,
		RestoredPartyIDs: partyIDs(parts),
	}, nil
}

func (m *memoryStore) ListDisconnectCandidates(_ context.Context, now time.Time, grace time.Duration, limit int) ([]matchplay.DisconnectCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := now.Add(-grace)
	var out []matchplay.DisconnectCandidate
	for matchID, match := range m.matches {
		if matchplay.IsTerminalMatch(match.Status) {
			continue
		}
		for _, p := range m.participants[matchID] {
			if p.AbandonedAt != nil || !matchplay.IsActiveParticipant(p.Status) {
				continue
			}
			for _, gp := range m.players[match.GameID] {
				if gp.ID != p.GamePlayerID {
					continue
				}
				if gp.Status != matchplay.PlayerStatusDisconnected {
					continue
				}
				at := match.UpdatedAt
				if gp.LeftAt != nil {
					at = *gp.LeftAt
				}
				if at.After(cutoff) {
					continue
				}
				out = append(out, matchplay.DisconnectCandidate{
					MatchID:        matchID,
					UserID:         p.UserID,
					GamePlayerID:   p.GamePlayerID,
					TeamSlot:       p.TeamSlot,
					DisconnectedAt: at,
					Playlist:       match.Playlist,
					GameID:         match.GameID,
					PartyID:        p.PartyID,
				})
			}
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryStore) ListInactiveCasualMatches(_ context.Context, cutoff time.Time, limit int) ([]matchplay.InactivityCandidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []matchplay.InactivityCandidate
	for matchID, match := range m.matches {
		if matchplay.IsTerminalMatch(match.Status) {
			continue
		}
		if !matchplay.IsCasualMatch(*match) {
			continue
		}
		if match.LastActivityAt.After(cutoff) {
			continue
		}
		out = append(out, matchplay.InactivityCandidate{
			MatchID:        matchID,
			GameID:         match.GameID,
			LastActivityAt: match.LastActivityAt,
			PartyIDs:       partyIDs(m.participants[matchID]),
		})
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryStore) TouchActivity(_ context.Context, matchID uuid.UUID, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if match, ok := m.matches[matchID]; ok {
		match.LastActivityAt = at.UTC()
		match.UpdatedAt = at.UTC()
	}
	return nil
}

func partyIDs(parts []matchplay.MatchParticipant) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	var out []uuid.UUID
	for _, p := range parts {
		if p.PartyID != nil && !seen[*p.PartyID] {
			seen[*p.PartyID] = true
			out = append(out, *p.PartyID)
		}
	}
	return out
}

// fakePartyRestorer records restore calls.
type fakePartyRestorer struct {
	mu      sync.Mutex
	calls   int
	lastIDs []uuid.UUID
	matchID uuid.UUID
	err     error
}

func (f *fakePartyRestorer) RestoreAfterTerminalMatch(_ context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.matchID = matchID
	f.lastIDs = append([]uuid.UUID{}, partyIDs...)
	return f.err
}

// fakeEvents records published events.
type fakeEvents struct {
	mu       sync.Mutex
	types    []string
	matchIDs []uuid.UUID
}

func (f *fakeEvents) PublishMatchEvent(_ context.Context, matchID uuid.UUID, eventType string, _ int64, _ *uuid.UUID, _ *uuid.UUID, _ any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.types = append(f.types, eventType)
	f.matchIDs = append(f.matchIDs, matchID)
	return nil
}

// fakePresence tracks reconnect windows.
type fakePresence struct {
	windows map[string]bool
}

func (f *fakePresence) key(matchID, userID uuid.UUID) string {
	return matchID.String() + ":" + userID.String()
}

func (f *fakePresence) HasReconnectWindow(_ context.Context, matchID, userID uuid.UUID) (bool, error) {
	if f.windows == nil {
		return false, nil
	}
	return f.windows[f.key(matchID, userID)], nil
}

func (f *fakePresence) ClearPresence(_ context.Context, matchID, userID uuid.UUID) error {
	if f.windows != nil {
		delete(f.windows, f.key(matchID, userID))
	}
	return nil
}
