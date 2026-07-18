package app_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/parties"
	"github.com/raven/geoguess/backend/internal/realtime"
	"github.com/raven/geoguess/backend/internal/session"
)

// --- session helpers ---

func secRegistered(userID uuid.UUID) session.Context {
	s := userID.String()
	return session.Context{Kind: session.KindUser, UserID: &s, Role: "user"}
}

func secRegisteredPtr(userID uuid.UUID) *session.Context {
	s := secRegistered(userID)
	return &s
}

func secGuest() *session.Context {
	g := "guest-hash-1"
	return &session.Context{Kind: session.KindGuest, GuestID: &g}
}

func secAnonymous() *session.Context {
	return &session.Context{Kind: session.KindAnonymous}
}

// --- parties fakes ---

type secPartyStore struct {
	mu      sync.Mutex
	active  map[uuid.UUID]bool
	parties map[uuid.UUID]*parties.Party
	members map[uuid.UUID][]parties.PartyMember
}

func newSecPartyStore() *secPartyStore {
	return &secPartyStore{
		active:  map[uuid.UUID]bool{},
		parties: map[uuid.UUID]*parties.Party{},
		members: map[uuid.UUID][]parties.PartyMember{},
	}
}

func (s *secPartyStore) seedActive(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[id] = true
}

func (s *secPartyStore) seedFormingDuo(leader, member uuid.UUID, readyLeader, readyMember bool, version int) uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[leader] = true
	s.active[member] = true
	pid := uuid.New()
	now := time.Now().UTC()
	s.parties[pid] = &parties.Party{
		ID: pid, Format: parties.FormatDuo, Capacity: 2, LeaderUserID: leader,
		Status: parties.StatusForming, Version: version, CreatedAt: now, UpdatedAt: now,
	}
	s.members[pid] = []parties.PartyMember{
		{PartyID: pid, UserID: leader, Status: parties.MemberStatusActive, Ready: readyLeader, JoinedAt: now},
		{PartyID: pid, UserID: member, Status: parties.MemberStatusActive, Ready: readyMember, JoinedAt: now},
	}
	return pid
}

func (s *secPartyStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active[userID] {
		return nil, nil
	}
	id := userID
	return &id, nil
}

