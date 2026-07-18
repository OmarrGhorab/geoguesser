package redis

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchplayKeyBuildersNeverEmbedSecrets(t *testing.T) {
	matchID := "11111111-1111-1111-1111-111111111111"
	roundID := "22222222-2222-2222-2222-222222222222"
	userID := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	ticket := "opaque-one-time-ticket-value-never-in-keys"

	keys := []string{
		matchplayVersionKey(matchID),
		matchplayMarkerKey(matchID, roundID, 1),
		matchplayViewsKey(matchID, roundID),
		matchplayLockedKey(matchID, roundID),
		matchplayCommandKey(matchID, userID, "cmd-1"),
		matchplayThrottleKey(matchID, "marker", userID),
		matchPresenceKey(matchID, userID),
		matchReconnectKey(matchID, userID),
	}
	for _, key := range keys {
		if strings.Contains(key, ticket) {
			t.Fatalf("key embeds ticket: %s", key)
		}
		if !strings.HasPrefix(key, "matchplay:v1:") {
			t.Fatalf("unexpected key prefix: %s", key)
		}
		redacted := RedactedKeySample(key)
		if strings.Contains(redacted, matchID) || strings.Contains(redacted, userID.String()) {
			t.Fatalf("redacted key still contains identifiers: %s", redacted)
		}
	}
}

func TestMatchLiveMarkerTeamScopedAndVersioned(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userA := uuid.New()
	userB := uuid.New()
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx, matchplayVersionKey(matchID)).Err()
	})

	rec, err := store.SetMarker(ctx, matchID, roundID, 1, userA, 12.5, 45.5, DefaultRoundLiveTTL)
	if err != nil {
		t.Fatalf("set marker team1: %v", err)
	}
	if rec.Version < 1 {
		t.Fatalf("version = %d", rec.Version)
	}

	// Opposing team markers are stored under a different hash.
	if _, err := store.SetMarker(ctx, matchID, roundID, 2, userB, -10, 20, DefaultRoundLiveTTL); err != nil {
		t.Fatalf("set marker team2: %v", err)
	}

	team1, err := store.GetTeamMarkers(ctx, matchID, roundID, 1)
	if err != nil {
		t.Fatalf("get team1: %v", err)
	}
	if len(team1) != 1 || team1[0].UserID != userA {
		t.Fatalf("team1 markers = %+v", team1)
	}
	if team1[0].Latitude != 12.5 || team1[0].Longitude != 45.5 {
		t.Fatalf("coords = %+v", team1[0])
	}

	team2, err := store.GetTeamMarkers(ctx, matchID, roundID, 2)
	if err != nil {
		t.Fatalf("get team2: %v", err)
	}
	if len(team2) != 1 || team2[0].UserID != userB {
		t.Fatalf("team2 markers = %+v", team2)
	}

	// Snapshot recovery: version is monotonic across teams.
	ver, err := store.GetVersion(ctx, matchID)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if ver < 2 {
		t.Fatalf("version = %d, want >= 2", ver)
	}
}

func TestMatchLiveMarkerThrottleTwoPerSecond(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userID := uuid.New()
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx,
			matchplayVersionKey(matchID),
			matchplayThrottleKey(matchID, "marker", userID),
		).Err()
	})

	for i := 0; i < DefaultMarkerRateLimit; i++ {
		if _, err := store.SetMarker(ctx, matchID, roundID, 1, userID, float64(i), float64(i), DefaultRoundLiveTTL); err != nil {
			t.Fatalf("set %d: %v", i, err)
		}
	}
	_, err := store.SetMarker(ctx, matchID, roundID, 1, userID, 99, 99, DefaultRoundLiveTTL)
	if err != ErrMarkerThrottled {
		t.Fatalf("err = %v, want ErrMarkerThrottled", err)
	}
}

func TestMatchLiveLockedPlayerRejected(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userID := uuid.New()
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx, matchplayVersionKey(matchID)).Err()
	})

	if _, err := store.SetMarker(ctx, matchID, roundID, 1, userID, 1, 2, DefaultRoundLiveTTL); err != nil {
		t.Fatalf("initial set: %v", err)
	}
	if err := store.MarkGuessLocked(ctx, matchID, roundID, userID, DefaultRoundLiveTTL); err != nil {
		t.Fatalf("lock: %v", err)
	}
	locked, err := store.IsGuessLocked(ctx, matchID, roundID, userID)
	if err != nil || !locked {
		t.Fatalf("locked=%v err=%v", locked, err)
	}
	_, err = store.SetMarker(ctx, matchID, roundID, 1, userID, 3, 4, DefaultRoundLiveTTL)
	if err != ErrPlayerLocked {
		t.Fatalf("err = %v, want ErrPlayerLocked", err)
	}
}

