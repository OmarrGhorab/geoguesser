package matchmaking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/session"
)

type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

type memoryStore struct {
	users                     map[uuid.UUID]*matchmaking.ActiveUser
	assignments               map[uuid.UUID]*matchmaking.ActiveAssignment
	conflicts                 map[uuid.UUID]bool
	matchesByKey              map[string]*matchmaking.Match
	postgresDown              bool
	assignmentCalls           int
	assignmentFailAt          int
	assignmentFailWhenPresent bool
	matchLookupErr            error
	formationFail             error
	formationCalls            int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		users:        map[uuid.UUID]*matchmaking.ActiveUser{},
		assignments:  map[uuid.UUID]*matchmaking.ActiveAssignment{},
		conflicts:    map[uuid.UUID]bool{},
		matchesByKey: map[string]*matchmaking.Match{},
	}
}

func (s *memoryStore) FindActiveUser(ctx context.Context, userID uuid.UUID) (*matchmaking.ActiveUser, error) {
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.users[userID], nil
}

func (s *memoryStore) FindActiveAssignment(ctx context.Context, userID uuid.UUID) (*matchmaking.ActiveAssignment, error) {
	s.assignmentCalls++
	if s.assignmentFailAt > 0 && s.assignmentCalls == s.assignmentFailAt {
		return nil, matchmaking.ErrUnavailable
	}
	if s.assignmentFailWhenPresent && s.assignments[userID] != nil {
		return nil, matchmaking.ErrUnavailable
	}
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.assignments[userID], nil
}

func (s *memoryStore) HasConflictingActiveGame(ctx context.Context, userID uuid.UUID) (bool, error) {
	if s.postgresDown {
		return false, matchmaking.ErrUnavailable
	}
	return s.conflicts[userID], nil
}

func (s *memoryStore) FindMatchByFormationKey(ctx context.Context, formationKey string) (*matchmaking.Match, error) {
	if s.matchLookupErr != nil {
		return nil, s.matchLookupErr
	}
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	return s.matchesByKey[formationKey], nil
}

func (s *memoryStore) CreateFormationBundle(ctx context.Context, input matchmaking.FormationInput) (*matchmaking.FormationResult, error) {
	timer := input.TimerSeconds
	return s.CreateTeamFormationBundle(ctx, matchmaking.TeamFormationInput{
		FormationKey: input.FormationKey,
		Mode:         input.Mode,
		Playlist:     matchmaking.PlaylistRanked,
		Format:       matchmaking.FormatSolo,
		TeamSize:     1,
		MapID:        input.MapID,
		RoundCount:   input.RoundCount,
		TimerSeconds: &timer,
		StartDelay:   input.StartDelay,
		TeamOne:      []uuid.UUID{input.UserIDs[0]},
		TeamTwo:      []uuid.UUID{input.UserIDs[1]},
		SeasonID:     input.SeasonID,
		LocationIDs:  input.LocationIDs,
		MatchedAt:    input.MatchedAt,
	})
}

func (s *memoryStore) CreateTeamFormationBundle(ctx context.Context, input matchmaking.TeamFormationInput) (*matchmaking.FormationResult, error) {
	s.formationCalls++
	if s.postgresDown {
		return nil, matchmaking.ErrUnavailable
	}
	if s.formationFail != nil {
		return nil, s.formationFail
	}
	if existing := s.matchesByKey[input.FormationKey]; existing != nil {
		return &matchmaking.FormationResult{Match: *existing}, nil
	}
	started := input.MatchedAt
	playlist, format, teamSize := input.Playlist, input.Format, input.TeamSize
	if playlist == "" || format == "" {
		if parts, err := matchmaking.ParseMode(input.Mode); err == nil {
			playlist, format, teamSize = parts.Playlist, parts.Format, parts.TeamSize
		}
	}
	match := matchmaking.Match{
		ID:             uuid.New(),
		FormationKey:   input.FormationKey,
		GameID:         uuid.New(),
		Mode:           input.Mode,
		Playlist:       playlist,
		Format:         format,
		TeamSize:       teamSize,
		Status:         matchmaking.MatchStatusActive,
		MatchedAt:      input.MatchedAt,
		StartedAt:      &started,
		LastActivityAt: input.MatchedAt,
	}
	s.matchesByKey[input.FormationKey] = &match
	for _, uid := range append(append([]uuid.UUID{}, input.TeamOne...), input.TeamTwo...) {
		s.assignments[uid] = &matchmaking.ActiveAssignment{
			MatchID:   match.ID,
			GameID:    match.GameID,
			Mode:      match.Mode,
			Status:    match.Status,
			MatchedAt: match.MatchedAt,
			UserID:    uid,
		}
	}
	return &matchmaking.FormationResult{Match: match}, nil
}

