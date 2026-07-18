package parties_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
	"github.com/raven/geoguess/backend/internal/parties"
	"github.com/raven/geoguess/backend/internal/session"
)

// --- fakes ---

type memStore struct {
	mu          sync.Mutex
	active      map[uuid.UUID]bool
	parties     map[uuid.UUID]*parties.Party
	members     map[uuid.UUID][]parties.PartyMember // partyID -> members
	invites     map[uuid.UUID]*parties.PartyInvite
	profiles    map[uuid.UUID]parties.PublicProfile
	activeMatch map[uuid.UUID]bool
	createErr   error
	acceptErr   error
	inviteErr   error
}

func newMemStore() *memStore {
	return &memStore{
		active:      map[uuid.UUID]bool{},
		parties:     map[uuid.UUID]*parties.Party{},
		members:     map[uuid.UUID][]parties.PartyMember{},
		invites:     map[uuid.UUID]*parties.PartyInvite{},
		profiles:    map[uuid.UUID]parties.PublicProfile{},
		activeMatch: map[uuid.UUID]bool{},
	}
}

func (m *memStore) seedUser(id uuid.UUID, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active[id] = true
	m.profiles[id] = parties.PublicProfile{UserID: id, DisplayName: name}
}

func (m *memStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.active[userID] {
		return nil, nil
	}
	id := userID
	return &id, nil
}

func (m *memStore) HasActiveMatchAssignment(_ context.Context, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeMatch[userID], nil
}

func (m *memStore) CreateParty(_ context.Context, leaderID uuid.UUID, format string, now time.Time) (*parties.PartySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return nil, m.createErr
	}
	cap, ok := parties.CapacityForFormat(format)
	if !ok {
		return nil, parties.ErrUnsupportedFormat
	}
	if m.activeMatch[leaderID] {
		return nil, parties.ErrActiveQueueOrMatch
	}
	for _, members := range m.members {
		for _, mem := range members {
			if mem.UserID == leaderID && mem.Status == parties.MemberStatusActive {
				return nil, parties.ErrActivePartyConflict
			}
		}
	}
	p := &parties.Party{
		ID:           uuid.New(),
		Format:       format,
		Capacity:     int16(cap),
		LeaderUserID: leaderID,
		Status:       parties.StatusForming,
		Version:      0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.parties[p.ID] = p
	m.members[p.ID] = []parties.PartyMember{{
		PartyID: p.ID, UserID: leaderID, Status: parties.MemberStatusActive, JoinedAt: now,
	}}
	return m.snapshotLocked(p.ID)
}

func (m *memStore) GetActivePartyForUser(_ context.Context, userID uuid.UUID) (*parties.PartySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for pid, members := range m.members {
		for _, mem := range members {
			if mem.UserID == userID && mem.Status == parties.MemberStatusActive {
				return m.snapshotLocked(pid)
			}
		}
	}
	return nil, nil
}

func (m *memStore) LoadSnapshot(_ context.Context, partyID uuid.UUID) (*parties.PartySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked(partyID)
}

func (m *memStore) snapshotLocked(partyID uuid.UUID) (*parties.PartySnapshot, error) {
	p, ok := m.parties[partyID]
	if !ok {
		return nil, parties.ErrNotFound
	}
	cp := *p
	snap := &parties.PartySnapshot{Party: cp}
	for _, mem := range m.members[partyID] {
		if mem.Status != parties.MemberStatusActive {
			continue
		}
		prof := m.profiles[mem.UserID]
		if prof.UserID == uuid.Nil {
			prof = parties.PublicProfile{UserID: mem.UserID, DisplayName: "User"}
		}
		snap.Members = append(snap.Members, parties.MemberSnapshot{Member: mem, Profile: prof})
	}
	return snap, nil
}

