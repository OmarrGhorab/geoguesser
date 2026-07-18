package realtime_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/realtime"
)

func TestNewChannelEventForPartyAndMatch(t *testing.T) {
	t.Parallel()

	gameID := uuid.New()
	env, err := realtime.NewChannelEvent(
		"evt-1",
		realtime.EventMatchStarted,
		realtime.ChannelKindMatch,
		uuid.NewString(),
		&gameID,
		nil,
		time.Now().UTC(),
		1,
		map[string]string{"status": "matched"},
	)
	if err != nil {
		t.Fatalf("NewChannelEvent: %v", err)
	}
	if env.ChannelKind != realtime.ChannelKindMatch || env.Version != 1 || len(env.Payload) == 0 {
		t.Fatalf("envelope = %+v", env)
	}

	_, err = realtime.NewChannelEvent("evt-2", realtime.EventRoomSnapshot, "lobby", "abc", nil, nil, time.Now().UTC(), 0, map[string]bool{"ok": true})
	if err == nil {
		t.Fatal("expected invalid channel_kind error")
	}
}

func TestIsChannelKind(t *testing.T) {
	t.Parallel()

	if !realtime.IsChannelKind(realtime.ChannelKindParty) || !realtime.IsChannelKind(realtime.ChannelKindMatch) {
		t.Fatal("party and match should be valid")
	}
	if !realtime.IsChannelKind(realtime.ChannelKindRoom) {
		t.Fatal("room should be a valid channel kind for hub routing")
	}
	if realtime.IsChannelKind("lobby") {
		t.Fatal("unknown kind should be invalid")
	}
}