type memoryQueue struct {
	entries         map[uuid.UUID]*matchmaking.QueueEntry
	claims          map[string]*matchmaking.PairClaim
	redisDown       bool
	nextClaim       *matchmaking.PairClaim
	claimCalls      int
	finalizeCalls   int
	releaseCalls    int
	releaseErr      error
	lastRequeueA    *bool
	lastRequeueB    *bool
	expiredClaimIDs []string
}

func newMemoryQueue() *memoryQueue {
	return &memoryQueue{
		entries: map[uuid.UUID]*matchmaking.QueueEntry{},
		claims:  map[string]*matchmaking.PairClaim{},
	}
}

func (q *memoryQueue) Join(ctx context.Context, userID uuid.UUID, mode string, now time.Time, leaseTTL time.Duration) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	// Mirror Redis: searching (same mode) and claimed entries are immutable until leave/finalize/release.
	if existing, ok := q.entries[userID]; ok {
		if existing.State == matchmaking.QueueStateSearching && existing.Mode == mode {
			return existing, nil
		}
		if existing.State == matchmaking.QueueStateClaimed {
			return existing, nil
		}
	}
	entry := &matchmaking.QueueEntry{
		EntryID:          uuid.NewString(),
		UserID:           userID,
		Mode:             mode,
		State:            matchmaking.QueueStateSearching,
		EnqueuedAtMs:     now.UnixMilli(),
		LeaseExpiresAtMs: now.Add(leaseTTL).UnixMilli(),
	}
	q.entries[userID] = entry
	return entry, nil
}

func (q *memoryQueue) Leave(ctx context.Context, userID uuid.UUID) error {
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	delete(q.entries, userID)
	return nil
}

func (q *memoryQueue) GetEntry(ctx context.Context, userID uuid.UUID) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	return q.entries[userID], nil
}

func (q *memoryQueue) RenewLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*matchmaking.QueueEntry, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	entry := q.entries[userID]
	if entry == nil {
		return nil, nil
	}
	entry.LeaseExpiresAtMs = now.Add(leaseTTL).UnixMilli()
	return entry, nil
}

func (q *memoryQueue) ClaimPair(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int) (*matchmaking.PairClaim, error) {
	q.claimCalls++
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	claim := q.nextClaim
	q.nextClaim = nil
	return claim, nil
}

func (q *memoryQueue) GetClaim(ctx context.Context, claimID string) (*matchmaking.PairClaim, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	if claimID == "" {
		return nil, nil
	}
	if q.nextClaim != nil && q.nextClaim.ClaimID == claimID {
		return q.nextClaim, nil
	}
	if claim, ok := q.claims[claimID]; ok {
		return claim, nil
	}
	return nil, nil
}

func (q *memoryQueue) FinalizeClaim(ctx context.Context, claim *matchmaking.PairClaim) error {
	q.finalizeCalls++
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	if claim != nil {
		delete(q.claims, claim.ClaimID)
	}
	return nil
}

func (q *memoryQueue) ReleaseClaim(ctx context.Context, claim *matchmaking.PairClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	q.releaseCalls++
	q.lastRequeueA = &requeueA
	q.lastRequeueB = &requeueB
	if q.releaseErr != nil {
		return q.releaseErr
	}
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	return nil
}

func (q *memoryQueue) ListExpiredClaims(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if q.redisDown {
		return nil, matchmaking.ErrUnavailable
	}
	if limit <= 0 || limit > len(q.expiredClaimIDs) {
		return append([]string(nil), q.expiredClaimIDs...), nil
	}
	return append([]string(nil), q.expiredClaimIDs[:limit]...), nil
}

func (q *memoryQueue) DropClaimIndex(ctx context.Context, claimID string) error {
	if q.redisDown {
		return matchmaking.ErrUnavailable
	}
	return nil
}

type memoryLocations struct {
	ids   []uuid.UUID
	err   error
	calls int
}