func (m *memStore) CreateInvite(_ context.Context, partyID, leaderID, inviteeID uuid.UUID, expiresAt, now time.Time) (*parties.InviteSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inviteErr != nil {
		return nil, m.inviteErr
	}
	p, ok := m.parties[partyID]
	if !ok {
		return nil, parties.ErrNotFound
	}
	if p.Status != parties.StatusForming {
		return nil, parties.ErrPartyLocked
	}
	if p.LeaderUserID != leaderID {
		return nil, parties.ErrLeaderRequired
	}
	active := 0
	for _, mem := range m.members[partyID] {
		if mem.Status == parties.MemberStatusActive {
			active++
			if mem.UserID == inviteeID {
				return nil, parties.ErrTargetBusy
			}
		}
	}
	if active >= int(p.Capacity) {
		return nil, parties.ErrPartyFull
	}
	for _, members := range m.members {
		for _, mem := range members {
			if mem.UserID == inviteeID && mem.Status == parties.MemberStatusActive {
				return nil, parties.ErrTargetBusy
			}
		}
	}
	if m.activeMatch[inviteeID] {
		return nil, parties.ErrTargetBusy
	}
	for _, inv := range m.invites {
		if inv.PartyID == partyID && inv.InviteeUserID == inviteeID && inv.Status == parties.InviteStatusPending {
			return nil, parties.ErrAlreadyInvited
		}
	}
	inv := &parties.PartyInvite{
		ID: uuid.New(), PartyID: partyID, InviterUserID: leaderID, InviteeUserID: inviteeID,
		Status: parties.InviteStatusPending, ExpiresAt: expiresAt, CreatedAt: now,
	}
	m.invites[inv.ID] = inv
	prof := m.profiles[leaderID]
	return &parties.InviteSnapshot{Invite: *inv, Inviter: prof, Format: p.Format, Capacity: p.Capacity}, nil
}

func (m *memStore) ListPendingInvitesForUser(_ context.Context, inviteeID uuid.UUID, now time.Time) ([]parties.InviteSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []parties.InviteSnapshot
	for _, inv := range m.invites {
		if inv.InviteeUserID != inviteeID || inv.Status != parties.InviteStatusPending {
			continue
		}
		if !inv.ExpiresAt.After(now) {
			continue
		}
		p := m.parties[inv.PartyID]
		if p == nil || p.Status != parties.StatusForming {
			continue
		}
		out = append(out, parties.InviteSnapshot{
			Invite: *inv, Inviter: m.profiles[inv.InviterUserID], Format: p.Format, Capacity: p.Capacity,
		})
	}
	return out, nil
}

func (m *memStore) AcceptInvite(_ context.Context, inviteID, inviteeID uuid.UUID, now time.Time) (*parties.PartySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acceptErr != nil {
		return nil, m.acceptErr
	}
	inv, ok := m.invites[inviteID]
	if !ok || inv.InviteeUserID != inviteeID {
		return nil, parties.ErrInviteNotFound
	}
	if inv.Status == parties.InviteStatusAccepted {
		return m.snapshotLocked(inv.PartyID)
	}
	if inv.Status != parties.InviteStatusPending || !inv.ExpiresAt.After(now) {
		return nil, parties.ErrInviteNotFound
	}
	p := m.parties[inv.PartyID]
	if p == nil {
		return nil, parties.ErrNotFound
	}
	if p.Status != parties.StatusForming {
		return nil, parties.ErrPartyLocked
	}
	active := 0
	for _, mem := range m.members[p.ID] {
		if mem.Status == parties.MemberStatusActive {
			active++
		}
	}
	if active >= int(p.Capacity) {
		return nil, parties.ErrPartyFull
	}
	for _, members := range m.members {
		for _, mem := range members {
			if mem.UserID == inviteeID && mem.Status == parties.MemberStatusActive {
				return nil, parties.ErrTargetBusy
			}
		}
	}
	// Clear readiness + add member.
	for i := range m.members[p.ID] {
		if m.members[p.ID][i].Status == parties.MemberStatusActive {
			m.members[p.ID][i].Ready = false
		}
	}
	m.members[p.ID] = append(m.members[p.ID], parties.PartyMember{
		PartyID: p.ID, UserID: inviteeID, Status: parties.MemberStatusActive, JoinedAt: now,
	})
	inv.Status = parties.InviteStatusAccepted
	t := now
	inv.RespondedAt = &t
	p.Version++
	p.UpdatedAt = now
	return m.snapshotLocked(p.ID)
}

func (m *memStore) DeclineInvite(_ context.Context, inviteID, inviteeID uuid.UUID, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invites[inviteID]
	if !ok || inv.InviteeUserID != inviteeID {
		return parties.ErrInviteNotFound
	}
	if inv.Status == parties.InviteStatusDeclined {
		return nil
	}
	if inv.Status != parties.InviteStatusPending {
		return parties.ErrInviteNotFound
	}
	inv.Status = parties.InviteStatusDeclined
	t := now
	inv.RespondedAt = &t
	return nil
}