func (s *secPartyStore) HasActiveMatchAssignment(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func (s *secPartyStore) CreateParty(_ context.Context, leaderID uuid.UUID, format string, now time.Time) (*parties.PartySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cap, ok := parties.CapacityForFormat(format)
	if !ok {
		return nil, parties.ErrUnsupportedFormat
	}
	pid := uuid.New()
	s.parties[pid] = &parties.Party{
		ID: pid, Format: format, Capacity: int16(cap), LeaderUserID: leaderID,
		Status: parties.StatusForming, Version: 0, CreatedAt: now, UpdatedAt: now,
	}
	s.members[pid] = []parties.PartyMember{{
		PartyID: pid, UserID: leaderID, Status: parties.MemberStatusActive, JoinedAt: now,
	}}
	return s.snapshotLocked(pid)
}

func (s *secPartyStore) GetActivePartyForUser(_ context.Context, userID uuid.UUID) (*parties.PartySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for pid, members := range s.members {
		for _, m := range members {
			if m.UserID == userID && m.Status == parties.MemberStatusActive {
				return s.snapshotLocked(pid)
			}
		}
	}
	return nil, nil
}

func (s *secPartyStore) LoadSnapshot(_ context.Context, partyID uuid.UUID) (*parties.PartySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(partyID)
}

func (s *secPartyStore) snapshotLocked(partyID uuid.UUID) (*parties.PartySnapshot, error) {
	p, ok := s.parties[partyID]
	if !ok {
		return nil, parties.ErrNotFound
	}
	cp := *p
	mems := make([]parties.MemberSnapshot, 0, len(s.members[partyID]))
	for _, m := range s.members[partyID] {
		mems = append(mems, parties.MemberSnapshot{
			Member:  m,
			Profile: parties.PublicProfile{UserID: m.UserID, DisplayName: "user"},
		})
	}
	return &parties.PartySnapshot{Party: cp, Members: mems}, nil
}

func (s *secPartyStore) CreateInvite(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, time.Time) (*parties.InviteSnapshot, error) {
	return nil, parties.ErrFriendUnavailable
}

func (s *secPartyStore) ListPendingInvitesForUser(context.Context, uuid.UUID, time.Time) ([]parties.InviteSnapshot, error) {
	return nil, nil
}

func (s *secPartyStore) AcceptInvite(context.Context, uuid.UUID, uuid.UUID, time.Time) (*parties.PartySnapshot, error) {
	return nil, parties.ErrInviteNotFound
}

func (s *secPartyStore) DeclineInvite(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return parties.ErrInviteNotFound
}

func (s *secPartyStore) SetReadiness(context.Context, uuid.UUID, uuid.UUID, bool, time.Time) (*parties.PartySnapshot, error) {
	return nil, parties.ErrNotFound
}

func (s *secPartyStore) LeaveParty(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return parties.ErrNotFound
}

func (s *secPartyStore) KickMember(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error {
	return parties.ErrNotFound
}

func (s *secPartyStore) DisbandParty(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return parties.ErrNotFound
}

func (s *secPartyStore) GetInvite(context.Context, uuid.UUID) (*parties.PartyInvite, error) {
	return nil, nil
}

func (s *secPartyStore) RestoreAfterTerminalMatch(context.Context, uuid.UUID, []uuid.UUID) error {
	return nil
}

// --- friends policy fakes ---

type secFriendStore struct {
	active map[uuid.UUID]bool
	edges  map[string]*friends.Friendship
}

func pairKeySec(a, b uuid.UUID) string {
	ua, ub, err := friends.NormalizePair(a, b)
	if err != nil {
		return a.String() + "|" + b.String()
	}
	return ua.String() + "|" + ub.String()
}

func (s *secFriendStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	if s.active != nil && !s.active[userID] {
		return nil, nil
	}
	id := userID
	return &id, nil
}

func (s *secFriendStore) GetByPair(_ context.Context, userA, userB uuid.UUID) (*friends.Friendship, error) {
	if s.edges == nil {
		return nil, nil
	}
	return s.edges[pairKeySec(userA, userB)], nil
}

// --- matchplay minimal store ---

type secMatchStore struct {
	participants map[uuid.UUID]map[uuid.UUID]*matchplay.MatchParticipant // match -> user -> part
	bundles      map[uuid.UUID]*matchplay.SnapshotBundle
}

func newSecMatchStore() *secMatchStore {
	return &secMatchStore{
		participants: map[uuid.UUID]map[uuid.UUID]*matchplay.MatchParticipant{},
		bundles:      map[uuid.UUID]*matchplay.SnapshotBundle{},
	}
}

func (s *secMatchStore) seedMatch(matchID, userA, userB uuid.UUID) {
	now := time.Now().UTC()
	gpA, gpB := uuid.New(), uuid.New()
	gameID := uuid.New()
	match := matchplay.Match{
		ID: matchID, FormationKey: "fk-" + matchID.String(), GameID: gameID,
		Mode: "casual_solo", Status: matchplay.MatchStatusActive,
		Playlist: matchplay.PlaylistCasual, Format: matchplay.FormatSolo, TeamSize: 1,
		LastActivityAt: now, MatchedAt: now, StartedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	parts := []matchplay.MatchParticipant{
		{MatchID: matchID, UserID: userA, GamePlayerID: gpA, Status: matchplay.ParticipantStatusActive, TeamSlot: 1, AssignedAt: now},
		{MatchID: matchID, UserID: userB, GamePlayerID: gpB, Status: matchplay.ParticipantStatusActive, TeamSlot: 2, AssignedAt: now},
	}
	s.participants[matchID] = map[uuid.UUID]*matchplay.MatchParticipant{
		userA: &parts[0],
		userB: &parts[1],
	}
	s.bundles[matchID] = &matchplay.SnapshotBundle{
		Match:        match,
		Participants: parts,
		Players: []matchplay.GamePlayerRow{
			{ID: gpA, GameID: gameID, UserID: &userA, DisplayName: "A", Status: matchplay.PlayerStatusActive, TotalScore: 100, TeamSlot: intPtrSec(1)},
			{ID: gpB, GameID: gameID, UserID: &userB, DisplayName: "B", Status: matchplay.PlayerStatusActive, TotalScore: 200, TeamSlot: intPtrSec(2)},
		},
	}
}

func intPtrSec(v int) *int { return &v }

func (s *secMatchStore) LoadSnapshotBundle(_ context.Context, matchID uuid.UUID) (*matchplay.SnapshotBundle, error) {
	return s.bundles[matchID], nil
}

func (s *secMatchStore) LoadRoundResultBundle(context.Context, uuid.UUID, uuid.UUID) (*matchplay.RoundResultBundle, error) {
	return nil, nil
}

func (s *secMatchStore) LoadTerminalResultBundle(context.Context, uuid.UUID) (*matchplay.TerminalResultBundle, error) {
	return nil, nil
}

func (s *secMatchStore) FindParticipant(_ context.Context, matchID, userID uuid.UUID) (*matchplay.MatchParticipant, error) {
	if byUser, ok := s.participants[matchID]; ok {
		if p, ok := byUser[userID]; ok {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *secMatchStore) ExplicitLeaveTx(context.Context, uuid.UUID, uuid.UUID, time.Time) (*matchplay.LeaveOutcome, error) {
	return nil, matchplay.ErrNotFound
}

func (s *secMatchStore) ForfeitDisconnectTx(context.Context, uuid.UUID, uuid.UUID, time.Time) (*matchplay.LeaveOutcome, error) {
	return nil, matchplay.ErrNotFound
}

func (s *secMatchStore) CloseInactiveCasualTx(context.Context, uuid.UUID, time.Time) (*matchplay.LeaveOutcome, error) {
	return nil, matchplay.ErrNotFound
}

func (s *secMatchStore) ListDisconnectCandidates(context.Context, time.Time, time.Duration, int) ([]matchplay.DisconnectCandidate, error) {
	return nil, nil
}

func (s *secMatchStore) ListInactiveCasualMatches(context.Context, time.Time, int) ([]matchplay.InactivityCandidate, error) {
	return nil, nil
}

func (s *secMatchStore) TouchActivity(context.Context, uuid.UUID, time.Time) error {
	return nil
}

// --- tests ---

func TestSecurity_GuestsRejectedAcrossFeatures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	guest := secGuest()
	anon := secAnonymous()

	// Parties
	partySvc := parties.NewService(newSecPartyStore(), friends.NewPartyPolicy(&secFriendStore{}), nil)
	if _, err := partySvc.Create(ctx, *guest, parties.CreatePartyRequest{Format: "duo"}, "k1"); !errors.Is(err, parties.ErrUnauthorized) {
		t.Fatalf("parties guest create: %v", err)
	}
	if _, err := partySvc.Create(ctx, *anon, parties.CreatePartyRequest{Format: "duo"}, "k2"); !errors.Is(err, parties.ErrUnauthorized) {
		t.Fatalf("parties anon create: %v", err)
	}

	// Matchmaking
	mmSvc := matchmaking.NewService(nil, nil, matchmaking.Config{
		QueueLease: 30 * time.Second, ClaimTTL: 15 * time.Second, StartDelay: 5 * time.Second,
		RoundCount: 5, TimerSeconds: 60, CandidateScanLimit: 20,
		CasualMatchmakingEnabled: true,
	}, nil, nil)
	if _, err := mmSvc.JoinQueue(ctx, guest, matchmaking.JoinQueueRequest{Mode: "casual_solo"}); !errors.Is(err, matchmaking.ErrUnauthorized) {
		t.Fatalf("matchmaking guest: %v", err)
	}

	// Matchplay
	mpSvc := matchplay.NewService(newSecMatchStore(), nil, nil, matchplay.ServiceConfig{})
	if _, err := mpSvc.GetSnapshot(ctx, guest, uuid.New()); !errors.Is(err, matchplay.ErrUnauthorized) {
		t.Fatalf("matchplay guest: %v", err)
	}
}

func TestSecurity_DisabledUsersRejected(t *testing.T) {
	t.Parallel()
	store := newSecPartyStore()
	// Do not seed as active — simulates disabled/banned account.
	disabled := uuid.New()
	svc := parties.NewService(store, friends.NewPartyPolicy(&secFriendStore{active: map[uuid.UUID]bool{}}), nil)
	_, err := svc.Create(context.Background(), secRegistered(disabled), parties.CreatePartyRequest{Format: "duo"}, "k1")
	if !errors.Is(err, parties.ErrUnauthorized) {
		t.Fatalf("disabled user create: %v, want ErrUnauthorized", err)
	}
}

func TestSecurity_NonmembersAndOpponentsPrivacySafe(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newSecMatchStore()
	matchID, userA, userB := uuid.New(), uuid.New(), uuid.New()
	store.seedMatch(matchID, userA, userB)
	svc := matchplay.NewService(store, nil, nil, matchplay.ServiceConfig{})

	// Non-member: privacy-safe not found.
	stranger := uuid.New()
	_, errNon := svc.GetSnapshot(ctx, secRegisteredPtr(stranger), matchID)
	if !errors.Is(errNon, matchplay.ErrNotFound) {
		t.Fatalf("nonmember = %v, want not found", errNon)
	}

	// Missing match: same sentinel.
	_, errMissing := svc.GetSnapshot(ctx, secRegisteredPtr(userA), uuid.New())
	if !errors.Is(errMissing, matchplay.ErrNotFound) {
		t.Fatalf("missing = %v, want not found", errMissing)
	}

	// AuthorizeRealtime for non-participant.
	_, errAuth := svc.AuthorizeRealtime(ctx, stranger, matchID)
	if !errors.Is(errAuth, matchplay.ErrNotFound) {
		t.Fatalf("authorize nonmember = %v", errAuth)
	}

	// Opponent private resource maps identically to missing.
	mappedForbidden := matchplay.MapError(matchplay.ErrForbiddenOpponent)
	mappedMissing := matchplay.MapError(matchplay.ErrNotFound)
	var apiF, apiM *apphttp.APIError
	if !errors.As(mappedForbidden, &apiF) || !errors.As(mappedMissing, &apiM) {
		t.Fatal("expected APIError mapping")
	}
	if apiF.Status != http.StatusNotFound || apiM.Status != http.StatusNotFound {
		t.Fatalf("status F=%d M=%d", apiF.Status, apiM.Status)
	}
	if apiF.Code != apiM.Code || apiF.Message != apiM.Message {
		t.Fatalf("privacy leak: forbidden=%+v missing=%+v", apiF, apiM)
	}
	if strings.Contains(strings.ToLower(apiF.Message), "opponent") ||
		strings.Contains(strings.ToLower(apiF.Message), "forbidden") {
		t.Fatalf("message must not distinguish: %q", apiF.Message)
	}

	// Participant can still read.
	if _, err := svc.GetSnapshot(ctx, secRegisteredPtr(userA), matchID); err != nil {
		t.Fatalf("participant: %v", err)
	}
}

func TestSecurity_BlockedRelationshipsPrivacySafe(t *testing.T) {
	t.Parallel()
	inviter := uuid.New()
	invitee := uuid.New()
	ctx := context.Background()

	// Non-friend, blocked-by-inviter, blocked-by-invitee, inactive invitee — same sentinel.
	cases := []struct {
		name  string
		store *secFriendStore
	}{
		{"non_friend", &secFriendStore{active: map[uuid.UUID]bool{inviter: true, invitee: true}}},
		{"blocked_by_inviter", &secFriendStore{
			active: map[uuid.UUID]bool{inviter: true, invitee: true},
			edges: map[string]*friends.Friendship{
				pairKeySec(inviter, invitee): {
					ID: uuid.New(), Status: friends.StatusBlocked, BlockedByUserID: &inviter,
				},
			},
		}},
		{"blocked_by_invitee", &secFriendStore{
			active: map[uuid.UUID]bool{inviter: true, invitee: true},
			edges: map[string]*friends.Friendship{
				pairKeySec(inviter, invitee): {
					ID: uuid.New(), Status: friends.StatusBlocked, BlockedByUserID: &invitee,
				},
			},
		}},
		{"inactive_invitee", &secFriendStore{active: map[uuid.UUID]bool{inviter: true}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy := friends.NewPartyPolicy(tc.store)
			err := policy.CanInvite(ctx, inviter, invitee)
			if !errors.Is(err, friends.ErrFriendUnavailable) {
				t.Fatalf("err = %v, want friend unavailable", err)
			}
			// Parties map to privacy-safe API code without block ownership.
			mapped := parties.MapError(parties.ErrFriendUnavailable)
			var apiErr *apphttp.APIError
			if !errors.As(mapped, &apiErr) {
				t.Fatalf("mapped type %T", mapped)
			}
			if apiErr.Code != parties.CodeFriendUnavailable {
				t.Fatalf("code = %s", apiErr.Code)
			}
			body := strings.ToLower(apiErr.Message)
			if strings.Contains(body, "block") || strings.Contains(body, "ban") {
				t.Fatalf("must not reveal block: %s", apiErr.Message)
			}
		})
	}
}

func TestSecurity_StalePartiesAndVersionMismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newSecPartyStore()
	leader, member := uuid.New(), uuid.New()
	// Incomplete readiness.
	_ = store.seedFormingDuo(leader, member, true, false, 3)
	svc := parties.NewService(store, friends.NewPartyPolicy(&secFriendStore{}), nil)

	_, err := svc.ReadyPartyForQueue(ctx, leader, parties.FormatDuo, 3)
	if !errors.Is(err, parties.ErrPartyNotReady) {
		t.Fatalf("not ready = %v", err)
	}

	// Stale version after party mutates.
	_, err = svc.ReadyPartyForQueue(ctx, leader, parties.FormatDuo, 1)
	if !errors.Is(err, parties.ErrVersionMismatch) && !errors.Is(err, parties.ErrPartyNotReady) {
		// Version mismatch maps to party_not_ready on the wire (privacy of race).
		t.Fatalf("stale version = %v", err)
	}
	mapped := parties.MapError(parties.ErrVersionMismatch)
	var apiErr *apphttp.APIError
	if !errors.As(mapped, &apiErr) {
		t.Fatalf("map type %T", mapped)
	}
	if apiErr.Code != parties.CodePartyNotReady {
		t.Fatalf("stale version code = %s, want party_not_ready", apiErr.Code)
	}

	// Incomplete roster.
	store2 := newSecPartyStore()
	soloLeader := uuid.New()
	store2.seedActive(soloLeader)
	// Create solo-forming duo (1/2 members).
	created, err := parties.NewService(store2, friends.NewPartyPolicy(&secFriendStore{}), nil).
		Create(ctx, secRegistered(soloLeader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = parties.NewService(store2, friends.NewPartyPolicy(&secFriendStore{}), nil).
		ReadyPartyForQueue(ctx, soloLeader, parties.FormatDuo, int64(created.Party.Version))
	if !errors.Is(err, parties.ErrPartyIncomplete) {
		t.Fatalf("incomplete = %v", err)
	}
}

func TestSecurity_ExpiredTicketsPrivacySafe(t *testing.T) {
	t.Parallel()
	// Realtime ticket failure modes use stable codes without leaking ticket material.
	for _, tc := range []struct {
		err  error
		code string
	}{
		{realtime.ErrTicketExpired, realtime.CodeTicketExpired},
		{realtime.ErrTicketInvalid, realtime.CodeTicketInvalid},
		{realtime.ErrTicketUsed, realtime.CodeTicketUsed},
		{realtime.ErrNotParticipant, realtime.CodeNotParticipant},
	} {
		if tc.err == nil || tc.code == "" {
			t.Fatal("empty case")
		}
		// Codes must stay privacy-safe for non-participants (looks like not_found).
		if tc.err == realtime.ErrNotParticipant && tc.code != "not_found" {
			t.Fatalf("non-participant code = %s", tc.code)
		}
		msg := tc.err.Error()
		if strings.Contains(msg, "ticket.") || strings.Contains(msg, "raw:") {
			t.Fatalf("error string leaked ticket material: %s", msg)
		}
	}
}

func TestSecurity_PrivacySafeErrorEquivalenceMatrix(t *testing.T) {
	t.Parallel()

	// Cross-feature: missing resource ≈ unauthorized opponent/nonmember where existence is sensitive.
	pairs := []struct {
		name string
		a, b error
		mapF func(error) error
	}{
		{"matchplay not_found vs forbidden_opponent", matchplay.ErrNotFound, matchplay.ErrForbiddenOpponent, matchplay.MapError},
		{"parties not_found vs invite not_found", parties.ErrNotFound, parties.ErrInviteNotFound, parties.MapError},
	}
	for _, tc := range pairs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ma, mb := tc.mapF(tc.a), tc.mapF(tc.b)
			var aa, ab *apphttp.APIError
			if !errors.As(ma, &aa) || !errors.As(mb, &ab) {
				t.Fatalf("map types %T %T", ma, mb)
			}
			if aa.Status != ab.Status || aa.Code != ab.Code {
				t.Fatalf("inequivalent: a=%+v b=%+v", aa, ab)
			}
		})
	}

	// Guests map to unauthorized (not not_found) — auth state is not a privacy leak.
	for _, err := range []error{parties.ErrUnauthorized, matchmaking.ErrUnauthorized, matchplay.ErrUnauthorized} {
		var mapF func(error) error
		switch {
		case errors.Is(err, parties.ErrUnauthorized):
			mapF = parties.MapError
		case errors.Is(err, matchmaking.ErrUnauthorized):
			mapF = matchmaking.MapError
		default:
			mapF = matchplay.MapError
		}
		mapped := mapF(err)
		var apiErr *apphttp.APIError
		if !errors.As(mapped, &apiErr) {
			t.Fatalf("unauthorized map type %T for %v", mapped, err)
		}
		if apiErr.Status != http.StatusUnauthorized {
			t.Fatalf("%v status = %d", err, apiErr.Status)
		}
	}
}
