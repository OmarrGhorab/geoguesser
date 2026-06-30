package challenges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestHandlerRoutesReturnJSON(t *testing.T) {
	handler := NewHandler(handlerServiceStub{}, nil)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/challenges/daily", ""},
		{http.MethodPost, "/challenges/daily/attempts", ""},
		{http.MethodPost, "/challenges/shared", `{"map_id":"` + uuid.NewString() + `","round_count":5}`},
		{http.MethodGet, "/challenges/shared/ABCDE", ""},
		{http.MethodPost, "/challenges/" + uuid.NewString() + "/attempts", ""},
		{http.MethodGet, "/challenges/" + uuid.NewString() + "/results", ""},
		{http.MethodGet, "/challenges/" + uuid.NewString() + "/leaderboard", ""},
		{http.MethodGet, "/streaks/daily", ""},
		{http.MethodGet, "/missions", ""},
		{http.MethodPost, "/missions/" + uuid.NewString() + "/claim", ""},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("%s %s content-type = %q body=%s", tc.method, tc.path, got, rec.Body.String())
		}
	}
}

type handlerServiceStub struct{}

func (handlerServiceStub) GetDaily(context.Context, *session.Context, string) (*ChallengeMetadataResponse, error) {
	return nil, ErrChallengeUnavailable
}
func (handlerServiceStub) StartDailyAttempt(context.Context, *session.Context) (*ChallengeAttemptResponse, error) {
	return nil, ErrChallengeUnavailable
}
func (handlerServiceStub) CreateShared(context.Context, *session.Context, CreateSharedChallengeRequest) (*ChallengeMetadataResponse, error) {
	return nil, ErrInvalidChallengeInput
}
func (handlerServiceStub) GetShared(context.Context, *session.Context, string) (*ChallengeMetadataResponse, error) {
	return nil, ErrChallengeNotFound
}
func (handlerServiceStub) StartChallengeAttempt(context.Context, *session.Context, string) (*ChallengeAttemptResponse, error) {
	return nil, ErrChallengeNotFound
}
func (handlerServiceStub) GetResults(context.Context, *session.Context, string) (*ResultResponse, error) {
	return nil, ErrResultsNotReady
}
func (handlerServiceStub) GetLeaderboard(context.Context, *session.Context, string, int, string) (*LeaderboardResponse, error) {
	return nil, ErrResultsNotReady
}
func (handlerServiceStub) GetDailyStreak(context.Context, *session.Context) (*StreakSummary, error) {
	return nil, ErrForbidden
}
func (handlerServiceStub) GetMissions(context.Context, *session.Context) ([]MissionSummary, error) {
	return nil, ErrForbidden
}
func (handlerServiceStub) ClaimMission(context.Context, *session.Context, string) (*MissionSummary, error) {
	return nil, ErrForbidden
}
