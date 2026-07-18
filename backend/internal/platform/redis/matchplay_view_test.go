package redis

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchLiveViewProviderSafeSchemaAndForbiddenFields(t *testing.T) {
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

	// Safe schema accepted.
	rec, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano-abc",
		Heading:    120,
		Pitch:      -5,
		Zoom:       1.5,
	}, DefaultRoundLiveTTL)
	if err != nil {
		t.Fatalf("safe set: %v", err)
	}
	if rec.Version < 1 || rec.PanoramaID != "pano-abc" {
		t.Fatalf("record = %+v", rec)
	}

	// Stored JSON must not include private fields.
	raw, err := client.HGet(ctx, matchplayViewsKey(matchID, roundID), userID.String()).Result()
	if err != nil {
		t.Fatalf("hget: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("decode stored: %v", err)
	}
	for _, key := range []string{"latitude", "longitude", "marker", "map", "cursor", "guess", "score", "answer"} {
		if _, ok := stored[key]; ok {
			t.Fatalf("stored scene contains forbidden key %q: %v", key, stored)
		}
	}
	if stored["panorama_id"] != "pano-abc" {
		t.Fatalf("stored = %v", stored)
	}

	// Explicit map/guess/answer rejection via SetViewFromMap.
	for _, fields := range []map[string]any{
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "latitude": 10.0},
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "marker": map[string]any{"lat": 1}},
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "guess": true},
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "answer": "secret"},
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "score": 5000},
		{"heading": 1.0, "pitch": 0.0, "zoom": 1.0, "private_client_field": true},
	} {
		_, err := store.SetViewFromMap(ctx, matchID, roundID, userID, fields, DefaultRoundLiveTTL)
		if !errors.Is(err, ErrViewForbiddenFields) {
			t.Fatalf("fields=%v err=%v want ErrViewForbiddenFields", fields, err)
		}
	}
}

func TestMatchLiveViewThrottleFourPerSecond(t *testing.T) {
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
			Heading:    float64(i * 10), // material each time
			Pitch:      1,
			Zoom:       2,
		}, DefaultRoundLiveTTL); err != nil {
			t.Fatalf("view %d: %v", i, err)
		}
	}
	_, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano",
		Heading:    350,
		Pitch:      1,
		Zoom:       2,
	}, DefaultRoundLiveTTL)
	if !errors.Is(err, ErrViewThrottled) {
		t.Fatalf("err = %v, want ErrViewThrottled", err)
	}
}

func TestMatchLiveViewMaterialChangeDedup(t *testing.T) {
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

	first, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano",
		Heading:    10,
		Pitch:      0,
		Zoom:       1,
	}, DefaultRoundLiveTTL)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	// Sub-threshold jitter must not bump version.
	unchanged, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano",
		Heading:    10.2,
		Pitch:      0.1,
		Zoom:       1.01,
	}, DefaultRoundLiveTTL)
	if !errors.Is(err, ErrViewUnchanged) {
		t.Fatalf("err = %v, want ErrViewUnchanged", err)
	}
	if unchanged.Version != first.Version {
		t.Fatalf("version changed on jitter: %d -> %d", first.Version, unchanged.Version)
	}

	// Material heading change stores and bumps version.
	second, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano",
		Heading:    20,
		Pitch:      0,
		Zoom:       1,
	}, DefaultRoundLiveTTL)
	if err != nil {
		t.Fatalf("material: %v", err)
	}
	if second.Version <= first.Version {
		t.Fatalf("version not bumped: first=%d second=%d", first.Version, second.Version)
	}
}

func TestMatchLiveViewRoundTTL(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewMatchLiveStore(client)

	matchID := uuid.NewString()
	roundID := uuid.NewString()
	userID := uuid.New()
	ttl := 2 * time.Second
	t.Cleanup(func() {
		_ = store.DeleteRoundKeys(ctx, matchID, roundID)
		_ = client.Del(ctx, matchplayVersionKey(matchID), matchplayThrottleKey(matchID, "view", userID)).Err()
	})

	if _, err := store.SetView(ctx, matchID, roundID, userID, ViewRecord{
		PanoramaID: "pano", Heading: 1, Pitch: 0, Zoom: 1,
	}, ttl); err != nil {
		t.Fatalf("set: %v", err)
	}

	pttl, err := client.PTTL(ctx, matchplayViewsKey(matchID, roundID)).Result()
	if err != nil {
		t.Fatalf("pttl: %v", err)
	}
	if pttl <= 0 || pttl > ttl {
		t.Fatalf("views key TTL = %v, want (0, %v]", pttl, ttl)
	}

	// ExpireRoundKeys extends TTL on views.
	if err := store.ExpireRoundKeys(ctx, matchID, roundID, 5*time.Second); err != nil {
		t.Fatalf("expire: %v", err)
	}
	pttl, err = client.PTTL(ctx, matchplayViewsKey(matchID, roundID)).Result()
	if err != nil {
		t.Fatalf("pttl2: %v", err)
	}
	if pttl < 2*time.Second {
		t.Fatalf("extended TTL too small: %v", pttl)
	}
}

func TestValidateViewFieldsTable(t *testing.T) {
	t.Parallel()
	if err := ValidateViewFields(map[string]any{
		"panorama_id": "x", "heading": 1.0, "pitch": 0.0, "zoom": 1.0,
	}); err != nil {
		t.Fatalf("safe: %v", err)
	}
	if err := ValidateViewFields(map[string]any{"latitude": 1.0}); !errors.Is(err, ErrViewForbiddenFields) {
		t.Fatalf("lat: %v", err)
	}
	if !ViewMateriallyChanged(
		ViewRecord{Heading: 1},
		ViewRecord{Heading: 2},
	) {
		t.Fatal("expected material heading change")
	}
}
