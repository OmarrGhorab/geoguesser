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

type stubChatService struct {
	send    *matchplay.SendMessageResult
	sendErr error
	list    *matchplay.MessageListResponse
	listErr error
	att     *matchplay.AttachmentURLResponse
	attErr  error
	muteErr error
	rep     *matchplay.ReportMessageResponse
	repErr  error
}

func (s *stubChatService) SendMessage(context.Context, *session.Context, uuid.UUID, matchplay.SendMessageRequest) (*matchplay.SendMessageResult, error) {
	return s.send, s.sendErr
}
func (s *stubChatService) ListMessages(context.Context, *session.Context, uuid.UUID, int64, int) (*matchplay.MessageListResponse, error) {
	return s.list, s.listErr
}
func (s *stubChatService) GetAttachmentURL(context.Context, *session.Context, uuid.UUID, uuid.UUID) (*matchplay.AttachmentURLResponse, error) {
	return s.att, s.attErr
}
func (s *stubChatService) MuteTeammate(context.Context, *session.Context, uuid.UUID, uuid.UUID) error {
	return s.muteErr
}
func (s *stubChatService) UnmuteTeammate(context.Context, *session.Context, uuid.UUID, uuid.UUID) error {
	return s.muteErr
}
func (s *stubChatService) ReportMessage(context.Context, *session.Context, uuid.UUID, uuid.UUID, matchplay.ReportMessageRequest) (*matchplay.ReportMessageResponse, error) {
	return s.rep, s.repErr
}

func TestChatHandlerSendListMuteReport(t *testing.T) {
	t.Parallel()
	matchID := uuid.New()
	msgID := uuid.New()
	chat := &stubChatService{
		send: &matchplay.SendMessageResult{
			Created: true,
			Message: matchplay.TeamMessageResponse{
				ID: msgID, Sequence: 1, Text: "hi", Status: matchplay.MessageStatusActive,
				Author:    matchplay.MessageAuthorDTO{UserID: uuid.New(), DisplayName: "P"},
				CreatedAt: time.Now().UTC(),
			},
		},
		list: &matchplay.MessageListResponse{
			Data: []matchplay.TeamMessageResponse{},
			Page: matchplay.MessagePageInfo{Limit: 50},
		},
		rep: &matchplay.ReportMessageResponse{Status: "accepted"},
	}
	// Snapshot service unused for chat routes.
	h := matchplay.NewHandler(&stubService{}, nil).WithChatService(chat)

	body := `{"client_message_id":"` + uuid.New().String() + `","text":"hi"}`
	req := httptest.NewRequest(http.MethodPost, "/matches/"+matchID.String()+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"matchId": matchID.String()})
	rec := httptest.NewRecorder()
	h.SendMessage(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Idempotent replay -> 200
	chat.send.Created = false
	req = httptest.NewRequest(http.MethodPost, "/matches/"+matchID.String()+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"matchId": matchID.String()})
	rec = httptest.NewRecorder()
	h.SendMessage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("replay status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/matches/"+matchID.String()+"/messages?after=0&limit=50", nil)
	req = withChiParams(req, map[string]string{"matchId": matchID.String()})
	rec = httptest.NewRecorder()
	h.ListMessages(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	target := uuid.New()
	req = httptest.NewRequest(http.MethodPut, "/matches/"+matchID.String()+"/mutes/"+target.String(), nil)
	req = withChiParams(req, map[string]string{"matchId": matchID.String(), "userId": target.String()})
	rec = httptest.NewRecorder()
	h.MuteTeammate(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("mute status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/matches/"+matchID.String()+"/messages/"+msgID.String()+"/report",
		strings.NewReader(`{"reason":"spam"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"matchId": matchID.String(), "messageId": msgID.String()})
	rec = httptest.NewRecorder()
	h.ReportMessage(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("report status = %d body=%s", rec.Code, rec.Body.String())
	}
	var ack matchplay.ReportMessageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil || ack.Status != "accepted" {
		t.Fatalf("report body = %s err=%v", rec.Body.String(), err)
	}

	// Privacy-safe not found
	chat.listErr = matchplay.ErrNotFound
	req = httptest.NewRequest(http.MethodGet, "/matches/"+uuid.New().String()+"/messages", nil)
	req = withChiParams(req, map[string]string{"matchId": uuid.New().String()})
	rec = httptest.NewRecorder()
	h.ListMessages(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d", rec.Code)
	}

	r := chi.NewRouter()
	h.RegisterChatRoutes(r)
}

func TestChatHandlerRateLimitObserverLabels(t *testing.T) {
	t.Parallel()
	h := matchplay.NewHandler(&stubService{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/matches/x/messages", nil)
	h.RecordRateLimited(req) // should not panic
}