func (l *memoryLocations) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]uuid.UUID, error) {
	l.calls++
	if l.err != nil {
		return nil, l.err
	}
	if len(l.ids) >= count {
		return l.ids[:count], nil
	}
	return l.ids, nil
}

func userSession(userID uuid.UUID) *session.Context {
	id := userID.String()
	return &session.Context{Kind: session.KindUser, UserID: &id, Role: "user"}
}

func TestJoinQueue_GuestDenied(t *testing.T) {
	svc := matchmaking.NewService(newMemoryStore(), newMemoryQueue(), testConfig(), nil, nil)
	guestID := "guest-1"
	_, err := svc.JoinQueue(context.Background(), &session.Context{Kind: session.KindGuest, GuestID: &guestID}, matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrUnauthorized {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestJoinQueue_DisabledAccountDenied(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "disabled"}
	svc := matchmaking.NewService(store, newMemoryQueue(), testConfig(), nil, nil)

	_, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrAccountIneligible {
		t.Fatalf("err = %v, want ErrAccountIneligible", err)
	}
}

func TestJoinQueue_UnsupportedMode(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	svc := matchmaking.NewService(store, newMemoryQueue(), testConfig(), nil, nil)

	_, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: "ranked_season"})
	if err != matchmaking.ErrUnsupportedMode {
		t.Fatalf("err = %v, want ErrUnsupportedMode", err)
	}
}

func TestJoinQueue_ActiveGameConflict(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	store.conflicts[userID] = true
	svc := matchmaking.NewService(store, newMemoryQueue(), testConfig(), nil, nil)

	_, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrActiveGameConflict {
		t.Fatalf("err = %v, want ErrActiveGameConflict", err)
	}
}

func TestJoinQueue_FirstAndDuplicatePreservePriority(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, newMemoryQueue(), testConfig(), nil, nil).
		WithClock(stubClock{now: now}).
		WithLocations(locs)

	first, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("first join: %v", err)
	}
	if first.Status != matchmaking.PublicStatusSearching {
		t.Fatalf("status = %q, want searching", first.Status)
	}
	if first.Queue == nil || first.Queue.SearchStartedAt.UnixMilli() != now.UnixMilli() {
		t.Fatalf("search_started_at = %v, want %v", first.Queue.SearchStartedAt, now)
	}

	// Advance clock; duplicate join must preserve original start.
	svc = svc.WithClock(stubClock{now: now.Add(10 * time.Second)})
	second, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("second join: %v", err)
	}
	if second.Queue == nil || second.Queue.SearchStartedAt.UnixMilli() != first.Queue.SearchStartedAt.UnixMilli() {
		t.Fatalf("priority not preserved: first=%v second=%v", first.Queue.SearchStartedAt, second.Queue.SearchStartedAt)
	}
}

func TestJoinQueue_DuplicateJoinWhileClaimedPreservesEntry(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	claimID := uuid.NewString()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	queue := newMemoryQueue()
	original := &matchmaking.QueueEntry{
		EntryID: "e-claimed-a", UserID: userA, Mode: matchmaking.ModeRankedStandard,
		State: matchmaking.QueueStateClaimed, ClaimID: claimID,
		EnqueuedAtMs: now.UnixMilli(), LeaseExpiresAtMs: now.Add(time.Minute).UnixMilli(),
	}
	queue.entries[userA] = original
	queue.claims[claimID] = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-claimed-a", EntryIDB: "e-b",
		EnqueuedAtMsA: now.UnixMilli(), EnqueuedAtMsB: now.UnixMilli(),
		// Still within claim TTL so recovery returns temporarily_unavailable (not requeue).
		RecoverAfterMs: now.Add(15 * time.Second).UnixMilli(),
	}
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: now}).
		WithLocations(locs)

	status, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join while claimed: %v", err)
	}
	// Active claim without durable match yet: recovery path, not a new searching entry.
	if status.Status != matchmaking.PublicStatusTemporarilyUnavailable {
		t.Fatalf("status = %q, want temporarily_unavailable during active claim", status.Status)
	}
	preserved := queue.entries[userA]
	if preserved == nil || preserved.EntryID != original.EntryID || preserved.State != matchmaking.QueueStateClaimed || preserved.ClaimID != claimID {
		t.Fatalf("claimed entry destroyed by duplicate join: got=%+v want entry=%s claim=%s", preserved, original.EntryID, claimID)
	}
	if queue.releaseCalls != 0 {
		t.Fatalf("releaseCalls=%d, claim must remain until finalize/expiry recovery", queue.releaseCalls)
	}
}

