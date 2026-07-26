package games

// Internal tests for the StartQuickPlay persistence path: single-transaction
// create-and-start, durable creation-key replay, claim/release behavior, and
// concurrent-duplicate handling. Uses the unexported quickPlayStore seam so
// the branching around PostgreSQL writes is testable without a database.

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

type fakeQuickPlayStore struct {
	byKey       map[string]*Game
	createErr   error
	createCalls int
	lastPlayer  *GamePlayer
	lastRounds  []Round
}

func newFakeQuickPlayStore() *fakeQuickPlayStore {
	return &fakeQuickPlayStore{byKey: map[string]*Game{}}
}

func (f *fakeQuickPlayStore) GetGameByCreationIdempotencyKey(_ context.Context, key string) (*Game, error) {
	if game, ok := f.byKey[key]; ok {
		copied := *game
		return &copied, nil
	}
	return nil, nil
}

func (f *fakeQuickPlayStore) CreateAndStartGameBundle(_ context.Context, game *Game, player *GamePlayer, rounds []Round, now time.Time) (*Game, error) {
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	game.ID = uuid.New()
	game.Status = GameStatusActive
	game.StartedAt = &now
	current := 1
	game.CurrentRoundNumber = &current
	if game.CreationIdempotencyKey != nil {
		stored := *game
		f.byKey[*game.CreationIdempotencyKey] = &stored
	}
	f.lastPlayer = player
	f.lastRounds = rounds
	return game, nil
}

type fakeClaimStore struct {
	claimed  bool
	claimErr error
	claims   []string
	releases []string
}

func (f *fakeClaimStore) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	f.claims = append(f.claims, key)
	return f.claimed, f.claimErr
}

func (f *fakeClaimStore) Release(_ context.Context, key string) error {
	f.releases = append(f.releases, key)
	return nil
}

type stubSelector struct {
	locations []maps.SelectedLocation
	err       error
}

func (s stubSelector) SelectLocations(context.Context, uuid.UUID, int) ([]maps.SelectedLocation, error) {
	return s.locations, s.err
}

func distinctLocations(n int) []maps.SelectedLocation {
	out := make([]maps.SelectedLocation, n)
	for i := range out {
		out[i] = maps.SelectedLocation{ID: uuid.New()}
	}
	return out
}

func quickPlayTestService(store *fakeQuickPlayStore, claims IdempotencyStore, selector LocationSelector) *Service {
	svc := NewServiceWithOptions(nil, selector, nil, clock.NewSystem(), slog.Default(), claims, nil).
		WithQuickPlayDefaults(uuid.MustParse("11111111-1111-1111-1111-111111111111"), 5, 60)
	svc.quickPlayStore = store
	return svc
}

func guestSession(id string) *session.Context {
	return &session.Context{Kind: session.KindGuest, GuestID: &id}
}

const quickPlayTestKey = "quick-play-idempotency-key"

func TestStartQuickPlayPersistsAndStartsAtomically(t *testing.T) {
	t.Parallel()
	store := newFakeQuickPlayStore()
	claims := &fakeClaimStore{claimed: true}
	svc := quickPlayTestService(store, claims, stubSelector{locations: distinctLocations(5)})

	resp, err := svc.StartQuickPlay(context.Background(), guestSession("qp-guest"), quickPlayTestKey)
	if err != nil {
		t.Fatalf("StartQuickPlay: %v", err)
	}
	if resp.Game.Mode != GameModeQuickPlay || resp.Game.Status != GameStatusActive {
		t.Fatalf("game = %+v, want active quick_play", resp.Game)
	}
	if resp.Game.RoundCount != 5 || resp.Game.TimerSeconds == nil || *resp.Game.TimerSeconds != 60 {
		t.Fatalf("rounds/timer = %d/%v, want 5/60", resp.Game.RoundCount, resp.Game.TimerSeconds)
	}
	if resp.Game.CurrentRoundNumber == nil || *resp.Game.CurrentRoundNumber != 1 {
		t.Fatalf("current_round_number = %v, want 1", resp.Game.CurrentRoundNumber)
	}
	if store.createCalls != 1 || len(store.lastRounds) != 5 {
		t.Fatalf("createCalls=%d rounds=%d, want 1/5", store.createCalls, len(store.lastRounds))
	}
	for i, round := range store.lastRounds {
		if round.RoundNumber != i+1 || round.Status != RoundStatusPending || round.LocationID == uuid.Nil {
			t.Fatalf("round[%d] = %+v", i, round)
		}
	}
	if store.lastPlayer == nil || store.lastPlayer.GuestIdentityHash == nil || *store.lastPlayer.GuestIdentityHash != "qp-guest" {
		t.Fatalf("player = %+v", store.lastPlayer)
	}
	wantKey := "game:quick_play:guest:qp-guest:" + quickPlayTestKey
	if len(claims.claims) != 1 || claims.claims[0] != wantKey {
		t.Fatalf("claims = %v, want [%s]", claims.claims, wantKey)
	}
	if len(claims.releases) != 0 {
		t.Fatalf("success must keep the in-flight claim, releases = %v", claims.releases)
	}
	if _, ok := store.byKey[wantKey]; !ok {
		t.Fatalf("creation key not persisted; stored keys = %v", store.byKey)
	}
}

