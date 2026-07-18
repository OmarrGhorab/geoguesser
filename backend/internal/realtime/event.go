package realtime

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ChannelKindRoom is the legacy private-room transport kind.
// ChannelKindParty and ChannelKindMatch are defined in contracts.go.
const ChannelKindRoom = "room"

const (
	EventRoomSnapshot           = "room.snapshot"
	EventRoomPlayerJoined       = "room.player_joined"
	EventRoomPlayerLeft         = "room.player_left"
	EventRoomPlayerDisconnected = "room.player_disconnected"
	EventRoomPlayerReconnected  = "room.player_reconnected"
	EventRoomPlayerRemoved      = "room.player_removed"
	EventRoomSettingsUpdated    = "room.settings_updated"
	EventRoomReadyUpdated       = "room.ready_updated"
	EventRoomReadyReset         = "room.ready_reset"
	EventRoomStarted            = "room.started"
	EventRoundStarted           = "round.started"
	EventRoundGuessCountChanged = "round.guess_count_changed"
	EventRoundEnded             = "round.ended"
	EventRoundResultsRevealed   = "round.results_revealed"
	EventGameCompleted          = "game.completed"
	EventRoomError              = "room.error"

	// Party channel event types (payload shapes filled in later stories).
	EventPartySnapshot         = "party.snapshot"
	EventPartyMemberJoined     = "party.member_joined"
	EventPartyMemberLeft       = "party.member_left"
	EventPartyMemberRemoved    = "party.member_removed"
	EventPartyLeaderChanged    = "party.leader_changed"
	EventPartyReadinessChanged = "party.readiness_changed"
	EventPartyQueueChanged     = "party.queue_changed"
	EventPartyMatchAssigned    = "party.match_assigned"
	EventPartyRosterUpdated    = "party.roster_updated"
	EventPartyClosed           = "party.closed"

	// Match channel event types (payload shapes filled in later stories).
	EventMatchSnapshot              = "match.snapshot"
	EventMatchStarted               = "match.started"
	EventMatchPlayerDisconnected    = "match.player_disconnected"
	EventMatchPlayerReconnected     = "match.player_reconnected"
	EventMatchPlayerForfeited       = "match.player_forfeited"
	EventRoundGuessLocked           = "round.guess_locked"
	EventRoundMarkerChanged         = "round.marker_changed"
	EventRoundSpectateTargetChanged = "round.spectate_target_changed"
	EventRoundViewChanged           = "round.view_changed"
	EventChatMessageCreated         = "chat.message_created"
	EventChatMessageRemoved         = "chat.message_removed"
	EventChatMuteChanged            = "chat.mute_changed"
	EventMatchCompleted             = "match.completed"
	EventMatchRoundClosed           = "match.round_closed"
	EventCompetitiveRatingApplied   = "competitive.rating_applied"
	EventChannelError               = "channel.error"
	EventCommandAccepted            = "command.accepted"

	// Client command types (WebSocket inbound).
	CommandPresenceHeartbeat   = "presence.heartbeat"
	CommandRoundMarkerSet      = "round.marker.set"
	CommandRoundViewUpdate     = "round.view.update"
	CommandRoundSpectateSelect = "round.spectate.select"

	// ProtocolVersion is the supported WebSocket protocol major version.
	ProtocolVersion = 1
	// SubprotocolV1 is the negotiated application subprotocol.
	SubprotocolV1 = "geoguess.v1"
	// TicketSubprotocolPrefix prefixes the opaque one-time ticket in Sec-WebSocket-Protocol.
	TicketSubprotocolPrefix = "ticket."
)

// Event is the server-to-client realtime envelope.
// Room clients keep reading room_code; party/match clients use channel_kind + channel_id.
// Audience is never part of the public payload — the hub resolves targets server-side.
type Event struct {
	ProtocolVersion int             `json:"protocol_version,omitempty"`
	EventID         string          `json:"event_id"`
	Type            string          `json:"type"`
	RoomCode        string          `json:"room_code,omitempty"`
	ChannelKind     string          `json:"channel_kind,omitempty"`
	ChannelID       string          `json:"channel_id,omitempty"`
	GameID          *uuid.UUID      `json:"game_id,omitempty"`
	RoundID         *uuid.UUID      `json:"round_id,omitempty"`
	OccurredAt      time.Time       `json:"occurred_at"`
	Version         int64           `json:"version"`
	Payload         json.RawMessage `json:"payload"`
}

// NewEvent builds a legacy room-targeted event. RoomCode remains populated for
// existing room websocket clients; channel fields are set for unified routing.
func NewEvent(eventID, eventType, roomCode string, gameID *uuid.UUID, occurredAt time.Time, version int64, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	event := Event{
		EventID:     eventID,
		Type:        eventType,
		RoomCode:    roomCode,
		ChannelKind: ChannelKindRoom,
		ChannelID:   roomCode,
		GameID:      gameID,
		OccurredAt:  occurredAt,
		Version:     version,
		Payload:     raw,
	}
	return event, event.Validate()
}

