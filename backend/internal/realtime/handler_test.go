package realtime

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
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

func TestHandlerFansPartyLobbyCompletionToAllRoomClients(t *testing.T) {
	roomID := uuid.New()
	gameID := uuid.New()
	timer := 180
	provider := stubRoomProvider{room: &rooms.RoomResponse{Room: rooms.RoomDTO{
		ID: roomID, Code: "PARTY1", GameID: &gameID, Mode: games.GameModePartyLobby,
		Version: 7, TimerSeconds: &timer, Players: []rooms.RoomPlayerDTO{},
		Standings: []rooms.PartyLobbyStanding{{Placement: 1, PlayerID: uuid.New(), DisplayName: "Host", TotalScore: 5000}},
	}}}
	hub := NewHub()
	handler := NewHandler(hub, provider, nil, nil)
	router := chi.NewRouter()
	router.Get("/realtime/rooms/{roomCode}", handler.Room)
	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + server.URL[len("http"):] + "/realtime/rooms/PARTY1"
	clients := make([]*websocket.Conn, 2)
	for i := range clients {
		conn, _, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Fatalf("dial client %d: %v", i, err)
		}
		clients[i] = conn
		defer func(conn *websocket.Conn) { _ = conn.Close(websocket.StatusNormalClosure, "done") }(conn)
		_, snapshot, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read client %d snapshot: %v", i, err)
		}
		if !strings.Contains(string(snapshot), `"mode":"party_lobby"`) || !strings.Contains(string(snapshot), `"timer_seconds":180`) || !strings.Contains(string(snapshot), `"standings"`) {
			t.Fatalf("client %d Party Lobby snapshot=%s", i, snapshot)
		}
	}
	deadline := time.Now().Add(time.Second)
	for hub.SubscriberCount(ChannelKey{Kind: ChannelKindRoom, ID: "PARTY1"}) != 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := hub.SubscriberCount(ChannelKey{Kind: ChannelKindRoom, ID: "PARTY1"}); got != 2 {
		t.Fatalf("room subscribers=%d, want 2", got)
	}
	event, err := NewEvent("party-completed", EventGameCompleted, "PARTY1", &gameID, time.Now().UTC(), 8, provider.room)
	if err != nil {
		t.Fatalf("build completion event: %v", err)
	}
	hub.Broadcast(ctx, "PARTY1", event)
	for i, conn := range clients {
		_, payload, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read client %d completion: %v", i, err)
		}
		var delivered Event
		if err := json.Unmarshal(payload, &delivered); err != nil {
			t.Fatalf("decode client %d completion: %v", i, err)
		}
		if delivered.Type != EventGameCompleted || delivered.Version != 8 {
			t.Fatalf("client %d completion=%+v", i, delivered)
		}
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