func TestStatusDTOUnions(t *testing.T) {
	notQueued := matchmaking.NewNotQueuedStatus()
	if notQueued.Status != matchmaking.PublicStatusNotQueued || notQueued.Queue != nil || notQueued.Match != nil {
		t.Fatalf("not_queued shape invalid: %+v", notQueued)
	}

	now := time.Now().UTC()
	searching := matchmaking.NewSearchingStatus(matchmaking.ModeRankedStandard, now, now.Add(30*time.Second))
	if searching.Status != matchmaking.PublicStatusSearching || searching.Queue == nil || searching.Match != nil {
		t.Fatalf("searching shape invalid: %+v", searching)
	}

	matchID, gameID := uuid.New(), uuid.New()
	matched := matchmaking.NewMatchedStatus(matchID, gameID, matchmaking.ModeRankedStandard, now)
	if matched.Status != matchmaking.PublicStatusMatched || matched.Queue != nil || matched.Match == nil {
		t.Fatalf("matched shape invalid: %+v", matched)
	}
	if matched.Match.Destination != matchmaking.DestinationPath(gameID) {
		t.Fatalf("destination = %q", matched.Match.Destination)
	}

	unavailable := matchmaking.NewTemporarilyUnavailableStatus()
	if unavailable.Status != matchmaking.PublicStatusTemporarilyUnavailable || unavailable.Queue != nil || unavailable.Match != nil {
		t.Fatalf("temporarily_unavailable shape invalid: %+v", unavailable)
	}
}

func testConfig() matchmaking.Config {
	return matchmaking.Config{
		DefaultMapID:       uuid.MustParse("00000000-0000-0000-0000-000000000099"),
		QueueLease:         30 * time.Second,
		ClaimTTL:           15 * time.Second,
		StartDelay:         5 * time.Second,
		RoundCount:         5,
		TimerSeconds:       60,
		CandidateScanLimit: 20,
	}
}

func TestJoinQueue_FormsMatchForBothPlayers(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID:      claimID,
		FormationKey: claimID,
		Mode:         matchmaking.ModeRankedStandard,
		UserIDA:      userA,
		UserIDB:      userB,
		EntryIDA:     "e-a",
		EntryIDB:     "e-b",
	}
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	now := time.Date(2026, 7, 11, 16, 0, 0, 0, time.UTC)
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: now}).
		WithLocations(locs)

	// Seed queue entry for A so join can return searching before formation.
	_, _ = queue.Join(context.Background(), userA, matchmaking.ModeRankedStandard, now, 30*time.Second)

	resp, err := svc.JoinQueue(context.Background(), userSession(userB), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join B: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusMatched {
		t.Fatalf("status = %q, want matched", resp.Status)
	}
	if resp.Match == nil || resp.Match.GameID == uuid.Nil || resp.Match.Destination == "" {
		t.Fatalf("matched payload incomplete: %+v", resp.Match)
	}
	if store.formationCalls != 1 || queue.finalizeCalls != 1 {
		t.Fatalf("formationCalls=%d finalizeCalls=%d", store.formationCalls, queue.finalizeCalls)
	}
	// Both players have durable assignments.
	if store.assignments[userA] == nil || store.assignments[userB] == nil {
		t.Fatal("expected both players assigned")
	}
}

func TestJoinQueue_PostFormationAssignmentRefreshFailureReturnsUnavailable(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	store.assignmentFailWhenPresent = true
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locations := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithLocations(locations)

	status, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrUnavailable {
		t.Fatalf("status=%+v err=%v, want ErrUnavailable", status, err)
	}
	if store.assignments[userA] == nil {
		t.Fatal("formation must be durably committed before refresh failure")
	}
}

func TestJoinQueue_PostClaimPostgresFailureReturnsUnavailable(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	store.matchLookupErr = matchmaking.ErrUnavailable
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locations := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithLocations(locations)

	status, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrUnavailable {
		t.Fatalf("status=%+v err=%v, want ErrUnavailable", status, err)
	}
	if queue.releaseCalls != 0 {
		t.Fatalf("releaseCalls=%d, want claim preserved while commit status is unknown", queue.releaseCalls)
	}
}