func TestStartQuickPlayReplaysPersistedKeyWithoutReclaiming(t *testing.T) {
	t.Parallel()
	store := newFakeQuickPlayStore()
	claims := &fakeClaimStore{claimed: true}
	svc := quickPlayTestService(store, claims, stubSelector{locations: distinctLocations(5)})

	first, err := svc.StartQuickPlay(context.Background(), guestSession("qp-replay"), quickPlayTestKey)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := svc.StartQuickPlay(context.Background(), guestSession("qp-replay"), quickPlayTestKey)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if second.Game.ID != first.Game.ID {
		t.Fatalf("replay id = %s, want original %s", second.Game.ID, first.Game.ID)
	}
	if store.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1 (replay must not create)", store.createCalls)
	}
	if len(claims.claims) != 1 {
		t.Fatalf("claims = %v, replay must not re-claim", claims.claims)
	}
	// A different actor with the same raw key must NOT replay another actor's game.
	other, err := svc.StartQuickPlay(context.Background(), guestSession("qp-other"), quickPlayTestKey)
	if err != nil {
		t.Fatalf("other actor: %v", err)
	}
	if other.Game.ID == first.Game.ID {
		t.Fatal("creation keys must be actor-scoped")
	}
}

func TestStartQuickPlayClaimConflict(t *testing.T) {
	t.Parallel()
	store := newFakeQuickPlayStore()
	claims := &fakeClaimStore{claimed: false}
	svc := quickPlayTestService(store, claims, stubSelector{locations: distinctLocations(5)})

	if _, err := svc.StartQuickPlay(context.Background(), guestSession("qp-conflict"), quickPlayTestKey); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
	}
	if store.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", store.createCalls)
	}
	if len(claims.releases) != 0 {
		t.Fatalf("lost claim must not be released, releases = %v", claims.releases)
	}
}

func TestStartQuickPlayReleasesClaimOnFailures(t *testing.T) {
	t.Parallel()
	selectorErr := errors.New("selector down")
	for _, tc := range []struct {
		name     string
		selector LocationSelector
		storeErr error
		wantErr  error
	}{
		{name: "selector error", selector: stubSelector{err: selectorErr}, wantErr: selectorErr},
		{name: "not enough locations", selector: stubSelector{locations: distinctLocations(3)}, wantErr: ErrNotEnoughLocations},
		{name: "store error", selector: stubSelector{locations: distinctLocations(5)}, storeErr: errors.New("insert failed"), wantErr: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeQuickPlayStore()
			store.createErr = tc.storeErr
			claims := &fakeClaimStore{claimed: true}
			svc := quickPlayTestService(store, claims, tc.selector)
			_, err := svc.StartQuickPlay(context.Background(), guestSession("qp-fail"), quickPlayTestKey)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if len(claims.releases) != 1 {
				t.Fatalf("failure must release the claim exactly once, releases = %v", claims.releases)
			}
		})
	}
}

func TestStartQuickPlayConcurrentDuplicateReplaysViaUniqueViolation(t *testing.T) {
	t.Parallel()
	store := newFakeQuickPlayStore()
	claims := &fakeClaimStore{claimed: true}
	svc := quickPlayTestService(store, claims, stubSelector{locations: distinctLocations(5)})

	// Simulate: no replay row at lookup time, but the insert loses the race
	// (unique violation) — the winner's game must be replayed.
	winner := &Game{ID: uuid.New(), Mode: GameModeQuickPlay, Status: GameStatusActive}
	key := "game:quick_play:guest:qp-race:" + quickPlayTestKey
	store.createErr = errors.New(`duplicate key value violates unique constraint "games_creation_idempotency_uidx" (SQLSTATE 23505)`)
	firstLookup := true
	// Wrap the store so the first lookup misses and the post-conflict lookup hits.
	svc.quickPlayStore = &raceStore{inner: store, winner: winner, key: key, firstLookup: &firstLookup}

	resp, err := svc.StartQuickPlay(context.Background(), guestSession("qp-race"), quickPlayTestKey)
	if err != nil {
		t.Fatalf("err = %v, want replay of winner", err)
	}
	if resp.Game.ID != winner.ID {
		t.Fatalf("game id = %s, want winner %s", resp.Game.ID, winner.ID)
	}
	if len(claims.releases) != 1 {
		t.Fatalf("losing insert must release its claim, releases = %v", claims.releases)
	}
}

type raceStore struct {
	inner       *fakeQuickPlayStore
	winner      *Game
	key         string
	firstLookup *bool
}

func (r *raceStore) GetGameByCreationIdempotencyKey(ctx context.Context, key string) (*Game, error) {
	if *r.firstLookup {
		*r.firstLookup = false
		return nil, nil
	}
	if key == r.key {
		copied := *r.winner
		return &copied, nil
	}
	return r.inner.GetGameByCreationIdempotencyKey(ctx, key)
}

func (r *raceStore) CreateAndStartGameBundle(ctx context.Context, game *Game, player *GamePlayer, rounds []Round, now time.Time) (*Game, error) {
	return r.inner.CreateAndStartGameBundle(ctx, game, player, rounds, now)
}

func TestStartQuickPlayUnavailableWithoutStore(t *testing.T) {
	t.Parallel()
	// Map configured but no persistence wired (repo nil) → 503-style, not a panic.
	svc := NewService(nil, stubSelector{locations: distinctLocations(5)}, clock.NewSystem(), slog.Default()).
		WithQuickPlayDefaults(uuid.MustParse("11111111-1111-1111-1111-111111111111"), 5, 60)
	if _, err := svc.StartQuickPlay(context.Background(), guestSession("qp-nostore"), quickPlayTestKey); !errors.Is(err, ErrQuickPlayUnavailable) {
		t.Fatalf("err = %v, want ErrQuickPlayUnavailable", err)
	}
}
