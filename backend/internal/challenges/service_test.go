package challenges

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestNormalizeSettingsValidation(t *testing.T) {
	if _, err := normalizeSettings(11, nil); !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("round count error = %v, want invalid input", err)
	}
	tooSmall := 5
	if _, err := normalizeSettings(5, &tooSmall); !errors.Is(err, ErrInvalidChallengeInput) {
		t.Fatalf("timer error = %v, want invalid input", err)
	}
}

func TestSelectUniqueRejectsInsufficientUniqueLocations(t *testing.T) {
	mapID := uuid.New()
	locationID := uuid.New()
	svc := &Service{selector: selectorStub{locations: []maps.SelectedLocation{{ID: locationID}, {ID: locationID}}}}

	if _, err := svc.selectUnique(context.Background(), mapID, 2); !errors.Is(err, ErrNotEnoughLocations) {
		t.Fatalf("selectUnique error = %v, want ErrNotEnoughLocations", err)
	}
}

func TestSelectUniqueBySeedUsesSeededSelector(t *testing.T) {
	mapID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	svc := &Service{selector: selectorStub{locationsBySeed: map[string][]maps.SelectedLocation{
		"seed-a": {{ID: first}, {ID: second}},
		"seed-b": {{ID: second}, {ID: first}},
	}}}

	gotA, err := svc.selectUniqueBySeed(context.Background(), mapID, 2, "seed-a")
	if err != nil {
		t.Fatalf("selectUniqueBySeed seed-a error = %v", err)
	}
	gotB, err := svc.selectUniqueBySeed(context.Background(), mapID, 2, "seed-b")
	if err != nil {
		t.Fatalf("selectUniqueBySeed seed-b error = %v", err)
	}
	if gotA[0].ID != first || gotB[0].ID != second {
		t.Fatalf("seeded selection did not preserve deterministic selector order: gotA=%v gotB=%v", gotA, gotB)
	}
}

func TestIdempotencyReplayAndConflict(t *testing.T) {
	store := newMemoryIdempotencyStore()
	userID := uuid.NewString()
	sess := &session.Context{Kind: session.KindUser, UserID: &userID}
	body := CreateSharedChallengeRequest{MapID: uuid.New(), RoundCount: 5}

	_, op, handled, err := beginIdempotency[ChallengeMetadataResponse](context.Background(), store, "retry-key", "create_shared_challenge", sess, body)
	if err != nil || handled || op == nil {
		t.Fatalf("begin idempotency first call replay=%v op=%v handled=%v err=%v", false, op, handled, err)
	}
	want := &ChallengeMetadataResponse{
		Challenge: ChallengeSummary{ID: uuid.New(), Type: TypeShared, Seed: "seed", Map: MapSummary{ID: body.MapID}, Settings: SettingsSnapshot{RoundCount: 5, MovementRules: "standard", ScoringVersion: 1}, Status: StatusActive},
		Streak:    EmptyStreakSummary(false),
	}
	if err := storeIdempotencyResponse(context.Background(), op, want); err != nil {
		t.Fatalf("store idempotency response failed: %v", err)
	}
	releaseIdempotency(context.Background(), op)

	replay, _, handled, err := beginIdempotency[ChallengeMetadataResponse](context.Background(), store, "retry-key", "create_shared_challenge", sess, body)
	if err != nil || !handled || replay == nil {
		t.Fatalf("begin idempotency replay = %+v handled=%v err=%v", replay, handled, err)
	}
	if replay.Challenge.ID != want.Challenge.ID {
		t.Fatalf("replay challenge id = %s, want %s", replay.Challenge.ID, want.Challenge.ID)
	}

	conflicting := body
	conflicting.RoundCount = 3
	if replay, _, handled, err = beginIdempotency[ChallengeMetadataResponse](context.Background(), store, "retry-key", "create_shared_challenge", sess, conflicting); !handled || !errors.Is(err, ErrIdempotencyConflict) || replay != nil {
		t.Fatalf("conflict replay=%+v handled=%v err=%v, want idempotency conflict", replay, handled, err)
	}
}

type selectorStub struct {
	locations       []maps.SelectedLocation
	locationsBySeed map[string][]maps.SelectedLocation
	err             error
}

func (s selectorStub) SelectLocations(context.Context, uuid.UUID, int) ([]maps.SelectedLocation, error) {
	return s.locations, s.err
}

func (s selectorStub) SelectLocationsBySeed(_ context.Context, _ uuid.UUID, _ int, seed string) ([]maps.SelectedLocation, error) {
	if len(s.locationsBySeed) > 0 {
		return s.locationsBySeed[seed], s.err
	}
	return s.locations, s.err
}

type memoryIdempotencyStore struct {
	records map[string]IdempotencyRecord
	locks   map[string]bool
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{records: map[string]IdempotencyRecord{}, locks: map[string]bool{}}
}

func (s *memoryIdempotencyStore) Get(_ context.Context, key string) (*IdempotencyRecord, error) {
	record, ok := s.records[key]
	if !ok {
		return nil, nil
	}
	return &record, nil
}

func (s *memoryIdempotencyStore) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	if s.locks[key] {
		return false, nil
	}
	s.locks[key] = true
	return true, nil
}

func (s *memoryIdempotencyStore) Release(_ context.Context, key string) error {
	delete(s.locks, key)
	return nil
}

func (s *memoryIdempotencyStore) Store(_ context.Context, key string, record IdempotencyRecord, _ time.Duration) error {
	s.records[key] = record
	return nil
}