func (m *memStore) SetReadiness(_ context.Context, partyID, userID uuid.UUID, ready bool, now time.Time) (*parties.PartySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parties[partyID]
	if !ok {
		return nil, parties.ErrNotFound
	}
	if p.Status != parties.StatusForming {
		return nil, parties.ErrPartyLocked
	}
	found := false
	for i := range m.members[partyID] {
		if m.members[partyID][i].UserID == userID && m.members[partyID][i].Status == parties.MemberStatusActive {
			if m.members[partyID][i].Ready != ready {
				m.members[partyID][i].Ready = ready
				p.Version++
				p.UpdatedAt = now
			}
			found = true
			break
		}
	}
	if !found {
		return nil, parties.ErrNotFound
	}
	return m.snapshotLocked(partyID)
}

func (m *memStore) LeaveParty(_ context.Context, partyID, userID uuid.UUID, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parties[partyID]
	if !ok {
		return parties.ErrNotFound
	}
	if p.Status == parties.StatusClosed {
		return parties.ErrNotFound
	}
	if p.Status != parties.StatusForming {
		return parties.ErrPartyLocked
	}
	found := false
	for i := range m.members[partyID] {
		if m.members[partyID][i].UserID == userID && m.members[partyID][i].Status == parties.MemberStatusActive {
			m.members[partyID][i].Status = parties.MemberStatusLeft
			t := now
			m.members[partyID][i].LeftAt = &t
			m.members[partyID][i].Ready = false
			found = true
		}
	}
	if !found {
		return nil
	}
	for i := range m.members[partyID] {
		if m.members[partyID][i].Status == parties.MemberStatusActive {
			m.members[partyID][i].Ready = false
		}
	}
	var remaining []parties.PartyMember
	for _, mem := range m.members[partyID] {
		if mem.Status == parties.MemberStatusActive {
			remaining = append(remaining, mem)
		}
	}
	if len(remaining) == 0 {
		p.Status = parties.StatusClosed
		t := now
		p.ClosedAt = &t
		p.Version++
		return nil
	}
	if p.LeaderUserID == userID {
		// earliest joined, then lowest user id
		best := remaining[0]
		for _, mem := range remaining[1:] {
			if mem.JoinedAt.Before(best.JoinedAt) ||
				(mem.JoinedAt.Equal(best.JoinedAt) && bytesLess(mem.UserID, best.UserID)) {
				best = mem
			}
		}
		p.LeaderUserID = best.UserID
	}
	p.Version++
	p.UpdatedAt = now
	return nil
}

func (m *memStore) KickMember(_ context.Context, partyID, leaderID, targetID uuid.UUID, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parties[partyID]
	if !ok {
		return parties.ErrNotFound
	}
	if p.Status != parties.StatusForming {
		return parties.ErrPartyLocked
	}
	if p.LeaderUserID != leaderID {
		return parties.ErrLeaderRequired
	}
	for i := range m.members[partyID] {
		if m.members[partyID][i].UserID == targetID && m.members[partyID][i].Status == parties.MemberStatusActive {
			m.members[partyID][i].Status = parties.MemberStatusKicked
			t := now
			m.members[partyID][i].LeftAt = &t
			m.members[partyID][i].Ready = false
		} else if m.members[partyID][i].Status == parties.MemberStatusActive {
			m.members[partyID][i].Ready = false
		}
	}
	p.Version++
	p.UpdatedAt = now
	return nil
}

func (m *memStore) DisbandParty(_ context.Context, partyID, leaderID uuid.UUID, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.parties[partyID]
	if !ok {
		return parties.ErrNotFound
	}
	if p.Status == parties.StatusClosed {
		if p.LeaderUserID != leaderID {
			return parties.ErrNotFound
		}
		return nil
	}
	if p.Status != parties.StatusForming {
		return parties.ErrPartyLocked
	}
	if p.LeaderUserID != leaderID {
		return parties.ErrLeaderRequired
	}
	for i := range m.members[partyID] {
		if m.members[partyID][i].Status == parties.MemberStatusActive {
			m.members[partyID][i].Status = parties.MemberStatusLeft
			t := now
			m.members[partyID][i].LeftAt = &t
			m.members[partyID][i].Ready = false
		}
	}
	for _, inv := range m.invites {
		if inv.PartyID == partyID && inv.Status == parties.InviteStatusPending {
			inv.Status = parties.InviteStatusRevoked
			t := now
			inv.RespondedAt = &t
		}
	}
	p.Status = parties.StatusClosed
	t := now
	p.ClosedAt = &t
	p.Version++
	return nil
}

