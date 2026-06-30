package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/rooms"
	"github.com/raven/geoguess/backend/internal/session"
)

type Handler struct {
	hub      *Hub
	provider RoomStateProvider
	logger   *slog.Logger
	metrics  MetricsRecorder
}

type RoomStateProvider interface {
	GetRoom(ctx context.Context, sess *session.Context, roomCode string) (*rooms.RoomResponse, error)
	TouchPresence(ctx context.Context, sess *session.Context, roomCode string) (*rooms.RoomResponse, error)
	MarkDisconnected(ctx context.Context, sess *session.Context, roomCode string, lastVersion int64) error
}

func NewHandler(hub *Hub, provider RoomStateProvider, logger *slog.Logger, metrics MetricsRecorder) *Handler {
	if hub == nil {
		hub = NewHub()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &Handler{hub: hub, provider: provider, logger: logger, metrics: metrics}
}

func (h *Handler) Room(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		http.Error(w, "room realtime is unavailable", http.StatusServiceUnavailable)
		return
	}
	roomCode := chi.URLParam(r, "roomCode")
	sess := appmiddleware.SessionFromContext(r.Context())
	state, err := h.provider.TouchPresence(r.Context(), sess, roomCode)
	if err != nil {
		http.Error(w, "room realtime auth required", http.StatusForbidden)
		return
	}

	originPatterns := []string{"http://localhost:*", "http://127.0.0.1:*", "https://localhost:*"}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin == "" {
		originPatterns = []string{"*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: originPatterns})
	if err != nil {
		h.logger.InfoContext(r.Context(), "room websocket accept failed", slog.Any("error", err))
		return
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "closed")
	}()

	event, err := NewEvent("snapshot-"+state.Room.Code, EventRoomSnapshot, state.Room.Code, state.Room.GameID, time.Now().UTC(), state.Room.Version, rooms.RoomResponse{Room: state.Room})
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "invalid snapshot")
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "encode snapshot")
		return
	}
	if err := conn.Write(r.Context(), websocket.MessageText, payload); err != nil {
		h.logger.InfoContext(r.Context(), "room websocket snapshot write failed", slog.String("room_code", state.Room.Code), slog.Any("error", err))
		return
	}

	h.metrics.RecordConnectionOpened(state.Room.Code)
	defer func() {
		_ = h.provider.MarkDisconnected(context.Background(), sess, state.Room.Code, state.Room.Version)
		h.metrics.RecordConnectionClosed(state.Room.Code, "closed")
	}()

	for {
		if _, payload, err := conn.Read(r.Context()); err != nil {
			return
		} else if len(payload) > 0 {
			_, _ = h.provider.TouchPresence(r.Context(), sess, state.Room.Code)
		}
	}
}
