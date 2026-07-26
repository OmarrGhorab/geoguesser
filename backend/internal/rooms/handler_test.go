package rooms

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

func TestHandlerCreateJoinAndGetRoom(t *testing.T) {
	service := &stubRoomService{room: &RoomResponse{Room: RoomDTO{ID: uuid.New(), Code: "ABC123", Players: []RoomPlayerDTO{}}}}
	handler := NewHandler(service, nil)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	for _, tc := range []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodPost, "/rooms", `{"map_id":"00000000-0000-0000-0000-000000000001","visibility":"private","round_count":5,"max_players":8}`, http.StatusCreated},
		{http.MethodPost, "/rooms/join", `{"code":"ABC123"}`, http.StatusOK},
		{http.MethodGet, "/rooms/ABC123", ``, http.StatusOK},
		{http.MethodPatch, "/rooms/ABC123/settings", `{"round_count":3}`, http.StatusOK},
		{http.MethodPost, "/rooms/ABC123/ready", `{"ready":true}`, http.StatusOK},
		{http.MethodPost, "/rooms/ABC123/start", ``, http.StatusOK},
		{http.MethodDelete, "/rooms/ABC123/players/me", ``, http.StatusNoContent},
		{http.MethodDelete, "/rooms/ABC123/players/00000000-0000-0000-0000-000000000001", ``, http.StatusOK},
		{http.MethodDelete, "/rooms/ABC123", ``, http.StatusNoContent},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.Header.Set("Idempotency-Key", "test-key")
		router.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s %s status = %d body=%s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}

func TestHandlerLeaveSelfAndCancelErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		err    error
		want   int
		code   string
	}{
		{name: "host leave", method: http.MethodDelete, path: "/rooms/ABC123/players/me", err: ErrHostActionRequired, want: http.StatusConflict, code: CodeHostActionRequired},
		{name: "leave missing", method: http.MethodDelete, path: "/rooms/ABC123/players/me", err: ErrRoomNotFound, want: http.StatusNotFound, code: CodeRoomNotFound},
		{name: "cancel not lobby", method: http.MethodDelete, path: "/rooms/ABC123", err: ErrRoomNotCancellable, want: http.StatusConflict, code: CodeRoomNotCancellable},
		{name: "cancel non-host", method: http.MethodDelete, path: "/rooms/ABC123", err: ErrRoomHostRequired, want: http.StatusForbidden, code: "forbidden"},
		// Regression: ErrRoomAlreadyStarted keeps its long-standing 422 mapping
		// on non-cancel endpoints (ready/start) — it must never drift to 409.
		{name: "already started stays 422", method: http.MethodDelete, path: "/rooms/ABC123/players/me", err: ErrRoomAlreadyStarted, want: http.StatusUnprocessableEntity, code: "unprocessable_entity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &stubRoomService{leaveErr: tc.err, cancelErr: tc.err}
			handler := NewHandler(service, nil)
			router := chi.NewRouter()
			handler.RegisterRoutes(router)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.path, nil)
			router.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status = %d body=%s, want %d", w.Code, w.Body.String(), tc.want)
			}
			if !strings.Contains(w.Body.String(), tc.code) {
				t.Fatalf("body = %s, want code %s", w.Body.String(), tc.code)
			}
		})
	}
}

type stubRoomService struct {
	room      *RoomResponse
	leaveErr  error
	cancelErr error
}

func (s *stubRoomService) CreateRoom(context.Context, *session.Context, CreateRoomRequest) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) JoinRoom(context.Context, *session.Context, JoinRoomRequest) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) GetRoom(context.Context, *session.Context, string) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) UpdateSettings(context.Context, *session.Context, string, UpdateRoomSettingsRequest) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) SetReady(context.Context, *session.Context, string, ReadyRoomRequest) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) StartRoom(context.Context, *session.Context, string, string) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) RemovePlayer(context.Context, *session.Context, string, uuid.UUID) (*RoomResponse, error) {
	return s.room, nil
}
func (s *stubRoomService) LeaveSelf(context.Context, *session.Context, string) error {
	return s.leaveErr
}
func (s *stubRoomService) CancelRoom(context.Context, *session.Context, string) error {
	return s.cancelErr
}
