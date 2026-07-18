package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ClientCommand is the inbound WebSocket command envelope.
type ClientCommand struct {
	ProtocolVersion int             `json:"protocol_version"`
	CommandID       string          `json:"command_id"`
	Type            string          `json:"type"`
	ExpectedVersion *int64          `json:"expected_version"`
	Payload         json.RawMessage `json:"payload"`
}

// MarkerCommandPayload is the body for round.marker.set.
type MarkerCommandPayload struct {
	RoundID   string  `json:"round_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ViewCommandPayload is the body for round.view.update (provider-safe only).
type ViewCommandPayload struct {
	RoundID    string  `json:"round_id"`
	PanoramaID string  `json:"panorama_id"`
	Heading    float64 `json:"heading"`
	Pitch      float64 `json:"pitch"`
	Zoom       float64 `json:"zoom"`
}

// SpectateSelectPayload is the body for round.spectate.select.
type SpectateSelectPayload struct {
	// TargetGamePlayerID is the preferred spectate target (game_players.id).
	TargetGamePlayerID string `json:"target_game_player_id"`
	// RoundID is optional; when empty the server uses the active round from policy.
	RoundID string `json:"round_id,omitempty"`
}

// MatchHandler serves party/match WebSocket endpoints with ticket subprotocol auth.
type MatchHandler struct {
	hub          *Hub
	tickets      TicketValidator
	authorizer   ChannelAuthorizer
	snapshots    SnapshotProvider
	commands     MatchCommandService
	lifecycle    ConnectionLifecycle
	fanout       *FanoutPublisher
	origins      []string
	logger       *slog.Logger
	metrics      MetricsRecorder
	pingInterval time.Duration
	queueSize    int
}

// WithConnectionLifecycle attaches durable/ephemeral connection transition handling.
func (h *MatchHandler) WithConnectionLifecycle(lifecycle ConnectionLifecycle) *MatchHandler {
	if h != nil {
		h.lifecycle = lifecycle
	}
	return h
}

// MatchHandlerConfig holds transport knobs.
type MatchHandlerConfig struct {
	AllowedOrigins []string
	PingInterval   time.Duration
	QueueSize      int
}

// NewMatchHandler constructs a party/match WebSocket handler.
func NewMatchHandler(
	hub *Hub,
	tickets TicketValidator,
	authorizer ChannelAuthorizer,
	snapshots SnapshotProvider,
	commands MatchCommandService,
	fanout *FanoutPublisher,
	cfg MatchHandlerConfig,
	logger *slog.Logger,
	metrics MetricsRecorder,
) *MatchHandler {
	if hub == nil {
		hub = NewHub()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = 20 * time.Second
	}
	if cfg.QueueSize < 1 {
		cfg.QueueSize = DefaultOutboundQueueSize
	}
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	return &MatchHandler{
		hub:          hub,
		tickets:      tickets,
		authorizer:   authorizer,
		snapshots:    snapshots,
		commands:     commands,
		fanout:       fanout,
		origins:      origins,
		logger:       logger,
		metrics:      metrics,
		pingInterval: cfg.PingInterval,
		queueSize:    cfg.QueueSize,
	}
}

// Match handles GET /realtime/matches/{matchId}.
func (h *MatchHandler) Match(w http.ResponseWriter, r *http.Request) {
	h.serveChannel(w, r, ChannelKindMatch, chi.URLParam(r, "matchId"))
}

// Party handles GET /realtime/parties/{partyId}.
func (h *MatchHandler) Party(w http.ResponseWriter, r *http.Request) {
	h.serveChannel(w, r, ChannelKindParty, chi.URLParam(r, "partyId"))
}

func (h *MatchHandler) serveChannel(w http.ResponseWriter, r *http.Request, channelKind, channelID string) {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		http.Error(w, "channel required", http.StatusBadRequest)
		return
	}
	if h.tickets == nil {
		http.Error(w, "realtime unavailable", http.StatusServiceUnavailable)
		return
	}

	// Origin allowlist.
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && !originAllowed(origin, h.origins) {
		h.metrics.ObserveTicketConsumed(channelKind, "forbidden")
		http.Error(w, "origin forbidden", http.StatusForbidden)
		return
	}

	protocols := ParseSecWebSocketProtocol(r.Header.Get("Sec-WebSocket-Protocol"))
	hasV1, ticket := DecodeTicketSubprotocols(protocols)
	if !hasV1 || ticket == "" {
		h.metrics.ObserveTicketConsumed(channelKind, "invalid")
		http.Error(w, "ticket required", http.StatusUnauthorized)
		return
	}

	claims, err := h.tickets.Consume(r.Context(), ticket)
	if err != nil {
		outcome := "invalid"
		status := http.StatusUnauthorized
		closeHint := CloseTicketUnauthorized
		switch {
		case errors.Is(err, ErrTicketUsed):
			outcome = "used"
		case errors.Is(err, ErrTicketExpired):
			outcome = "expired"
		}
		h.metrics.ObserveTicketConsumed(channelKind, outcome)
		// HTTP rejection before upgrade when ticket is bad.
		_ = closeHint
		http.Error(w, "ticket unauthorized", status)
		return
	}
	if claims == nil || claims.ChannelKind != channelKind || claims.ChannelID != channelID {
		h.metrics.ObserveTicketConsumed(channelKind, "invalid")
		http.Error(w, "ticket unauthorized", http.StatusUnauthorized)
		return
	}
	h.metrics.ObserveTicketConsumed(channelKind, "consumed")

	// Re-check durable membership after consume.
	var membership ChannelMembership
	if h.authorizer != nil {
		membership, err = h.authorizer.Authorize(r.Context(), claims.UserID, channelKind, channelID)
		if err != nil {
			http.Error(w, "not a participant", http.StatusForbidden)
			return
		}
	} else {
		membership = ChannelMembership{UserID: claims.UserID}
	}

	originPatterns := h.origins
	if origin == "" {
		// Some test clients omit Origin; allow when patterns include wildcards/local.
		originPatterns = append([]string{}, h.origins...)
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPatterns,
		Subprotocols:   []string{SubprotocolV1},
	})
	if err != nil {
		h.logger.InfoContext(r.Context(), "websocket accept failed",
			slog.String("channel_kind", channelKind),
			slog.Any("error", err),
		)
		return
	}
	closeReason := "normal"
	defer func() {
		h.metrics.ObserveSocketClosed(channelKind, closeReason)
		_ = conn.Close(websocket.StatusNormalClosure, "closed")
	}()

	if conn.Subprotocol() != SubprotocolV1 {
		closeReason = "unauthorized"
		_ = conn.Close(websocket.StatusCode(CloseTicketUnauthorized), "protocol")
		return
	}

	client := NewClient(channelKind, channelID, membership.UserID, membership.TeamSlot, h.queueSize)
	h.hub.Subscribe(client)
	channel := ChannelRef{Kind: channelKind, ID: channelID}
	defer func() {
		h.hub.Unsubscribe(client)
		if h.lifecycle == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.lifecycle.Disconnected(cleanupCtx, channel, membership.UserID, h.hub.LastVersion(channel.Key())); err != nil {
			h.logger.WarnContext(cleanupCtx, "realtime disconnect transition failed",
				slog.String("channel_kind", channelKind), slog.Any("error", err))
		}
	}()
	if h.lifecycle != nil {
		if err := h.lifecycle.Connected(r.Context(), channel, membership.UserID, h.hub.LastVersion(channel.Key())); err != nil {
			h.logger.WarnContext(r.Context(), "realtime connection transition failed",
				slog.String("channel_kind", channelKind), slog.Any("error", err))
			closeReason = "error"
			return
		}
	}

	if h.fanout != nil {
		_ = h.fanout.SubscribeChannel(r.Context(), channelKind, channelID)
		defer func() {
			_ = h.fanout.UnsubscribeChannel(context.Background(), channelKind, channelID)
		}()
	}

	h.metrics.ObserveSocketOpened(channelKind)

	// First message: authorized snapshot at current version.
	if err := h.writeSnapshot(r.Context(), conn, membership.UserID, channelKind, channelID); err != nil {
		closeReason = "error"
		return
	}

	// Writer pump: outbound events + slow-consumer detection via hub Done.
	writeCtx, writeCancel := context.WithCancel(r.Context())
	defer writeCancel()
	writeErr := make(chan error, 1)
	go func() {
		writeErr <- h.writePump(writeCtx, conn, client, channelKind)
	}()

	// Reader pump: commands + heartbeats.
	readErr := h.readPump(r.Context(), conn, client, membership, channelKind, channelID)

	writeCancel()
	select {
	case err := <-writeErr:
		if err != nil && closeReason == "normal" {
			if errors.Is(err, errSlowConsumer) {
				closeReason = "slow_consumer"
				_ = conn.Close(websocket.StatusCode(CloseSlowConsumer), "slow consumer")
			} else {
				closeReason = "error"
			}
		}
	case <-time.After(time.Second):
	}

	if readErr != nil && closeReason == "normal" {
		closeReason = "closed"
	}
}

func (h *MatchHandler) writeSnapshot(ctx context.Context, conn *websocket.Conn, userID uuid.UUID, channelKind, channelID string) error {
	var payload any = map[string]any{}
	var version int64
	if h.snapshots != nil {
		p, v, err := h.snapshots.Snapshot(ctx, userID, channelKind, channelID)
		if err != nil {
			return err
		}
		payload = p
		version = v
	}
	eventType := EventMatchSnapshot
	if channelKind == ChannelKindParty {
		eventType = EventPartySnapshot
	}
	evt, err := NewChannelEvent(uuid.NewString(), eventType, channelKind, channelID, nil, nil, time.Now().UTC(), version, payload)
	if err != nil {
		return err
	}
	return writeEvent(ctx, conn, evt)
}

var errSlowConsumer = errors.New("slow consumer")

func (h *MatchHandler) writePump(ctx context.Context, conn *websocket.Conn, client *Client, channelKind string) error {
	pingTicker := time.NewTicker(h.pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-client.Done():
			h.metrics.ObserveSlowConsumer(channelKind)
			return errSlowConsumer
		case evt, ok := <-client.Send:
			if !ok {
				return nil
			}
			if err := writeEvent(ctx, conn, evt); err != nil {
				return err
			}
			h.metrics.RecordEventDelivered(evt.Type)
		case <-pingTicker.C:
			// Application-level keepalive via websocket ping frames.
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return err
			}
			if h.lifecycle != nil {
				_ = h.lifecycle.Heartbeat(ctx, ChannelRef{Kind: client.ChannelKind, ID: client.ChannelID}, client.UserID)
			}
		}
	}
}

func (h *MatchHandler) readPump(
	ctx context.Context,
	conn *websocket.Conn,
	client *Client,
	membership ChannelMembership,
	channelKind, channelID string,
) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue
		}
		var cmd ClientCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			_ = h.writePrivateError(ctx, conn, channelKind, channelID, "", CodeCommandInvalid, "invalid command json", 0)
			continue
		}
		h.dispatchCommand(ctx, conn, client, membership, channelKind, channelID, cmd)
	}
}

func (h *MatchHandler) dispatchCommand(
	ctx context.Context,
	conn *websocket.Conn,
	client *Client,
	membership ChannelMembership,
	channelKind, channelID string,
	cmd ClientCommand,
) {
	if cmd.ProtocolVersion != 0 && cmd.ProtocolVersion != ProtocolVersion {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "unsupported protocol_version", 0)
		h.metrics.ObserveCommand("other", "rejected")
		return
	}
	if !IsKnownCommandType(cmd.Type) {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "unknown command type", 0)
		h.metrics.ObserveCommand("other", "rejected")
		return
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, "", CodeCommandInvalid, "command_id is required", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}

	// Party channels only accept heartbeats in v1 of this handler.
	if channelKind == ChannelKindParty && cmd.Type != CommandPresenceHeartbeat {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "command not supported on party channel", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}

	matchID, err := uuid.Parse(channelID)
	if err != nil && channelKind == ChannelKindMatch {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "invalid match id", 0)
		return
	}

	// Command replay: return prior ack without republishing.
	if channelKind == ChannelKindMatch && h.commands != nil {
		if prior, found, loadErr := h.commands.LoadCommandAck(ctx, matchID, membership.UserID, cmd.CommandID); loadErr == nil && found {
			_ = conn.Write(ctx, websocket.MessageText, prior)
			h.metrics.ObserveCommand(cmd.Type, "replay")
			return
		}
	}

	// Version gap detection when client supplies expected_version.
	if cmd.ExpectedVersion != nil && h.commands != nil && channelKind == ChannelKindMatch {
		current, vErr := h.commands.CurrentVersion(ctx, matchID)
		if vErr == nil && current > 0 && *cmd.ExpectedVersion < current-1 {
			// Gap: client is behind by more than one; ask for HTTP snapshot recovery.
			_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeVersionGap, "version gap; fetch HTTP snapshot", current)
			h.metrics.ObserveCommand(cmd.Type, "version_gap")
			return
		}
	}

	switch cmd.Type {
	case CommandPresenceHeartbeat:
		if h.lifecycle != nil {
			_ = h.lifecycle.Heartbeat(ctx, ChannelRef{Kind: channelKind, ID: channelID}, membership.UserID)
		}
		if channelKind == ChannelKindMatch && h.commands != nil {
			_ = h.commands.Heartbeat(ctx, matchID, membership.UserID)
		}
		_ = h.writeAccepted(ctx, conn, channelKind, channelID, cmd.CommandID, 0, map[string]any{"ok": true})
		h.metrics.ObserveCommand(cmd.Type, "accepted")
	case CommandRoundMarkerSet:
		h.handleMarkerSet(ctx, conn, membership, matchID, channelKind, channelID, cmd)
	case CommandRoundViewUpdate:
		h.handleViewUpdate(ctx, conn, membership, matchID, channelKind, channelID, cmd)
	case CommandRoundSpectateSelect:
		h.handleSpectateSelect(ctx, conn, membership, matchID, channelKind, channelID, cmd)
	}
}

func (h *MatchHandler) handleMarkerSet(
	ctx context.Context,
	conn *websocket.Conn,
	membership ChannelMembership,
	matchID uuid.UUID,
	channelKind, channelID string,
	cmd ClientCommand,
) {
	if h.commands == nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeFeatureDisabled, "commands unavailable", 0)
		h.metrics.ObserveMarkerCommand("error")
		return
	}
	var body MarkerCommandPayload
	if err := json.Unmarshal(cmd.Payload, &body); err != nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "invalid marker payload", 0)
		h.metrics.ObserveMarkerCommand("rejected")
		return
	}
	roundID, err := uuid.Parse(strings.TrimSpace(body.RoundID))
	if err != nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "round_id required", 0)
		h.metrics.ObserveMarkerCommand("rejected")
		return
	}
	teamSlot := 0
	if membership.TeamSlot != nil {
		teamSlot = *membership.TeamSlot
	}
	if teamSlot != 1 && teamSlot != 2 {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "team required", 0)
		h.metrics.ObserveMarkerCommand("rejected")
		return
	}

	version, err := h.commands.SetMarker(ctx, matchID, roundID, membership.UserID, teamSlot, body.Latitude, body.Longitude)
	if err != nil {
		code, msg, outcome := mapCommandError(err)
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, code, msg, 0)
		h.metrics.ObserveMarkerCommand(outcome)
		h.metrics.ObserveCommand(cmd.Type, outcome)
		return
	}

	// Publish team-only marker event after durable/live commit.
	evt, err := NewChannelEvent(
		uuid.NewString(),
		EventRoundMarkerChanged,
		ChannelKindMatch,
		channelID,
		nil,
		&roundID,
		time.Now().UTC(),
		version,
		map[string]any{
			"user_id":   membership.UserID.String(),
			"round_id":  roundID.String(),
			"latitude":  body.Latitude,
			"longitude": body.Longitude,
			"version":   version,
		},
	)
	if err == nil {
		slot := teamSlot
		audience := Audience{Kind: AudienceTeam, TeamSlot: &slot}
		publisher := ChannelPublisher(HubChannelPublisher{Hub: h.hub})
		if h.fanout != nil {
			publisher = h.fanout
		}
		_ = publisher.Publish(ctx, ChannelRef{Kind: ChannelKindMatch, ID: channelID}, audience, evt)
	}

	ackPayload := map[string]any{"version": version}
	_ = h.writeAccepted(ctx, conn, channelKind, channelID, cmd.CommandID, version, ackPayload)
	if h.commands != nil {
		if raw, mErr := json.Marshal(mustAcceptedEvent(channelKind, channelID, cmd.CommandID, version, ackPayload)); mErr == nil {
			_ = h.commands.SaveCommandAck(ctx, matchID, membership.UserID, cmd.CommandID, raw)
		}
	}
	h.metrics.ObserveMarkerCommand("accepted")
	h.metrics.ObserveCommand(cmd.Type, "accepted")
}

func (h *MatchHandler) handleViewUpdate(
	ctx context.Context,
	conn *websocket.Conn,
	membership ChannelMembership,
	matchID uuid.UUID,
	channelKind, channelID string,
	cmd ClientCommand,
) {
	if h.commands == nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeFeatureDisabled, "commands unavailable", 0)
		return
	}
	// Reject map/marker/guess fields if present in raw payload.
	if containsForbiddenViewFields(cmd.Payload) {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "view payload contains forbidden fields", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}
	var body ViewCommandPayload
	if err := json.Unmarshal(cmd.Payload, &body); err != nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "invalid view payload", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}
	roundID, err := uuid.Parse(strings.TrimSpace(body.RoundID))
	if err != nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "round_id required", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}
	version, err := h.commands.SetView(ctx, matchID, roundID, membership.UserID, body.PanoramaID, body.Heading, body.Pitch, body.Zoom)
	if err != nil {
		if errors.Is(err, ErrViewUnchanged) {
			// Non-material: accept without fanout.
			_ = h.writeAccepted(ctx, conn, channelKind, channelID, cmd.CommandID, version, map[string]any{
				"version": version,
				"changed": false,
			})
			h.metrics.ObserveCommand(cmd.Type, "accepted")
			return
		}
		code, msg, outcome := mapCommandError(err)
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, code, msg, 0)
		h.metrics.ObserveCommand(cmd.Type, outcome)
		return
	}

	// Fan out provider-safe scene only to currently authorized spectators.
	recipients, _ := h.commands.ViewRecipients(ctx, matchID, roundID, membership.UserID)
	if len(recipients) > 0 {
		payload := map[string]any{
			"user_id":  membership.UserID.String(),
			"round_id": roundID.String(),
			"heading":  body.Heading,
			"pitch":    body.Pitch,
			"zoom":     body.Zoom,
			"version":  version,
		}
		if body.PanoramaID != "" {
			payload["panorama_id"] = body.PanoramaID
		}
		evt, evtErr := NewChannelEvent(
			uuid.NewString(),
			EventRoundViewChanged,
			ChannelKindMatch,
			channelID,
			nil,
			&roundID,
			time.Now().UTC(),
			version,
			payload,
		)
		if evtErr == nil {
			publisher := ChannelPublisher(HubChannelPublisher{Hub: h.hub})
			if h.fanout != nil {
				publisher = h.fanout
			}
			_ = publisher.Publish(ctx, ChannelRef{Kind: ChannelKindMatch, ID: channelID}, Audience{
				Kind:    AudienceUsers,
				UserIDs: recipients,
			}, evt)
		}
	}

	ackPayload := map[string]any{"version": version, "changed": true}
	_ = h.writeAccepted(ctx, conn, channelKind, channelID, cmd.CommandID, version, ackPayload)
	if raw, mErr := json.Marshal(mustAcceptedEvent(channelKind, channelID, cmd.CommandID, version, ackPayload)); mErr == nil {
		_ = h.commands.SaveCommandAck(ctx, matchID, membership.UserID, cmd.CommandID, raw)
	}
	h.metrics.ObserveCommand(cmd.Type, "accepted")
}

func (h *MatchHandler) handleSpectateSelect(
	ctx context.Context,
	conn *websocket.Conn,
	membership ChannelMembership,
	matchID uuid.UUID,
	channelKind, channelID string,
	cmd ClientCommand,
) {
	if h.commands == nil {
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeFeatureDisabled, "commands unavailable", 0)
		h.metrics.ObserveCommand(cmd.Type, "rejected")
		return
	}
	var body SpectateSelectPayload
	if len(cmd.Payload) > 0 {
		if err := json.Unmarshal(cmd.Payload, &body); err != nil {
			_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeCommandInvalid, "invalid spectate payload", 0)
			h.metrics.ObserveCommand(cmd.Type, "rejected")
			return
		}
	}
	preferred := uuid.Nil
	if strings.TrimSpace(body.TargetGamePlayerID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(body.TargetGamePlayerID))
		if err != nil {
			// Privacy-safe: invalid target looks like forbidden, not a parse leak.
			_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, CodeSpectateForbidden, "you cannot spectate that player right now", 0)
			h.metrics.ObserveCommand(cmd.Type, "rejected")
			return
		}
		preferred = id
	}

	result, err := h.commands.SelectSpectate(ctx, matchID, membership.UserID, preferred)
	if err != nil {
		code, msg, outcome := mapCommandError(err)
		_ = h.writePrivateError(ctx, conn, channelKind, channelID, cmd.CommandID, code, msg, 0)
		h.metrics.ObserveCommand(cmd.Type, outcome)
		return
	}

	allowed := make([]string, 0, len(result.AllowedTargetIDs))
	for _, id := range result.AllowedTargetIDs {
		allowed = append(allowed, id.String())
	}
	payload := map[string]any{
		"allowed_spectate_player_ids": allowed,
		"fallback":                    result.Fallback,
	}
	if result.SelectedTargetID != uuid.Nil {
		payload["target_game_player_id"] = result.SelectedTargetID.String()
	} else {
		payload["target_game_player_id"] = nil
	}
	if result.RoundID != uuid.Nil {
		payload["round_id"] = result.RoundID.String()
	}

	// Private spectate_target_changed to the requesting spectator only.
	var roundPtr *uuid.UUID
	if result.RoundID != uuid.Nil {
		rid := result.RoundID
		roundPtr = &rid
	}
	evt, evtErr := NewChannelEvent(
		uuid.NewString(),
		EventRoundSpectateTargetChanged,
		ChannelKindMatch,
		channelID,
		nil,
		roundPtr,
		time.Now().UTC(),
		result.Version,
		payload,
	)
	if evtErr == nil {
		publisher := ChannelPublisher(HubChannelPublisher{Hub: h.hub})
		if h.fanout != nil {
			publisher = h.fanout
		}
		_ = publisher.Publish(ctx, ChannelRef{Kind: ChannelKindMatch, ID: channelID}, Audience{
			Kind:    AudienceUsers,
			UserIDs: []uuid.UUID{membership.UserID},
		}, evt)
	}

	ackPayload := map[string]any{
		"version":                     result.Version,
		"target_game_player_id":       payload["target_game_player_id"],
		"allowed_spectate_player_ids": allowed,
		"fallback":                    result.Fallback,
	}
	_ = h.writeAccepted(ctx, conn, channelKind, channelID, cmd.CommandID, result.Version, ackPayload)
	if raw, mErr := json.Marshal(mustAcceptedEvent(channelKind, channelID, cmd.CommandID, result.Version, ackPayload)); mErr == nil {
		_ = h.commands.SaveCommandAck(ctx, matchID, membership.UserID, cmd.CommandID, raw)
	}
	h.metrics.ObserveCommand(cmd.Type, "accepted")
}

func (h *MatchHandler) writeAccepted(
	ctx context.Context,
	conn *websocket.Conn,
	channelKind, channelID, commandID string,
	version int64,
	payload any,
) error {
	evt := mustAcceptedEvent(channelKind, channelID, commandID, version, payload)
	return writeEvent(ctx, conn, evt)
}

func mustAcceptedEvent(channelKind, channelID, commandID string, version int64, payload any) Event {
	if payload == nil {
		payload = map[string]any{"command_id": commandID}
	}
	// Embed command_id for client correlation.
	body := map[string]any{"command_id": commandID}
	if m, ok := payload.(map[string]any); ok {
		for k, v := range m {
			body[k] = v
		}
	}
	evt, err := NewChannelEvent(uuid.NewString(), EventCommandAccepted, channelKind, channelID, nil, nil, time.Now().UTC(), version, body)
	if err != nil {
		raw, _ := json.Marshal(body)
		return Event{
			ProtocolVersion: ProtocolVersion,
			EventID:         uuid.NewString(),
			Type:            EventCommandAccepted,
			ChannelKind:     channelKind,
			ChannelID:       channelID,
			OccurredAt:      time.Now().UTC(),
			Version:         version,
			Payload:         raw,
		}
	}
	return evt
}

func (h *MatchHandler) writePrivateError(
	ctx context.Context,
	conn *websocket.Conn,
	channelKind, channelID, commandID, code, message string,
	version int64,
) error {
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	if commandID != "" {
		payload["command_id"] = commandID
	}
	evt, err := NewChannelEvent(uuid.NewString(), EventChannelError, channelKind, channelID, nil, nil, time.Now().UTC(), version, payload)
	if err != nil {
		return err
	}
	return writeEvent(ctx, conn, evt)
}

func writeEvent(ctx context.Context, conn *websocket.Conn, evt Event) error {
	if evt.ProtocolVersion == 0 {
		evt.ProtocolVersion = ProtocolVersion
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, raw)
}

func mapCommandError(err error) (code, message, outcome string) {
	switch {
	case errors.Is(err, ErrCommandThrottled), errors.Is(err, errThrottled):
		return CodeCommandThrottled, "command rate limited", "throttled"
	case errors.Is(err, ErrPlayerLocked), errors.Is(err, errLocked):
		return CodePlayerLocked, "guess is locked", "locked"
	case errors.Is(err, ErrSpectateForbidden), errors.Is(err, errSpectateForbidden):
		// Privacy-safe: never explain which policy denied the target.
		return CodeSpectateForbidden, "you cannot spectate that player right now", "rejected"
	case errors.Is(err, ErrVersionGap):
		return CodeVersionGap, "version gap; fetch HTTP snapshot", "version_gap"
	default:
		// Map platform redis sentinel strings via error text when wrapped.
		msg := err.Error()
		if strings.Contains(msg, "rate limited") || strings.Contains(msg, "throttled") {
			return CodeCommandThrottled, "command rate limited", "throttled"
		}
		if strings.Contains(msg, "locked") {
			return CodePlayerLocked, "guess is locked", "locked"
		}
		if strings.Contains(msg, "spectate") {
			return CodeSpectateForbidden, "you cannot spectate that player right now", "rejected"
		}
		return CodeCommandInvalid, "command failed", "error"
	}
}

// Sentinel helpers so command adapters can wrap platform errors without importing redis here.
var (
	errThrottled         = errors.New("throttled")
	errLocked            = errors.New("locked")
	errSpectateForbidden = errors.New("spectate forbidden")
)

// WrapThrottleError marks a throttle rejection for mapCommandError.
func WrapThrottleError(err error) error {
	if err == nil {
		return nil
	}
	return errors.Join(errThrottled, err)
}

// WrapLockedError marks a locked-player rejection for mapCommandError.
func WrapLockedError(err error) error {
	if err == nil {
		return nil
	}
	return errors.Join(errLocked, err)
}

// WrapSpectateForbidden marks a privacy-safe spectate denial for mapCommandError.
func WrapSpectateForbidden(err error) error {
	if err == nil {
		return ErrSpectateForbidden
	}
	return errors.Join(ErrSpectateForbidden, errSpectateForbidden, err)
}

// WrapViewUnchanged marks a non-material view update.
func WrapViewUnchanged(err error) error {
	if err == nil {
		return ErrViewUnchanged
	}
	return errors.Join(ErrViewUnchanged, err)
}

func containsForbiddenViewFields(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	forbidden := []string{
		"latitude", "longitude", "marker", "markers", "map", "cursor",
		"guess", "guesses", "score", "answer", "actual_location",
	}
	for _, key := range forbidden {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func originAllowed(origin string, patterns []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "*" || p == origin {
			return true
		}
		// Support suffix wildcards like http://localhost:*
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
	}
	return false
}
