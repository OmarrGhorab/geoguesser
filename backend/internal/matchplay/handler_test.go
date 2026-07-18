package matchplay_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/session"
)

type stubService struct {
	snapshot *matchplay.MatchSnapshotResponse
	terminal *matchplay.TerminalResultResponse
	leaveErr error
	snapErr  error
}

func (s *stubService) GetSnapshot(context.Context, *session.Context, uuid.UUID) (*matchplay.MatchSnapshotResponse, error) {
	return s.snapshot, s.snapErr
}
func (s *stubService) GetRoundResults(context.Context, *session.Context, uuid.UUID, uuid.UUID) (*matchplay.RoundResultsResponse, error) {
	return nil, matchplay.ErrNotFound
}
func (s *stubService) GetTerminalResult(context.Context, *session.Context, uuid.UUID) (*matchplay.TerminalResultResponse, error) {
	return s.terminal, nil
}
func (s *stubService) Leave(context.Context, *session.Context, uuid.UUID, matchplay.LeaveRequest) error {
	return s.leaveErr
}

func TestHandlerSnapshotAndLeaveRoutes(t *testing.T) {
	t.Parallel()
	matchID := uuid.New()
	svc := &stubService{
		snapshot: &matchplay.MatchSnapshotResponse{
			Match: matchplay.MatchSnapshotDTO{
				ID:          matchID,
				Playlist:    matchplay.PlaylistCasual,
				Format:      matchplay.FormatSolo,
				Status:      matchplay.MatchStatusActive,
				TeamSize:    1,
				Teams:       []matchplay.TeamProjection{},
				TeamMarkers: []matchplay.TeamMarkerDTO{},
				FormedAt:    time.Now().UTC(),
			},
		},
		terminal: &matchplay.TerminalResultResponse{
			Result:      matchplay.MatchResultTeamOneWin,
			Teams:       []matchplay.TeamProjection{},
			Progression: matchplay.CasualNoProgression(),
		},
	}
	h := matchplay.NewHandler(svc, nil)

	// Direct handler call with chi URL params.
	req := httptest.NewRequest(http.MethodGet, "/matches/"+matchID.String(), nil)
	req = withChiParams(req, map[string]string{"matchId": matchID.String()})
	rec := httptest.NewRecorder()
	h.GetSnapshot(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d body=%s", rec.Code, rec.Body.String())
	}
	var snap matchplay.MatchSnapshotResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snap.Match.ID != matchID {
		t.Fatalf("id = %s", snap.Match.ID)
	}

	// Leave with empty body + idempotency key
	req = httptest.NewRequest(http.MethodPost, "/matches/"+matchID.String()+"/leave", strings.NewReader(""))
	req = withChiParams(req, map[string]string{"matchId": matchID.String()})
	req.Header.Set("Idempotency-Key", "leave-1")
	rec = httptest.NewRecorder()
	h.Leave(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("leave status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Privacy-safe not found
	svc.snapErr = matchplay.ErrNotFound
	req = httptest.NewRequest(http.MethodGet, "/matches/"+uuid.New().String(), nil)
	req = withChiParams(req, map[string]string{"matchId": uuid.New().String()})
	rec = httptest.NewRecorder()
	h.GetSnapshot(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d", rec.Code)
	}

	// RegisterRoutes mounts expected paths.
	r := chi.NewRouter()
	h.RegisterRoutes(r)
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