// NewChannelEvent builds an event for a party, match, or room channel.
func NewChannelEvent(eventID, eventType, channelKind, channelID string, gameID, roundID *uuid.UUID, occurredAt time.Time, version int64, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	event := Event{
		ProtocolVersion: ProtocolVersion,
		EventID:         eventID,
		Type:            eventType,
		ChannelKind:     channelKind,
		ChannelID:       channelID,
		GameID:          gameID,
		RoundID:         roundID,
		OccurredAt:      occurredAt,
		Version:         version,
		Payload:         raw,
	}
	if channelKind == ChannelKindRoom {
		event.RoomCode = channelID
	}
	return event, event.Validate()
}

// EventFromEnvelope converts a party/match ChannelEnvelope into the hub Event shape.
func EventFromEnvelope(env ChannelEnvelope) Event {
	return Event{
		EventID:     env.EventID,
		Type:        env.Type,
		ChannelKind: env.ChannelKind,
		ChannelID:   env.ChannelID,
		GameID:      env.GameID,
		RoundID:     env.RoundID,
		OccurredAt:  env.OccurredAt,
		Version:     env.Version,
		Payload:     env.Payload,
	}
}

// ToEnvelope converts a party/match Event into a ChannelEnvelope.
func (e Event) ToEnvelope() ChannelEnvelope {
	return ChannelEnvelope{
		EventID:     e.EventID,
		Type:        e.Type,
		ChannelKind: e.ChannelKind,
		ChannelID:   e.ChannelID,
		GameID:      e.GameID,
		RoundID:     e.RoundID,
		OccurredAt:  e.OccurredAt,
		Version:     e.Version,
		Payload:     e.Payload,
	}
}

func (e Event) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if !IsKnownEventType(e.Type) {
		return errors.New("unknown event type")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	if e.Version < 0 {
		return errors.New("version must be non-negative")
	}
	if len(e.Payload) == 0 {
		return errors.New("payload is required")
	}
	if err := e.validateTarget(); err != nil {
		return err
	}
	return nil
}

func (e Event) validateTarget() error {
	kind := e.ChannelKind
	id := e.ChannelID
	if kind == "" && id == "" && e.RoomCode != "" {
		// Legacy room-only envelope.
		return nil
	}
	if kind == "" && e.RoomCode != "" {
		kind = ChannelKindRoom
		if id == "" {
			id = e.RoomCode
		}
	}
	if kind == "" || id == "" {
		if e.RoomCode != "" {
			return nil
		}
		return errors.New("room_code or channel_kind+channel_id is required")
	}
	switch kind {
	case ChannelKindRoom, ChannelKindParty, ChannelKindMatch:
	default:
		return errors.New("invalid channel_kind")
	}
	if kind == ChannelKindRoom && e.RoomCode != "" && e.RoomCode != id {
		return errors.New("room_code must match channel_id for room channels")
	}
	return nil
}

// ResolveChannel returns the effective channel kind and id for hub routing.
func (e Event) ResolveChannel() (kind, id string) {
	if e.ChannelKind != "" && e.ChannelID != "" {
		return e.ChannelKind, e.ChannelID
	}
	if e.RoomCode != "" {
		return ChannelKindRoom, e.RoomCode
	}
	return e.ChannelKind, e.ChannelID
}

func IsKnownEventType(eventType string) bool {
	switch eventType {
	case EventRoomSnapshot, EventRoomPlayerJoined, EventRoomPlayerLeft, EventRoomPlayerDisconnected,
		EventRoomPlayerReconnected, EventRoomPlayerRemoved, EventRoomSettingsUpdated, EventRoomReadyUpdated,
		EventRoomReadyReset, EventRoomStarted, EventRoundStarted, EventRoundGuessCountChanged, EventRoundEnded,
		EventRoundResultsRevealed, EventGameCompleted, EventRoomError,
		EventPartySnapshot, EventPartyMemberJoined, EventPartyMemberLeft, EventPartyMemberRemoved,
		EventPartyLeaderChanged, EventPartyReadinessChanged, EventPartyQueueChanged, EventPartyMatchAssigned,
		EventPartyRosterUpdated, EventPartyClosed,
		EventMatchSnapshot, EventMatchStarted, EventMatchPlayerDisconnected, EventMatchPlayerReconnected,
		EventMatchPlayerForfeited, EventRoundGuessLocked, EventRoundMarkerChanged, EventRoundSpectateTargetChanged,
		EventRoundViewChanged, EventChatMessageCreated, EventChatMessageRemoved, EventChatMuteChanged,
		EventMatchCompleted, EventMatchRoundClosed, EventCompetitiveRatingApplied, EventChannelError,
		EventCommandAccepted:
		return true
	default:
		return false
	}
}

// IsKnownCommandType reports whether type is a supported client WebSocket command.
func IsKnownCommandType(commandType string) bool {
	switch commandType {
	case CommandPresenceHeartbeat, CommandRoundMarkerSet, CommandRoundViewUpdate, CommandRoundSpectateSelect:
		return true
	default:
		return false
	}
}
