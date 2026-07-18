package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEventRoomCompatibility(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	event, err := NewEvent("evt-1", EventRoomSnapshot, "ABC123", &gameID, time.Now().UTC(), 3, map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if event.RoomCode != "ABC123" {
		t.Fatalf("RoomCode = %q, want ABC123", event.RoomCode)
	}
	if event.ChannelKind != ChannelKindRoom || event.ChannelID != "ABC123" {
		t.Fatalf("channel = %s/%s, want room/ABC123", event.ChannelKind, event.ChannelID)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded["room_code"] != "ABC123" {
		t.Fatalf("json room_code = %v", decoded["room_code"])
	}
	if decoded["event_id"] != "evt-1" || decoded["type"] != EventRoomSnapshot {
		t.Fatalf("json identity = %v", decoded)
	}
}

func TestNewChannelEventPartyAndMatch(t *testing.T) {
	t.Parallel()

	partyID := uuid.New().String()
	matchID := uuid.New().String()
	gameID := uuid.New()
	roundID := uuid.New()
	now := time.Now().UTC()

	partyEvt, err := NewChannelEvent("evt-party", EventPartyRosterUpdated, ChannelKindParty, partyID, nil, nil, now, 1, map[string]any{"members": 2})
	if err != nil {
		t.Fatalf("NewChannelEvent party: %v", err)
	}
	if partyEvt.RoomCode != "" {
		t.Fatalf("party RoomCode should be empty, got %q", partyEvt.RoomCode)
	}
	if partyEvt.ChannelKind != ChannelKindParty || partyEvt.ChannelID != partyID {
		t.Fatalf("party channel = %s/%s", partyEvt.ChannelKind, partyEvt.ChannelID)
	}

	matchEvt, err := NewChannelEvent("evt-match", EventMatchRoundClosed, ChannelKindMatch, matchID, &gameID, &roundID, now, 7, map[string]any{"closed": true})
	if err != nil {
		t.Fatalf("NewChannelEvent match: %v", err)
	}
	if matchEvt.ChannelKind != ChannelKindMatch || matchEvt.ChannelID != matchID {
		t.Fatalf("match channel = %s/%s", matchEvt.ChannelKind, matchEvt.ChannelID)
	}
	if matchEvt.GameID == nil || *matchEvt.GameID != gameID {
		t.Fatalf("GameID = %v, want %v", matchEvt.GameID, gameID)
	}
	if matchEvt.RoundID == nil || *matchEvt.RoundID != roundID {
		t.Fatalf("RoundID = %v, want %v", matchEvt.RoundID, roundID)
	}

	// Party/match stubs are known without requiring full payload types.
	for _, typ := range []string{
		EventPartySnapshot, EventPartyMemberJoined, EventPartyMemberLeft, EventPartyMemberRemoved,
		EventPartyLeaderChanged, EventPartyReadinessChanged, EventPartyQueueChanged, EventPartyMatchAssigned,
		EventPartyRosterUpdated, EventPartyClosed, EventMatchSnapshot, EventMatchStarted, EventMatchPlayerDisconnected,
		EventMatchPlayerReconnected, EventMatchPlayerForfeited, EventRoundGuessLocked, EventRoundMarkerChanged,
		EventMatchCompleted, EventMatchRoundClosed, EventChannelError, EventCompetitiveRatingApplied,
	} {
		if !IsKnownEventType(typ) {
			t.Fatalf("expected known event type %q", typ)
		}
	}
}

func TestEventValidateRequiresTarget(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{"x": 1})

	missing := Event{
		EventID:    "e1",
		Type:       EventPartySnapshot,
		OccurredAt: now,
		Version:    1,
		Payload:    payload,
	}
	if err := missing.Validate(); err == nil {
		t.Fatal("expected validation error without room_code or channel")
	}

	legacyRoom := Event{
		EventID:    "e2",
		Type:       EventRoomSnapshot,
		RoomCode:   "XYZ789",
		OccurredAt: now,
		Version:    1,
		Payload:    payload,
	}
	if err := legacyRoom.Validate(); err != nil {
		t.Fatalf("legacy room event: %v", err)
	}

	invalidKind := Event{
		EventID:     "e3",
		Type:        EventMatchSnapshot,
		ChannelKind: "lobby",
		ChannelID:   "id",
		OccurredAt:  now,
		Version:     1,
		Payload:     payload,
	}
	if err := invalidKind.Validate(); err == nil {
		t.Fatal("expected invalid channel_kind error")
	}

	if IsKnownEventType("not.a.real.event") {
		t.Fatal("unknown type should not be known")
	}
}

func TestEventJSONOmitsEmptyRoomCodeForParty(t *testing.T) {
	t.Parallel()

	partyID := uuid.New().String()
	evt, err := NewChannelEvent("e", EventPartyQueueChanged, ChannelKindParty, partyID, nil, nil, time.Now().UTC(), 2, map[string]any{"state": "searching"})
	if err != nil {
		t.Fatalf("NewChannelEvent: %v", err)
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := decoded["room_code"]; ok {
		t.Fatalf("room_code should be omitted for party events, got %v", decoded["room_code"])
	}
	if decoded["channel_kind"] != ChannelKindParty || decoded["channel_id"] != partyID {
		t.Fatalf("channel fields = %v", decoded)
	}
}
