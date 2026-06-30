package realtime

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestHandlerSendsInitialSnapshot(t *testing.T) {
	roomID := uuid.New()
	provider := stubRoomProvider{room: &rooms.RoomResponse{Room: rooms.RoomDTO{ID: roomID, Code: "ABC123", Version: 1, Players: []rooms.RoomPlayerDTO{}}}}
	handler := NewHandler(NewHub(), provider, nil, nil)
	router := chi.NewRouter()
	router.Get("/realtime/rooms/{roomCode}", handler.Room)
	server := httptest.NewServer(router)
	defer server.Close()

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws"+server.URL[len("http"):]+"/realtime/rooms/ABC123", nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	}()

	_, payload, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !strings.Contains(string(payload), "room.snapshot") || !strings.Contains(string(payload), "ABC123") {
		t.Fatalf("snapshot payload = %s", string(payload))
	}
}

type stubRoomProvider struct {
	room *rooms.RoomResponse
}

func (s stubRoomProvider) GetRoom(context.Context, *session.Context, string) (*rooms.RoomResponse, error) {
	return s.room, nil
}

func (s stubRoomProvider) TouchPresence(context.Context, *session.Context, string) (*rooms.RoomResponse, error) {
	return s.room, nil
}

func (s stubRoomProvider) MarkDisconnected(context.Context, *session.Context, string, int64) error {
	return nil
}