func TestFormation_LocationShortageRequeues(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	// Join content gate succeeds once; formation call returns shortage.
	full := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	locs := &shrinkingLocations{full: full, shortAfter: 1}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	_, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if store.formationCalls != 0 {
		t.Fatalf("formation should not commit without locations")
	}
	if queue.releaseCalls != 1 || queue.lastRequeueA == nil || !*queue.lastRequeueA || queue.lastRequeueB == nil || !*queue.lastRequeueB {
		t.Fatalf("expected requeue both on location shortage, A=%v B=%v calls=%d", queue.lastRequeueA, queue.lastRequeueB, queue.releaseCalls)
	}
}

type shrinkingLocations struct {
	full       []uuid.UUID
	calls      int
	shortAfter int
}

func (l *shrinkingLocations) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]uuid.UUID, error) {
	l.calls++
	if l.calls > l.shortAfter {
		return l.full[:1], nil
	}
	if len(l.full) >= count {
		return l.full[:count], nil
	}
	return l.full, nil
}

func TestFormation_DatabaseFailurePreservesClaimForDurableRecovery(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	store.formationFail = matchmaking.ErrUnavailable
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	resp, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrUnavailable {
		t.Fatalf("status=%+v err=%v, want ErrUnavailable after ambiguous formation failure", resp, err)
	}
	if queue.releaseCalls != 0 {
		t.Fatalf("releaseCalls = %d, want 0 while commit outcome is unknown", queue.releaseCalls)
	}
}

func TestFormation_CandidateIneligibleRequeuesEligibleOnly(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "disabled"}
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	_, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if store.formationCalls != 0 {
		t.Fatal("formation must not commit with ineligible candidate")
	}
	// Eligible A must be requeued; ineligible B must not.
	if queue.releaseCalls != 1 || queue.lastRequeueA == nil || !*queue.lastRequeueA {
		t.Fatalf("expected requeue eligible A, got A=%v", queue.lastRequeueA)
	}
	if queue.lastRequeueB == nil || *queue.lastRequeueB {
		t.Fatalf("expected discard ineligible B, got B=%v", queue.lastRequeueB)
	}
}

func TestFormation_ConflictingCandidateRequeuesEligibleOnly(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	store.conflicts[userB] = true
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	_, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if store.formationCalls != 0 {
		t.Fatal("formation must not commit with a conflicting active game")
	}
	if queue.releaseCalls != 1 || queue.lastRequeueA == nil || !*queue.lastRequeueA {
		t.Fatalf("expected requeue eligible A, got A=%v", queue.lastRequeueA)
	}
	if queue.lastRequeueB == nil || *queue.lastRequeueB {
		t.Fatalf("expected discard conflicting B, got B=%v", queue.lastRequeueB)
	}
}

func TestReconcileExpiredClaim_DurableFirstFinalizes(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	matchID, gameID := uuid.New(), uuid.New()
	claimID := uuid.NewString()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	store.matchesByKey[claimID] = &matchmaking.Match{
		ID: matchID, FormationKey: claimID, GameID: gameID, Mode: matchmaking.ModeRankedStandard,
		Status: matchmaking.MatchStatusActive, MatchedAt: now,
	}
	queue := newMemoryQueue()
	queue.expiredClaimIDs = []string{claimID}
	queue.claims[claimID] = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
		RecoverAfterMs: now.Add(-time.Second).UnixMilli(),
	}
	// User already has durable assignment — join returns matched without new claim.
	store.assignments[userA] = &matchmaking.ActiveAssignment{
		MatchID: matchID, GameID: gameID, Mode: matchmaking.ModeRankedStandard,
		Status: matchmaking.MatchStatusActive, MatchedAt: now, UserID: userA,
	}
	// Drive reconcile via GetStatus (request-driven formation path).
	// Clear assignment temporarily so status path runs tryFormMatch reconcile.
	delete(store.assignments, userA)
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: now}).
		WithLocations(locs)

	_, err := svc.GetStatus(context.Background(), userSession(userA))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if queue.finalizeCalls < 1 {
		t.Fatalf("finalizeCalls = %d, want >=1 (durable-first)", queue.finalizeCalls)
	}
	if queue.releaseCalls != 0 {
		t.Fatalf("releaseCalls = %d, want 0 when formation already committed", queue.releaseCalls)
	}
}