func (m *memStore) GetInvite(_ context.Context, inviteID uuid.UUID) (*parties.PartyInvite, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invites[inviteID]
	if !ok {
		return nil, nil
	}
	cp := *inv
	return &cp, nil
}

func (m *memStore) RestoreAfterTerminalMatch(_ context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[uuid.UUID]struct{}{}
	for _, id := range partyIDs {
		if id == uuid.Nil {
			continue
		}
		seen[id] = struct{}{}
	}
	if matchID != uuid.Nil {
		for id, p := range m.parties {
			if p.ActiveMatchID != nil && *p.ActiveMatchID == matchID {
				seen[id] = struct{}{}
			}
		}
	}
	for id := range seen {
		p, ok := m.parties[id]
		if !ok || p.Status != parties.StatusInMatch {
			continue
		}
		p.Status = parties.StatusForming
		p.ActiveMatchID = nil
		p.Version++
		for i := range m.members[id] {
			if m.members[id][i].Status == parties.MemberStatusActive {
				m.members[id][i].Ready = false
			}
		}
	}
	return nil
}

func (m *memStore) setStatus(partyID uuid.UUID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.parties[partyID]; ok {
		p.Status = status
	}
}

func bytesLess(a, b uuid.UUID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

type memFriends struct {
	// allowed pairs: inviter|invitee
	allowed map[string]bool
	blocks  map[string]bool
}

func (f *memFriends) CanInvite(_ context.Context, inviter, invitee uuid.UUID) error {
	if f.blocks[inviter.String()+"|"+invitee.String()] || f.blocks[invitee.String()+"|"+inviter.String()] {
		return friends.ErrFriendUnavailable
	}
	if f.allowed[inviter.String()+"|"+invitee.String()] || f.allowed[invitee.String()+"|"+inviter.String()] {
		return nil
	}
	return friends.ErrFriendUnavailable
}

func (f *memFriends) allow(a, b uuid.UUID) {
	if f.allowed == nil {
		f.allowed = map[string]bool{}
	}
	f.allowed[a.String()+"|"+b.String()] = true
	f.allowed[b.String()+"|"+a.String()] = true
}

func (f *memFriends) block(a, b uuid.UUID) {
	if f.blocks == nil {
		f.blocks = map[string]bool{}
	}
	f.blocks[a.String()+"|"+b.String()] = true
}

type memIdem struct {
	mu    sync.Mutex
	store map[string]struct {
		hash string
		body []byte
		done bool
	}
}

func (m *memIdem) Begin(_ context.Context, scope, callerID, idemKey, bodyHash string, _ time.Duration) (parties.IdempotencyBeginResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store == nil {
		m.store = map[string]struct {
			hash string
			body []byte
			done bool
		}{}
	}
	key := scope + "|" + callerID + "|" + idemKey
	if existing, ok := m.store[key]; ok {
		if existing.hash != bodyHash {
			return parties.IdempotencyBeginResult{Conflict: true}, nil
		}
		if existing.done {
			return parties.IdempotencyBeginResult{Hit: true, Response: existing.body}, nil
		}
		return parties.IdempotencyBeginResult{Conflict: true, InFlight: true}, nil
	}
	m.store[key] = struct {
		hash string
		body []byte
		done bool
	}{hash: bodyHash}
	return parties.IdempotencyBeginResult{}, nil
}

func (m *memIdem) Complete(_ context.Context, scope, callerID, idemKey, bodyHash string, response []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scope + "|" + callerID + "|" + idemKey
	m.store[key] = struct {
		hash string
		body []byte
		done bool
	}{hash: bodyHash, body: response, done: true}
	return nil
}

func (m *memIdem) Release(_ context.Context, scope, callerID, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scope + "|" + callerID + "|" + idemKey
	if existing, ok := m.store[key]; ok && !existing.done {
		delete(m.store, key)
	}
	return nil
}

func registered(id uuid.UUID) session.Context {
	s := id.String()
	return session.Context{Kind: session.KindUser, UserID: &s, Role: "user"}
}

type partyEventCapture struct {
	types []string
}

func (c *partyEventCapture) Publish(_ context.Context, _ uuid.UUID, eventType string, _ int64, _ any) error {
	c.types = append(c.types, eventType)
	return nil
}

func TestPartyMutationsPublishPostCommitEvents(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "Leader")
	capture := &partyEventCapture{}
	svc := parties.NewService(store, &memFriends{}, nil).WithEvents(capture)
	created, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "events-create")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.SetReadiness(context.Background(), registered(leader), created.Party.ID.String(), parties.ReadinessRequest{Ready: true}); err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if len(capture.types) != 2 || capture.types[0] != parties.EventPartyRosterUpdated || capture.types[1] != parties.EventPartyReadinessChanged {
		t.Fatalf("events = %v", capture.types)
	}
}

