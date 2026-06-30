package redis

import (
	"testing"

	"github.com/google/uuid"
)

func TestRoomRedisKeys(t *testing.T) {
	playerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := map[string]string{
		roomSnapshotKey("ABCD12"):            "rooms:ABCD12:snapshot",
		roomVersionKey("ABCD12"):             "rooms:ABCD12:version",
		roomPresenceKey("ABCD12", playerID):  "rooms:ABCD12:presence:00000000-0000-0000-0000-000000000001",
		roomReconnectKey("ABCD12", playerID): "rooms:ABCD12:reconnect:00000000-0000-0000-0000-000000000001",
		roomReadyKey("ABCD12"):               "rooms:ABCD12:ready",
		roomLockKey("ABCD12", "start"):       "rooms:ABCD12:lock:start",
	}

	for got, want := range tests {
		if got != want {
			t.Fatalf("key = %q, want %q", got, want)
		}
	}
}