func TestFormation_FinalizeRedisFailureStillMatched(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	queue := newMemoryQueue()
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	// Finalize fails after durable commit: force redis down only for finalize by
	// using a custom sequence — set redisDown after claim is taken via formation path.
	// Simpler: leave finalize returning error by flipping redisDown after claim.
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}

	// Wrap finalize failure: claim works, then redisDown for finalize.
	failFinalize := &failFinalizeQueue{memoryQueue: queue}
	svc := matchmaking.NewService(store, failFinalize, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	resp, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusMatched {
		t.Fatalf("status = %q, want matched despite finalize failure", resp.Status)
	}
	if store.assignments[userA] == nil {
		t.Fatal("durable assignment required after finalize failure")
	}
}

type failFinalizeQueue struct {
	*memoryQueue
}

func (q *failFinalizeQueue) FinalizeClaim(ctx context.Context, claim *matchmaking.PairClaim) error {
	q.finalizeCalls++
	return matchmaking.ErrUnavailable
}

func TestLeaveQueue_AbsentAndSearching(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	queue := newMemoryQueue()
	locs := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(locs)

	// Absent leave is idempotent.
	if err := svc.LeaveQueue(context.Background(), userSession(userID)); err != nil {
		t.Fatalf("leave absent: %v", err)
	}

	_, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := svc.LeaveQueue(context.Background(), userSession(userID)); err != nil {
		t.Fatalf("leave searching: %v", err)
	}
	status, err := svc.GetStatus(context.Background(), userSession(userID))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != matchmaking.PublicStatusNotQueued {
		t.Fatalf("status = %q, want not_queued", status.Status)
	}
}

func TestLeaveQueue_ClaimedReturnsConflict(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	queue := newMemoryQueue()
	queue.entries[userID] = &matchmaking.QueueEntry{
		EntryID: "e1", UserID: userID, Mode: matchmaking.ModeRankedStandard,
		State: matchmaking.QueueStateClaimed, ClaimID: "c1",
		EnqueuedAtMs: time.Now().UnixMilli(), LeaseExpiresAtMs: time.Now().Add(time.Minute).UnixMilli(),
	}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil)
	if err := svc.LeaveQueue(context.Background(), userSession(userID)); err != matchmaking.ErrClaimInProgress {
		t.Fatalf("err = %v, want ErrClaimInProgress", err)
	}
}

func TestGetStatus_DurableFirstAndUnavailable(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	matchID, gameID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	store.assignments[userID] = &matchmaking.ActiveAssignment{
		MatchID: matchID, GameID: gameID, Mode: matchmaking.ModeRankedStandard, Status: matchmaking.MatchStatusMatched, MatchedAt: now, UserID: userID,
	}
	queue := newMemoryQueue()
	queue.redisDown = true
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil)

	// Durable match wins even when Redis is down.
	status, err := svc.GetStatus(context.Background(), userSession(userID))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != matchmaking.PublicStatusMatched {
		t.Fatalf("status = %q, want matched", status.Status)
	}

	// No durable assignment + redis down => temporarily unavailable (not not_queued).
	delete(store.assignments, userID)
	status, err = svc.GetStatus(context.Background(), userSession(userID))
	if err != nil {
		t.Fatalf("status without durable: %v", err)
	}
	if status.Status != matchmaking.PublicStatusTemporarilyUnavailable {
		t.Fatalf("status = %q, want temporarily_unavailable", status.Status)
	}
}

func TestGetStatus_ClaimedWithDurableFormation(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	claimID := uuid.NewString()
	matchID, gameID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	store.matchesByKey[claimID] = &matchmaking.Match{
		ID: matchID, FormationKey: claimID, GameID: gameID, Mode: matchmaking.ModeRankedStandard,
		Status: matchmaking.MatchStatusMatched, MatchedAt: now,
	}
	queue := newMemoryQueue()
	queue.entries[userID] = &matchmaking.QueueEntry{
		EntryID: "e1", UserID: userID, Mode: matchmaking.ModeRankedStandard,
		State: matchmaking.QueueStateClaimed, ClaimID: claimID,
		EnqueuedAtMs: now.UnixMilli(), LeaseExpiresAtMs: now.Add(time.Minute).UnixMilli(),
	}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithClock(stubClock{now: now})
	status, err := svc.GetStatus(context.Background(), userSession(userID))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != matchmaking.PublicStatusMatched || status.Match == nil || status.Match.MatchID != matchID {
		t.Fatalf("expected durable recovery from claim, got %+v", status)
	}
}

