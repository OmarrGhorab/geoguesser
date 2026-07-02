package leaderboards

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/challenges"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestServiceRejectsInvalidLimit(t *testing.T) {
	svc := NewService(&serviceStoreStub{}, nil, clock.Fixed(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)), nil, 0, nil)

	_, err := svc.GetGlobal(context.Background(), 101, "")

	if err != ErrInvalidLimit {
		t.Fatalf("GetGlobal error = %v, want ErrInvalidLimit", err)
	}
}

func TestServiceRejectsOversizedCursor(t *testing.T) {
	svc := NewService(&serviceStoreStub{}, nil, clock.Fixed(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)), nil, 0, nil)

	_, err := svc.GetGlobal(context.Background(), 20, strings.Repeat("x", maxCursorLen+1))

	if err != ErrInvalidCursor {
		t.Fatalf("GetGlobal error = %v, want ErrInvalidCursor", err)
	}
}

func TestServiceRejectsInvalidDailyDate(t *testing.T) {
	svc := NewService(&serviceStoreStub{}, nil, clock.Fixed(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)), nil, 0, nil)

	_, err := svc.GetDaily(context.Background(), 20, "", "not-a-date")

	if err != ErrInvalidDate {
		t.Fatalf("GetDaily error = %v, want ErrInvalidDate", err)
	}
}

func TestServiceReturnsEmptyDailyLeaderboardForExistingChallenge(t *testing.T) {
	challengeID := uuid.New()
	store := &serviceStoreStub{
		daily: &challenges.Challenge{ID: challengeID, Type: challenges.TypeDaily},
	}
	svc := NewService(store, nil, clock.Fixed(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)), nil, 0, nil)

	resp, err := svc.GetDaily(context.Background(), 0, "", "")

	if err != nil {
		t.Fatalf("GetDaily failed: %v", err)
	}
	if resp.Page.Limit != defaultLimit {
		t.Fatalf("limit = %d, want %d", resp.Page.Limit, defaultLimit)
	}
	if len(resp.Data) != 0 {
		t.Fatalf("entries = %d, want 0", len(resp.Data))
	}
	if store.dailyListChallengeID != challengeID {
		t.Fatalf("daily challenge id = %s, want %s", store.dailyListChallengeID, challengeID)
	}
}

func TestServiceTrimsGeneralLeaderboardAndReturnsStableCursor(t *testing.T) {
	firstUser := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondUser := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	thirdUser := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	fast := int64(1000)
	slow := int64(2000)
	completedAt := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	store := &serviceStoreStub{
		generalEntries: []Entry{
			{Rank: 1, UserID: firstUser, DisplayNameSnapshot: "First", Score: 5000, GamesPlayed: 1, CompletionDurationMS: &fast, CompletedAt: completedAt},
			{Rank: 2, UserID: secondUser, DisplayNameSnapshot: "Second", Score: 4500, GamesPlayed: 1, CompletionDurationMS: &slow, CompletedAt: completedAt.Add(time.Minute)},
			{Rank: 3, UserID: thirdUser, DisplayNameSnapshot: "Third", Score: 4000, GamesPlayed: 1, CompletedAt: completedAt.Add(2 * time.Minute)},
		},
	}
	svc := NewService(store, nil, clock.Fixed(completedAt), nil, 0, nil)

	resp, err := svc.GetGlobal(context.Background(), 2, "")

	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}
	if store.generalLimit != 3 {
		t.Fatalf("repository limit = %d, want 3", store.generalLimit)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("entries = %d, want 2", len(resp.Data))
	}
	if resp.Page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}
	decoded, err := decodeCursor(*resp.Page.NextCursor)
	if err != nil {
		t.Fatalf("next cursor did not decode: %v", err)
	}
	if decoded.Score != 4500 || decoded.StableID != secondUser || decoded.CompletionDurationMS == nil || *decoded.CompletionDurationMS != slow {
		t.Fatalf("next cursor = %+v, want second entry ordering tuple", decoded)
	}
}

func TestServiceMaterializesDailyChallengeBeforeRead(t *testing.T) {
	store := &serviceStoreStub{}
	provider := &dailyProviderStub{store: store, challengeID: uuid.New()}
	svc := NewService(store, nil, clock.Fixed(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)), nil, 0, provider)

	resp, err := svc.GetDaily(context.Background(), 20, "", "")

	if err != nil {
		t.Fatalf("GetDaily failed: %v", err)
	}
	if !provider.called {
		t.Fatal("expected daily provider to be called")
	}
	if store.dailyListChallengeID != provider.challengeID {
		t.Fatalf("daily challenge id = %s, want %s", store.dailyListChallengeID, provider.challengeID)
	}
	if len(resp.Data) != 0 {
		t.Fatalf("entries = %d, want 0", len(resp.Data))
	}
}

type serviceStoreStub struct {
	globalBoard          *Leaderboard
	mapBoard             *Leaderboard
	daily                *challenges.Challenge
	dailyListChallengeID uuid.UUID
	generalEntries       []Entry
	generalLimit         int
}

func (s *serviceStoreStub) EnsureGlobalLeaderboard(context.Context) (*Leaderboard, error) {
	if s.globalBoard == nil {
		s.globalBoard = &Leaderboard{ID: uuid.New(), Kind: KindGlobal, ScopeKey: "all"}
	}
	return s.globalBoard, nil
}

func (s *serviceStoreStub) EnsureMapLeaderboard(_ context.Context, mapID uuid.UUID) (*Leaderboard, error) {
	if s.mapBoard == nil {
		s.mapBoard = &Leaderboard{ID: uuid.New(), Kind: KindMap, ScopeKey: mapID.String(), MapID: &mapID}
	}
	return s.mapBoard, nil
}

func (s *serviceStoreStub) GetDailyChallengeByDate(context.Context, time.Time) (*challenges.Challenge, error) {
	return s.daily, nil
}

func (s *serviceStoreStub) ListGeneralEntries(_ context.Context, _ uuid.UUID, limit int, _ string) ([]Entry, error) {
	s.generalLimit = limit
	if limit > len(s.generalEntries) {
		limit = len(s.generalEntries)
	}
	return s.generalEntries[:limit], nil
}

func (s *serviceStoreStub) ListDailyEntries(_ context.Context, challengeID uuid.UUID, _ int, _ string) ([]challenges.LeaderboardEntry, error) {
	s.dailyListChallengeID = challengeID
	return nil, nil
}

func (s *serviceStoreStub) MaterializeCompletedGame(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func (s *serviceStoreStub) DailyCacheScopeForGame(context.Context, uuid.UUID) (*string, error) {
	return nil, nil
}

type dailyProviderStub struct {
	store       *serviceStoreStub
	challengeID uuid.UUID
	called      bool
}

func (p *dailyProviderStub) OnGameCompleted(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (p *dailyProviderStub) GetDaily(context.Context, *session.Context, string) (*challenges.ChallengeMetadataResponse, error) {
	p.called = true
	p.store.daily = &challenges.Challenge{ID: p.challengeID, Type: challenges.TypeDaily}
	return &challenges.ChallengeMetadataResponse{}, nil
}