// --- tests ---

func TestCreateRequiresRegistered(t *testing.T) {
	svc := parties.NewService(newMemStore(), &memFriends{}, nil)
	_, err := svc.Create(context.Background(), session.Context{Kind: session.KindGuest}, parties.CreatePartyRequest{Format: "duo"}, "k1")
	if !errors.Is(err, parties.ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateUnsupportedFormat(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	svc := parties.NewService(store, &memFriends{}, nil)
	_, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "solo"}, "k1")
	if !errors.Is(err, parties.ErrUnsupportedFormat) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateAndCurrent(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "Leader")
	svc := parties.NewService(store, &memFriends{}, nil)

	resp, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "create-1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.Party.Format != "duo" || resp.Party.Capacity != 2 || resp.Party.LeaderUserID != leader {
		t.Fatalf("party = %+v", resp.Party)
	}
	if len(resp.Party.Members) != 1 || !resp.Party.Members[0].IsLeader {
		t.Fatalf("members = %+v", resp.Party.Members)
	}

	cur, err := svc.Current(context.Background(), registered(leader))
	if err != nil || cur.Party == nil || cur.Party.ID != resp.Party.ID {
		t.Fatalf("current = %+v err=%v", cur, err)
	}

	// Active party conflict
	if _, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "squad"}, "create-2"); !errors.Is(err, parties.ErrActivePartyConflict) {
		t.Fatalf("conflict err = %v", err)
	}
}

func TestInviteFriendOnlyAndBlocks(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	friend := uuid.New()
	stranger := uuid.New()
	blocked := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(friend, "F")
	store.seedUser(stranger, "S")
	store.seedUser(blocked, "B")

	policy := &memFriends{}
	policy.allow(leader, friend)
	policy.block(leader, blocked)

	svc := parties.NewService(store, policy, nil)
	created, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := created.Party.ID.String()

	// Friend ok
	inv, err := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: friend.String()}, "i1")
	if err != nil {
		t.Fatalf("invite friend: %v", err)
	}
	if inv.Invite.PartyID != created.Party.ID {
		t.Fatalf("invite party mismatch")
	}

	// Stranger rejected privacy-safe
	if _, err := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: stranger.String()}, "i2"); !errors.Is(err, parties.ErrFriendUnavailable) {
		t.Fatalf("stranger err = %v", err)
	}

	// Either-direction block
	if _, err := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: blocked.String()}, "i3"); !errors.Is(err, parties.ErrFriendUnavailable) {
		t.Fatalf("block err = %v", err)
	}

	// Reverse block direction
	policy2 := &memFriends{}
	policy2.block(blocked, leader) // blocked blocked leader
	svc2 := parties.NewService(store, policy2, nil)
	// need another party - leader already has one; use new leader
	leader2 := uuid.New()
	store.seedUser(leader2, "L2")
	// leader2 not friends with blocked
	created2, _ := svc2.Create(context.Background(), registered(leader2), parties.CreatePartyRequest{Format: "duo"}, "c2")
	if _, err := svc2.Invite(context.Background(), registered(leader2), created2.Party.ID.String(), parties.InviteRequest{UserID: blocked.String()}, "i4"); !errors.Is(err, parties.ErrFriendUnavailable) {
		t.Fatalf("reverse block err = %v", err)
	}
}