func TestGetStatus_ClaimedWithPostgresUnavailablePreservesClaim(t *testing.T) {
	userID := uuid.New()
	claimID := uuid.NewString()
	now := time.Now().UTC()
	store := newMemoryStore()
	store.postgresDown = true
	queue := newMemoryQueue()
	queue.entries[userID] = &matchmaking.QueueEntry{
		EntryID: "e1", UserID: userID, Mode: matchmaking.ModeRankedStandard,
		State: matchmaking.QueueStateClaimed, ClaimID: claimID,
		EnqueuedAtMs: now.UnixMilli(), LeaseExpiresAtMs: now.Add(time.Minute).UnixMilli(),
	}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithClock(stubClock{now: now})

	status, err := svc.GetStatus(context.Background(), userSession(userID))
	if err != matchmaking.ErrUnavailable {
		t.Fatalf("status=%+v err=%v, want ErrUnavailable", status, err)
	}
	if queue.releaseCalls != 0 {
		t.Fatalf("releaseCalls = %d, want 0 while PostgreSQL is unavailable", queue.releaseCalls)
	}
}

func TestGetStatus_PostgresUnavailableNeverReportsRedisState(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry *matchmaking.QueueEntry
	}{
		{name: "absent"},
		{name: "searching", entry: &matchmaking.QueueEntry{
			EntryID: "e1", Mode: matchmaking.ModeRankedStandard, State: matchmaking.QueueStateSearching,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			userID := uuid.New()
			store := newMemoryStore()
			store.postgresDown = true
			queue := newMemoryQueue()
			if tc.entry != nil {
				entry := *tc.entry
				entry.UserID = userID
				queue.entries[userID] = &entry
			}
			svc := matchmaking.NewService(store, queue, testConfig(), nil, nil)
			status, err := svc.GetStatus(context.Background(), userSession(userID))
			if err != matchmaking.ErrUnavailable {
				t.Fatalf("status=%+v err=%v, want ErrUnavailable", status, err)
			}
		})
	}
}

func TestFormation_ReleaseFailureReturnsUnavailable(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	store := newMemoryStore()
	store.users[userA] = &matchmaking.ActiveUser{ID: userA, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "disabled"}
	queue := newMemoryQueue()
	queue.releaseErr = matchmaking.ErrUnavailable
	claimID := uuid.NewString()
	queue.nextClaim = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userA, UserIDB: userB, EntryIDA: "e-a", EntryIDB: "e-b",
	}
	locations := &memoryLocations{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithLocations(locations)

	status, err := svc.JoinQueue(context.Background(), userSession(userA), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrUnavailable {
		t.Fatalf("status=%+v err=%v, want ErrUnavailable", status, err)
	}
}

type recordedCommand struct {
	command string
	outcome string
}

type commandMetrics struct{ commands []recordedCommand }

func (m *commandMetrics) ObserveCommand(command, outcome string, _ time.Duration) {
	m.commands = append(m.commands, recordedCommand{command: command, outcome: outcome})
}
func (*commandMetrics) ObserveStatus(string)                   {}
func (*commandMetrics) ObserveFormation(string, time.Duration) {}
func (*commandMetrics) ObserveRecovery(string)                 {}
func (*commandMetrics) ObserveStaleEntry()                     {}
func (*commandMetrics) ObserveDependencyFailure(string)        {}
func (*commandMetrics) ObserveRateLimited(string)              {}
func (*commandMetrics) ObserveRankedFormation(string, string, string, time.Duration) {
}

func TestCommandMetricsRecordExactlyOneOutcome(t *testing.T) {
	metrics := &commandMetrics{}
	svc := matchmaking.NewService(newMemoryStore(), newMemoryQueue(), testConfig(), nil, metrics)
	guestID := "guest"
	sess := &session.Context{Kind: session.KindGuest, GuestID: &guestID}
	_, _ = svc.JoinQueue(context.Background(), sess, matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	_ = svc.LeaveQueue(context.Background(), sess)

	want := []recordedCommand{{command: "join", outcome: "unauthorized"}, {command: "leave", outcome: "unauthorized"}}
	if len(metrics.commands) != len(want) {
		t.Fatalf("commands=%+v, want %+v", metrics.commands, want)
	}
	for i := range want {
		if metrics.commands[i] != want[i] {
			t.Fatalf("command[%d]=%+v, want %+v", i, metrics.commands[i], want[i])
		}
	}
}

func TestJoinQueue_RejectsMissingMapContent(t *testing.T) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	queue := newMemoryQueue()
	// No locations selector / empty map.
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithClock(stubClock{now: time.Now().UTC()})
	_, err := svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrContentUnavailable {
		t.Fatalf("err = %v, want ErrContentUnavailable", err)
	}

	// Selector present but insufficient locations.
	svc = matchmaking.NewService(store, queue, testConfig(), nil, nil).
		WithClock(stubClock{now: time.Now().UTC()}).
		WithLocations(&memoryLocations{ids: []uuid.UUID{uuid.New()}})
	_, err = svc.JoinQueue(context.Background(), userSession(userID), matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard})
	if err != matchmaking.ErrContentUnavailable {
		t.Fatalf("err = %v, want ErrContentUnavailable for short location set", err)
	}
}