func TestMatchLiveRoundTTL(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userID := uuid.New()
	ttl := 2 * time.Second
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx, matchplayVersionKey(matchID)).Err()
	})

	if _, err := store.SetMarker(ctx, matchID, roundID, 1, userID, 1, 1, ttl); err != nil {
		t.Fatalf("set: %v", err)
	}
	key := matchplayMarkerKey(matchID, roundID, 1)
	pttl, err := client.PTTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("pttl: %v", err)
	}
	if pttl <= 0 || pttl > ttl+time.Second {
		t.Fatalf("pttl = %v, want ~%v", pttl, ttl)
	}

	// ExpireRoundKeys extends TTL on existing keys.
	if err := store.ExpireRoundKeys(ctx, matchID, roundID, 5*time.Second); err != nil {
		t.Fatalf("expire: %v", err)
	}
	pttl2, err := client.PTTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("pttl2: %v", err)
	}
	if pttl2 < 2*time.Second {
		t.Fatalf("extended pttl = %v", pttl2)
	}
}

func TestMatchLiveSnapshotRecoveryMarkers(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	u1, u2 := uuid.New(), uuid.New()
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx, matchplayVersionKey(matchID)).Err()
	})

	if _, err := store.SetMarker(ctx, matchID, roundID, 1, u1, 10, 20, DefaultRoundLiveTTL); err != nil {
		t.Fatalf("u1: %v", err)
	}
	// Small pause so throttle does not block different users (per-user keys).
	if _, err := store.SetMarker(ctx, matchID, roundID, 1, u2, 30, 40, DefaultRoundLiveTTL); err != nil {
		t.Fatalf("u2: %v", err)
	}

	markers, err := store.GetTeamMarkers(ctx, matchID, roundID, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(markers) != 2 {
		t.Fatalf("markers = %d, want 2", len(markers))
	}
	byUser := map[uuid.UUID]MarkerRecord{}
	for _, m := range markers {
		byUser[m.UserID] = m
	}
	if byUser[u1].Latitude != 10 || byUser[u2].Longitude != 40 {
		t.Fatalf("recovery = %+v", byUser)
	}
}

func TestMatchLiveViewThrottleAndSafeFields(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userID := uuid.New()
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx,
			matchplayVersionKey(matchID),
			matchplayThrottleKey(matchID, "view", userID),
		).Err()
	})

	for i := 0; i < DefaultViewRateLimit; i++ {
		if _, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
			PanoramaID: "pano",
			Heading:    float64(i),
			Pitch:      1,
			Zoom:       2,
		}, DefaultRoundLiveTTL); err != nil {
			t.Fatalf("view %d: %v", i, err)
		}
	}
	_, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{Heading: 99}, DefaultRoundLiveTTL)
	if err != ErrViewThrottled {
		t.Fatalf("err = %v, want ErrViewThrottled", err)
	}

	// Clear throttle for read-back.
	_ = client.Del(ctx, matchplayThrottleKey(matchID, "view", userID)).Err()
	got, err := store.GetView(ctx, matchID, roundID, userID)
	if err != nil || got == nil {
		t.Fatalf("get view: %v %#v", err, got)
	}
	if got.PanoramaID != "pano" {
		t.Fatalf("panorama = %q", got.PanoramaID)
	}
}

func TestMatchLiveCommandAckReplay(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	userID := uuid.New()
	cmdID := uuid.NewString()
	key := matchplayCommandKey(matchID, userID, cmdID)
	t.Cleanup(func() { _ = client.Del(ctx, key).Err() })

	first, replay, err := store.StoreCommandAck(ctx, matchID, userID, CommandAck{
		CommandID: cmdID,
		OK:        true,
		Version:   5,
	}, DefaultCommandAckTTL)
	if err != nil || replay {
		t.Fatalf("first: err=%v replay=%v", err, replay)
	}
	if !first.OK || first.Version != 5 {
		t.Fatalf("first ack = %+v", first)
	}

	second, replay, err := store.StoreCommandAck(ctx, matchID, userID, CommandAck{
		CommandID: cmdID,
		OK:        false,
		Version:   99,
	}, DefaultCommandAckTTL)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !replay {
		t.Fatal("expected replay")
	}
	if second.Version != 5 || !second.OK {
		t.Fatalf("replay should return original ack: %+v", second)
	}
}

func TestMatchLivePresenceAndReconnect(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	userID := uuid.New()
	t.Cleanup(func() {
		_ = client.Del(ctx,
			matchPresenceKey(matchID, userID),
			matchReconnectKey(matchID, userID),
		).Err()
	})

	if err := store.SetPresence(ctx, matchID, userID, "connected", DefaultMatchPresenceTTL); err != nil {
		t.Fatalf("presence: %v", err)
	}
	status, err := store.GetPresence(ctx, matchID, userID)
	if err != nil || status != "connected" {
		t.Fatalf("status=%q err=%v", status, err)
	}
	if err := store.SetReconnectWindow(ctx, matchID, userID, 42, DefaultMatchPresenceTTL); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	ver, ok, err := store.GetReconnectWindow(ctx, matchID, userID)
	if err != nil || !ok || ver != 42 {
		t.Fatalf("reconnect ver=%d ok=%v err=%v", ver, ok, err)
	}
}