func TestInviteCapacityAndNonLeader(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	m1 := uuid.New()
	m2 := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(m1, "M1")
	store.seedUser(m2, "M2")
	policy := &memFriends{}
	policy.allow(leader, m1)
	policy.allow(leader, m2)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	pid := created.Party.ID.String()

	inv, err := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: m1.String()}, "i1")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if _, err := svc.AcceptInvite(context.Background(), registered(m1), inv.Invite.ID.String(), "a1"); err != nil {
		t.Fatalf("accept: %v", err)
	}

	// Party full (duo capacity 2)
	if _, err := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: m2.String()}, "i2"); !errors.Is(err, parties.ErrPartyFull) {
		t.Fatalf("full err = %v", err)
	}

	// Non-leader cannot invite on a squad with room
	store2 := newMemStore()
	l := uuid.New()
	f := uuid.New()
	other := uuid.New()
	store2.seedUser(l, "L")
	store2.seedUser(f, "F")
	store2.seedUser(other, "O")
	p := &memFriends{}
	p.allow(l, f)
	p.allow(f, other)
	svc3 := parties.NewService(store2, p, nil)
	c, _ := svc3.Create(context.Background(), registered(l), parties.CreatePartyRequest{Format: "squad"}, "c")
	inv2, _ := svc3.Invite(context.Background(), registered(l), c.Party.ID.String(), parties.InviteRequest{UserID: f.String()}, "i")
	if _, err := svc3.AcceptInvite(context.Background(), registered(f), inv2.Invite.ID.String(), "a"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err := svc3.Invite(context.Background(), registered(f), c.Party.ID.String(), parties.InviteRequest{UserID: other.String()}, "ix"); !errors.Is(err, parties.ErrLeaderRequired) {
		t.Fatalf("non-leader err = %v", err)
	}
}

func TestReadinessResetOnRosterChange(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	friend := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(friend, "F")
	policy := &memFriends{}
	policy.allow(leader, friend)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	pid := created.Party.ID.String()

	// Leader ready
	ready, err := svc.SetReadiness(context.Background(), registered(leader), pid, parties.ReadinessRequest{Ready: true})
	if err != nil || !ready.Party.Members[0].Ready {
		t.Fatalf("ready: %+v err=%v", ready, err)
	}

	// Accept friend — readiness should clear
	inv, _ := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: friend.String()}, "i1")
	joined, err := svc.AcceptInvite(context.Background(), registered(friend), inv.Invite.ID.String(), "a1")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	for _, mem := range joined.Party.Members {
		if mem.Ready {
			t.Fatalf("expected readiness cleared, got %+v", joined.Party.Members)
		}
	}
}

func TestLeaderTransferOnLeave(t *testing.T) {
	store := newMemStore()
	// Fixed UUIDs for deterministic leadership (earliest joined wins; tie -> lowest id)
	leader := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	early := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	late := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	store.seedUser(leader, "L")
	store.seedUser(early, "E")
	store.seedUser(late, "T")
	policy := &memFriends{}
	policy.allow(leader, early)
	policy.allow(leader, late)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "squad"}, "c1")
	pid := created.Party.ID.String()

	inv1, _ := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: early.String()}, "i1")
	if _, err := svc.AcceptInvite(context.Background(), registered(early), inv1.Invite.ID.String(), "a1"); err != nil {
		t.Fatalf("accept early: %v", err)
	}
	// slight delay not needed — mem store uses clock times; join order by accept order
	time.Sleep(2 * time.Millisecond)
	inv2, _ := svc.Invite(context.Background(), registered(leader), pid, parties.InviteRequest{UserID: late.String()}, "i2")
	if _, err := svc.AcceptInvite(context.Background(), registered(late), inv2.Invite.ID.String(), "a2"); err != nil {
		t.Fatalf("accept late: %v", err)
	}

	if err := svc.Leave(context.Background(), registered(leader), pid); err != nil {
		t.Fatalf("leave: %v", err)
	}
	cur, err := svc.Current(context.Background(), registered(early))
	if err != nil || cur.Party == nil {
		t.Fatalf("current: %+v err=%v", cur, err)
	}
	if cur.Party.LeaderUserID != early {
		t.Fatalf("leader = %s, want early %s", cur.Party.LeaderUserID, early)
	}
}