func TestGetStatus_ExpiredClaimRequeues(t *testing.T) {
	userID := uuid.New()
	userB := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	store.users[userB] = &matchmaking.ActiveUser{ID: userB, Status: "active"}
	claimID := uuid.NewString()
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	queue := newMemoryQueue()
	queue.entries[userID] = &matchmaking.QueueEntry{
		EntryID: "e1", UserID: userID, Mode: matchmaking.ModeRankedStandard,
		State: matchmaking.QueueStateClaimed, ClaimID: claimID,
		EnqueuedAtMs: now.UnixMilli(), LeaseExpiresAtMs: now.Add(time.Minute).UnixMilli(),
	}
	// Expired claim lives in claims map only (not nextClaim) so tryFormMatch does not form a new match.
	queue.claims[claimID] = &matchmaking.PairClaim{
		ClaimID: claimID, FormationKey: claimID, Mode: matchmaking.ModeRankedStandard,
		UserIDA: userID, UserIDB: userB, EntryIDA: "e1", EntryIDB: "e2",
		EnqueuedAtMsA: now.UnixMilli(), EnqueuedAtMsB: now.UnixMilli(),
		RecoverAfterMs: now.Add(-time.Second).UnixMilli(),
	}
	// Custom queue that requeues on release.
	rq := &requeueOnReleaseQueue{memoryQueue: queue}
	svc := matchmaking.NewService(store, rq, testConfig(), nil, nil).WithClock(stubClock{now: now})
	status, err := svc.GetStatus(context.Background(), userSession(userID))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status != matchmaking.PublicStatusSearching {
		t.Fatalf("status = %q, want searching after expired claim requeue", status.Status)
	}
}

type requeueOnReleaseQueue struct {
	*memoryQueue
}

func (q *requeueOnReleaseQueue) GetClaim(ctx context.Context, claimID string) (*matchmaking.PairClaim, error) {
	return q.memoryQueue.GetClaim(ctx, claimID)
}

func (q *requeueOnReleaseQueue) ReleaseClaim(ctx context.Context, claim *matchmaking.PairClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	if err := q.memoryQueue.ReleaseClaim(ctx, claim, requeueA, requeueB, leaseTTL); err != nil {
		return err
	}
	if claim == nil {
		return nil
	}
	now := time.Now().UTC()
	if requeueA {
		q.entries[claim.UserIDA] = &matchmaking.QueueEntry{
			EntryID: claim.EntryIDA, UserID: claim.UserIDA, Mode: claim.Mode,
			State: matchmaking.QueueStateSearching, EnqueuedAtMs: claim.EnqueuedAtMsA,
			LeaseExpiresAtMs: now.Add(leaseTTL).UnixMilli(),
		}
	}
	if requeueB {
		q.entries[claim.UserIDB] = &matchmaking.QueueEntry{
			EntryID: claim.EntryIDB, UserID: claim.UserIDB, Mode: claim.Mode,
			State: matchmaking.QueueStateSearching, EnqueuedAtMs: claim.EnqueuedAtMsB,
			LeaseExpiresAtMs: now.Add(leaseTTL).UnixMilli(),
		}
	}
	return nil
}
