package home

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/challenges"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/profiles"
	"github.com/raven/geoguess/backend/internal/session"
)

type profileReaderStub struct {
	profile    *profiles.PublicProfileSummary
	stats      *profiles.StatsSummary
	profileErr error
	statsErr   error
	started    chan<- string
	release    <-chan struct{}
}

func (s profileReaderStub) GetPublicProfile(ctx context.Context, _ uuid.UUID) (*profiles.PublicProfileSummary, error) {
	return s.profile, s.profileErr
}

func (s profileReaderStub) GetStats(ctx context.Context, _ uuid.UUID) (*profiles.StatsSummary, error) {
	notifyStarted(ctx, s.started, s.release, "stats")
	return s.stats, s.statsErr
}

type dailyReaderStub struct {
	response *challenges.ChallengeMetadataResponse
	err      error
	started  chan<- string
	release  <-chan struct{}
}

func (s dailyReaderStub) GetDaily(ctx context.Context, _ *session.Context, _ string) (*challenges.ChallengeMetadataResponse, error) {
	notifyStarted(ctx, s.started, s.release, "daily")
	return s.response, s.err
}

type mapReaderStub struct {
	response *maps.MapListResponse
	err      error
	started  chan<- string
	release  <-chan struct{}
}

func (s mapReaderStub) ListMaps(ctx context.Context, _ maps.ListFilters, _ string, _ int) (*maps.MapListResponse, error) {
	notifyStarted(ctx, s.started, s.release, "maps")
	return s.response, s.err
}

func notifyStarted(ctx context.Context, started chan<- string, release <-chan struct{}, name string) {
	if started == nil {
		return
	}
	select {
	case started <- name:
	case <-ctx.Done():
		return
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
		}
	}
}

func TestServiceGetComposesRealBackendData(t *testing.T) {
	userID := uuid.New()
	mapID := uuid.New()
	challengeID := uuid.New()
	avatar := "https://example.com/avatar.webp"
	country := "EG"
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	service := NewService(
		profileReaderStub{
			profile: &profiles.PublicProfileSummary{UserID: userID, DisplayName: "Radiant", AvatarURL: &avatar, CountryCode: &country},
			stats:   &profiles.StatsSummary{GamesPlayed: 248, TotalScore: 1000, AverageScore: 500, BestScore: 24680, LastPlayedAt: &now},
		},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{
			Challenge: challenges.ChallengeSummary{ID: challengeID, Type: challenges.TypeDaily, Status: challenges.StatusActive},
			Streak:    challenges.StreakSummary{CurrentCount: 4, BestCount: 9},
		}},
		mapReaderStub{response: &maps.MapListResponse{Data: []maps.MapDTO{{ID: mapID, Name: "World", Status: "active"}}}},
		nil,
		nil,
	)

	response, err := service.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if response.Viewer.UserID != userID || response.Viewer.DisplayName != "Radiant" {
		t.Fatalf("viewer = %+v", response.Viewer)
	}
	if response.Stats.GamesPlayed != 248 || response.Stats.BestScore != 24680 {
		t.Fatalf("stats = %+v", response.Stats)
	}
	if response.DailyChallenge.Challenge.ID != challengeID {
		t.Fatalf("daily challenge = %+v", response.DailyChallenge)
	}
	if len(response.RecommendedMaps) != 1 || response.RecommendedMaps[0].ID != mapID {
		t.Fatalf("recommended maps = %+v", response.RecommendedMaps)
	}
}

func TestServiceGetRejectsNonRegisteredSessions(t *testing.T) {
	service := NewService(profileReaderStub{}, dailyReaderStub{}, mapReaderStub{}, nil, nil)
	tests := []struct {
		name string
		sess *session.Context
	}{
		{name: "nil", sess: nil},
		{name: "anonymous", sess: &session.Context{Kind: session.KindAnonymous}},
		{name: "guest", sess: &session.Context{Kind: session.KindGuest}},
		{name: "malformed user id", sess: &session.Context{Kind: session.KindUser, UserID: stringPointer("not-a-uuid")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Get(context.Background(), tt.sess)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Get() error = %v, want ErrUnauthorized", err)
			}
		})
	}
}

func TestServiceGetConcealsInactiveOrMissingViewer(t *testing.T) {
	userID := uuid.New()
	service := NewService(
		profileReaderStub{profile: nil},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{}},
		mapReaderStub{response: &maps.MapListResponse{}},
		nil,
		nil,
	)

	_, err := service.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Get() error = %v, want ErrUnauthorized", err)
	}
}

func TestServiceGetMapsDependencyFailureIsUnavailable(t *testing.T) {
	userID := uuid.New()
	service := NewService(
		profileReaderStub{profile: &profiles.PublicProfileSummary{UserID: userID}, stats: &profiles.StatsSummary{}},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{}},
		mapReaderStub{err: errors.New("postgres unavailable")},
		nil,
		nil,
	)

	_, err := service.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Get() error = %v, want ErrUnavailable", err)
	}
}

func TestServiceGetBoundsAndInitializesRecommendedMaps(t *testing.T) {
	userID := uuid.New()
	mapItems := make([]maps.MapDTO, recommendedMapLimit+1)
	for index := range mapItems {
		mapItems[index].ID = uuid.New()
	}
	service := NewService(
		profileReaderStub{profile: &profiles.PublicProfileSummary{UserID: userID}, stats: &profiles.StatsSummary{}},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{}},
		mapReaderStub{response: &maps.MapListResponse{Data: mapItems}},
		nil,
		nil,
	)

	response, err := service.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(response.RecommendedMaps) != recommendedMapLimit {
		t.Fatalf("recommended map count = %d, want %d", len(response.RecommendedMaps), recommendedMapLimit)
	}

	emptyService := NewService(
		profileReaderStub{profile: &profiles.PublicProfileSummary{UserID: userID}, stats: &profiles.StatsSummary{}},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{}},
		mapReaderStub{response: &maps.MapListResponse{}},
		nil,
		nil,
	)
	emptyResponse, err := emptyService.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
	if err != nil {
		t.Fatalf("Get() empty maps error = %v", err)
	}
	if emptyResponse.RecommendedMaps == nil || len(emptyResponse.RecommendedMaps) != 0 {
		t.Fatalf("empty recommended maps = %#v, want initialized empty slice", emptyResponse.RecommendedMaps)
	}
}

func TestServiceGetStartsThreeBoundedBranchesConcurrently(t *testing.T) {
	userID := uuid.New()
	started := make(chan string, 3)
	release := make(chan struct{})
	service := NewService(
		profileReaderStub{profile: &profiles.PublicProfileSummary{UserID: userID}, stats: &profiles.StatsSummary{}, started: started, release: release},
		dailyReaderStub{response: &challenges.ChallengeMetadataResponse{}, started: started, release: release},
		mapReaderStub{response: &maps.MapListResponse{}, started: started, release: release},
		nil,
		nil,
	)

	done := make(chan error, 1)
	go func() {
		_, err := service.Get(context.Background(), &session.Context{Kind: session.KindUser, UserID: stringPointer(userID.String())})
		done <- err
	}()

	names := make([]string, 0, 3)
	for len(names) < 3 {
		select {
		case name := <-started:
			names = append(names, name)
		case <-time.After(time.Second):
			t.Fatalf("only %d branches started before release: %v", len(names), names)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	sort.Strings(names)
	if got, want := names, []string{"daily", "maps", "stats"}; !equalStrings(got, want) {
		t.Fatalf("started branches = %v, want %v", got, want)
	}
}

func stringPointer(value string) *string { return &value }

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
