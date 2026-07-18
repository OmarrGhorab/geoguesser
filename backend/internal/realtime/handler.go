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

	// Optional party/match surfaces (US3). Nil when feature not wired.
	tickets *TicketHandler
	match   *MatchHandler
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

// WithTicket attaches the HTTP ticket issuer (POST /realtime/tickets).
func (h *Handler) WithTicket(t *TicketHandler) *Handler {
	if h != nil {
		h.tickets = t
	}
	return h
}

// WithMatch attaches the party/match WebSocket handler.
func (h *Handler) WithMatch(m *MatchHandler) *Handler {
	if h != nil {
		h.match = m
	}
	return h
}

// IssueTicket handles POST /api/v1/realtime/tickets when a TicketHandler is attached.
func (h *Handler) IssueTicket(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.tickets == nil {
		http.Error(w, "realtime tickets unavailable", http.StatusServiceUnavailable)
		return
	}
	h.tickets.Issue(w, r)
}

// MatchWS handles GET /realtime/matches/{matchId}.
func (h *Handler) MatchWS(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.match == nil {
		http.Error(w, "match realtime unavailable", http.StatusServiceUnavailable)
		return
	}
	h.match.Match(w, r)
}

// PartyWS handles GET /realtime/parties/{partyId}.
func (h *Handler) PartyWS(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.match == nil {
		http.Error(w, "party realtime unavailable", http.StatusServiceUnavailable)
		return
	}
	h.match.Party(w, r)
}

// HasMatchTransport reports whether party/match WS is configured.
func (h *Handler) HasMatchTransport() bool {
	return h != nil && h.match != nil
}

// HasTicketIssuer reports whether ticket HTTP issuance is configured.
func (h *Handler) HasTicketIssuer() bool {
	return h != nil && h.tickets != nil
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
		h.logger.InfoContext(r.Context(), "room websocket snapshot write failed", slog.Any("error", err))
		return
	}
	client := NewRoomClient(state.Room.Code, h.hub.QueueSize())
	h.hub.Add(client)
	defer h.hub.Remove(client)

	h.metrics.RecordConnectionOpened(state.Room.Code)
	defer func() {
		_ = h.provider.MarkDisconnected(context.Background(), sess, state.Room.Code, state.Room.Version)
		h.metrics.RecordConnectionClosed(state.Room.Code, "closed")
	}()

	readCtx, cancelRead := context.WithCancel(r.Context())
	defer cancelRead()
	readPayloads := make(chan []byte)
	readErrors := make(chan error, 1)
	go func() {
		for {
			_, inbound, readErr := conn.Read(readCtx)
			if readErr != nil {
				readErrors <- readErr
				return
			}
			select {
			case readPayloads <- inbound:
			case <-readCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case event := <-client.Send:
			outbound, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				h.logger.WarnContext(r.Context(), "room websocket event encode failed", slog.String("event_type", event.Type), slog.Any("error", marshalErr))
				return
			}
			if writeErr := conn.Write(r.Context(), websocket.MessageText, outbound); writeErr != nil {
				return
			}
			h.metrics.RecordEventDelivered(event.Type)
		case inbound := <-readPayloads:
			if len(inbound) > 0 {
				_, _ = h.provider.TouchPresence(r.Context(), sess, state.Room.Code)
			}
		case <-client.Done():
			h.metrics.ObserveSlowConsumer(ChannelKindRoom)
			_ = conn.Close(websocket.StatusPolicyViolation, "slow consumer")
			return
		case <-readErrors:
			return
		case <-r.Context().Done():
			return
		}
	}
}
