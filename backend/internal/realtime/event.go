package realtime

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

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
)

type Event struct {
	EventID    string          `json:"event_id"`
	Type       string          `json:"type"`
	RoomCode   string          `json:"room_code"`
	GameID     *uuid.UUID      `json:"game_id,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
	Version    int64           `json:"version"`
	Payload    json.RawMessage `json:"payload"`
}

func NewEvent(eventID, eventType, roomCode string, gameID *uuid.UUID, occurredAt time.Time, version int64, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	event := Event{EventID: eventID, Type: eventType, RoomCode: roomCode, GameID: gameID, OccurredAt: occurredAt, Version: version, Payload: raw}
	return event, event.Validate()
}

func (e Event) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if !IsKnownEventType(e.Type) {
		return errors.New("unknown event type")
	}
	if e.RoomCode == "" {
		return errors.New("room_code is required")
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
	return nil
}

func IsKnownEventType(eventType string) bool {
	switch eventType {
	case EventRoomSnapshot, EventRoomPlayerJoined, EventRoomPlayerLeft, EventRoomPlayerDisconnected, EventRoomPlayerReconnected, EventRoomPlayerRemoved, EventRoomSettingsUpdated, EventRoomReadyUpdated, EventRoomReadyReset, EventRoomStarted, EventRoundStarted, EventRoundGuessCountChanged, EventRoundEnded, EventRoundResultsRevealed, EventGameCompleted, EventRoomError:
		return true
	default:
		return false
	}
}
