package matchmaking_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/session"
)

type stubService struct {
	joinResp   *matchmaking.StatusResponse
	joinErr    error
	leaveErr   error
	statusResp *matchmaking.StatusResponse
	statusErr  error
	lastJoin   matchmaking.JoinQueueRequest
}

func (s *stubService) JoinQueue(ctx context.Context, sess *session.Context, req matchmaking.JoinQueueRequest) (*matchmaking.StatusResponse, error) {
	s.lastJoin = req
	return s.joinResp, s.joinErr
}

func (s *stubService) LeaveQueue(ctx context.Context, sess *session.Context) error {
	return s.leaveErr
}

func (s *stubService) GetStatus(ctx context.Context, sess *session.Context) (*matchmaking.StatusResponse, error) {
	return s.statusResp, s.statusErr
}

func TestJoinQueueHandler_AcceptedSearching(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	svc := &stubService{
		joinResp: matchmaking.NewSearchingStatus(matchmaking.ModeRankedStandard, now, now.Add(30*time.Second)),
	}
	h := matchmaking.NewHandler(svc, nil)

	body := []byte(`{"mode":"ranked_standard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/matchmaking/queue", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.JoinQueue(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s, want 202", rr.Code, rr.Body.String())
	}
	var resp matchmaking.StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusSearching {
		t.Fatalf("status = %q", resp.Status)
	}
	raw := strings.ToLower(rr.Body.String())
	for _, forbidden := range []string{"email", "opponent", "token", "latitude", "longitude", "password"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, raw)
		}
	}
}

func TestJoinQueueHandler_InvalidJSON(t *testing.T) {
	h := matchmaking.NewHandler(&stubService{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/matchmaking/queue", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.JoinQueue(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestLeaveQueueHandler_NoContent(t *testing.T) {
	h := matchmaking.NewHandler(&stubService{}, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/matchmaking/queue", nil)
	rr := httptest.NewRecorder()
	h.LeaveQueue(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestGetStatusHandler_OK(t *testing.T) {
	svc := &stubService{statusResp: matchmaking.NewNotQueuedStatus()}
	h := matchmaking.NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/matchmaking/status", nil)
	rr := httptest.NewRecorder()
	h.GetStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestJoinQueueHandler_MatchedPrivacy(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	matchID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	gameID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	svc := &stubService{
		joinResp: matchmaking.NewMatchedStatus(matchID, gameID, matchmaking.ModeRankedStandard, now),
	}
	h := matchmaking.NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/matchmaking/queue", bytes.NewReader([]byte(`{"mode":"ranked_standard"}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.JoinQueue(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s, want 202", rr.Code, rr.Body.String())
	}
	var resp matchmaking.StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusMatched || resp.Match == nil || resp.Queue != nil {
		t.Fatalf("matched shape invalid: %+v", resp)
	}
	if resp.Match.MatchID != matchID || resp.Match.GameID != gameID {
		t.Fatalf("ids mismatch: %+v", resp.Match)
	}
	if resp.Match.Destination != "/games/"+gameID.String() {
		t.Fatalf("destination = %q", resp.Match.Destination)
	}
	raw := strings.ToLower(rr.Body.String())
	for _, forbidden := range []string{"email", "opponent", "token", "latitude", "longitude", "password", "user_id", "entry_id"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, raw)
		}
	}
}

func TestGetStatusHandler_MatchedPrivacy(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	matchID := uuid.New()
	gameID := uuid.New()
	svc := &stubService{statusResp: matchmaking.NewMatchedStatus(matchID, gameID, matchmaking.ModeRankedStandard, now)}
	h := matchmaking.NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/matchmaking/status", nil)
	rr := httptest.NewRecorder()
	h.GetStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	raw := strings.ToLower(rr.Body.String())
	for _, forbidden := range []string{"email", "opponent", "latitude", "longitude", "password"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, raw)
		}
	}
}

func TestLeaveQueueHandler_ClaimInProgress(t *testing.T) {
	h := matchmaking.NewHandler(&stubService{leaveErr: matchmaking.ErrClaimInProgress}, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/matchmaking/queue", nil)
	rr := httptest.NewRecorder()
	h.LeaveQueue(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}

func TestGetStatusHandler_TemporarilyUnavailable(t *testing.T) {
	svc := &stubService{statusResp: matchmaking.NewTemporarilyUnavailableStatus()}
	h := matchmaking.NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/matchmaking/status", nil)
	rr := httptest.NewRecorder()
	h.GetStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp matchmaking.StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != matchmaking.PublicStatusTemporarilyUnavailable {
		t.Fatalf("status = %q", resp.Status)
	}
}

func TestPrivacyRegression_NoSensitiveFieldsInMatchmakingJSON(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	payloads := []*matchmaking.StatusResponse{
		matchmaking.NewNotQueuedStatus(),
		matchmaking.NewSearchingStatus(matchmaking.ModeRankedStandard, now, now.Add(30*time.Second)),
		matchmaking.NewMatchedStatus(uuid.New(), uuid.New(), matchmaking.ModeRankedStandard, now),
		matchmaking.NewTemporarilyUnavailableStatus(),
	}
	for _, payload := range payloads {
		h := matchmaking.NewHandler(&stubService{statusResp: payload, joinResp: payload}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/matchmaking/status", nil)
		rr := httptest.NewRecorder()
		h.GetStatus(rr, req)
		raw := strings.ToLower(rr.Body.String())
		for _, forbidden := range []string{
			"email", "password", "token", "refresh", "csrf", "opponent", "preference",
			"latitude", "longitude", "location_id", "provider_ref", "user_id", "entry_id",
		} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("status payload leaked %q: %s", forbidden, raw)
			}
		}
	}
}

func TestMapErrorCodes(t *testing.T) {
	for _, err := range []error{
		matchmaking.ErrUnauthorized,
		matchmaking.ErrAccountIneligible,
		matchmaking.ErrUnsupportedMode,
		matchmaking.ErrAlreadyAssigned,
		matchmaking.ErrActiveGameConflict,
		matchmaking.ErrClaimInProgress,
		matchmaking.ErrContentUnavailable,
		matchmaking.ErrUnavailable,
		matchmaking.ErrInvalidRequest,
	} {
		if mapped := matchmaking.MapError(err); mapped == nil {
			t.Fatalf("MapError(%v) = nil", err)
		}
	}
}