func TestQueueLockingBlocksMutations(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	friend := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(friend, "F")
	policy := &memFriends{}
	policy.allow(leader, friend)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	pid := created.Party.ID
	store.setStatus(pid, parties.StatusQueued)

	if _, err := svc.Invite(context.Background(), registered(leader), pid.String(), parties.InviteRequest{UserID: friend.String()}, "i1"); !errors.Is(err, parties.ErrPartyLocked) {
		t.Fatalf("invite locked: %v", err)
	}
	if _, err := svc.SetReadiness(context.Background(), registered(leader), pid.String(), parties.ReadinessRequest{Ready: true}); !errors.Is(err, parties.ErrPartyLocked) {
		t.Fatalf("ready locked: %v", err)
	}
	if err := svc.Leave(context.Background(), registered(leader), pid.String()); !errors.Is(err, parties.ErrPartyLocked) {
		t.Fatalf("leave locked: %v", err)
	}
	if err := svc.Disband(context.Background(), registered(leader), pid.String()); !errors.Is(err, parties.ErrPartyLocked) {
		t.Fatalf("disband locked: %v", err)
	}
	if err := svc.Kick(context.Background(), registered(leader), pid.String(), friend.String()); !errors.Is(err, parties.ErrPartyLocked) {
		t.Fatalf("kick locked: %v", err)
	}
}

func TestIdempotentDeclineAndDisband(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	friend := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(friend, "F")
	policy := &memFriends{}
	policy.allow(leader, friend)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	inv, _ := svc.Invite(context.Background(), registered(leader), created.Party.ID.String(), parties.InviteRequest{UserID: friend.String()}, "i1")

	if err := svc.DeclineInvite(context.Background(), registered(friend), inv.Invite.ID.String()); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if err := svc.DeclineInvite(context.Background(), registered(friend), inv.Invite.ID.String()); err != nil {
		t.Fatalf("decline retry: %v", err)
	}

	if err := svc.Disband(context.Background(), registered(leader), created.Party.ID.String()); err != nil {
		t.Fatalf("disband: %v", err)
	}
	if err := svc.Disband(context.Background(), registered(leader), created.Party.ID.String()); err != nil {
		t.Fatalf("disband retry: %v", err)
	}
}

func TestCommandIdempotencyReplayAndConflict(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	store.seedUser(leader, "L")
	idem := &memIdem{}
	svc := parties.NewService(store, &memFriends{}, nil).WithIdempotency(idem)

	r1, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "same-key")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	r2, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "same-key")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if r1.Party.ID != r2.Party.ID {
		t.Fatalf("replay should return same party")
	}

	// Conflict body
	if _, err := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "squad"}, "same-key"); !errors.Is(err, parties.ErrIdempotencyConflict) {
		t.Fatalf("conflict err = %v", err)
	}
}

func TestReadyPartyForQueue(t *testing.T) {
	store := newMemStore()
	leader := uuid.New()
	friend := uuid.New()
	store.seedUser(leader, "L")
	store.seedUser(friend, "F")
	policy := &memFriends{}
	policy.allow(leader, friend)
	svc := parties.NewService(store, policy, nil)

	created, _ := svc.Create(context.Background(), registered(leader), parties.CreatePartyRequest{Format: "duo"}, "c1")
	inv, _ := svc.Invite(context.Background(), registered(leader), created.Party.ID.String(), parties.InviteRequest{UserID: friend.String()}, "i1")
	joined, _ := svc.AcceptInvite(context.Background(), registered(friend), inv.Invite.ID.String(), "a1")

	// Not ready yet
	if _, err := svc.ReadyPartyForQueue(context.Background(), leader, "duo", int64(joined.Party.Version)); !errors.Is(err, parties.ErrPartyNotReady) {
		t.Fatalf("not ready: %v", err)
	}

	_, _ = svc.SetReadiness(context.Background(), registered(leader), created.Party.ID.String(), parties.ReadinessRequest{Ready: true})
	readyResp, _ := svc.SetReadiness(context.Background(), registered(friend), created.Party.ID.String(), parties.ReadinessRequest{Ready: true})

	snap, err := svc.ReadyPartyForQueue(context.Background(), leader, "duo", int64(readyResp.Party.Version))
	if err != nil {
		t.Fatalf("ready queue: %v", err)
	}
	if snap == nil || !snap.AllReady || len(snap.MemberIDs) != 2 {
		t.Fatalf("snap = %+v", snap)
	}
}

func TestToAPIErrorPrivacySafeFriendUnavailable(t *testing.T) {
	apiErr := parties.ToAPIError(parties.ErrFriendUnavailable)
	if apiErr == nil {
		t.Fatal("expected api error")
	}
	// Ensure code is friend_unavailable not block details
	msg := apiErr.Error()
	if !contains(msg, "friend_unavailable") {
		t.Fatalf("api err = %s", msg)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
