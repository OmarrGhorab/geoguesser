package redis

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRealtimePubSubChannelName(t *testing.T) {
	got := RealtimePubSubChannel("party", "abc")
	if got != "realtime:v1:pubsub:party:abc" {
		t.Fatalf("channel = %q", got)
	}
	got = RealtimePubSubChannel("match", "00000000-0000-0000-0000-000000000001")
	if got != "realtime:v1:pubsub:match:00000000-0000-0000-0000-000000000001" {
		t.Fatalf("channel = %q", got)
	}
}

func TestRealtimePubSubRefcountSubscribeUnsubscribe(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	ps := NewRealtimePubSub(client, nil)
	t.Cleanup(func() {
		_ = ps.Shutdown(context.Background())
	})

	channelID := uuid.NewString()
	if err := ps.Subscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("first subscribe: %v", err)
	}
	if got := ps.RefCount(ChannelKindParty, channelID); got != 1 {
		t.Fatalf("refcount after first = %d", got)
	}
	if err := ps.Subscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("second subscribe: %v", err)
	}
	if got := ps.RefCount(ChannelKindParty, channelID); got != 2 {
		t.Fatalf("refcount after second = %d", got)
	}

	if err := ps.Unsubscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("first unsubscribe: %v", err)
	}
	if got := ps.RefCount(ChannelKindParty, channelID); got != 1 {
		t.Fatalf("refcount after first unsub = %d", got)
	}
	if err := ps.Unsubscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("second unsubscribe: %v", err)
	}
	if got := ps.RefCount(ChannelKindParty, channelID); got != 0 {
		t.Fatalf("refcount after cleanup = %d", got)
	}
	// Extra unsubscribe is a no-op.
	if err := ps.Unsubscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("extra unsubscribe: %v", err)
	}
}

func TestRealtimePubSubFanoutDelivers(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	channelID := uuid.NewString()
	eventID := uuid.NewString()

	var (
		mu       sync.Mutex
		received []string
		wg       sync.WaitGroup
	)
	wg.Add(1)
	ps := NewRealtimePubSub(client, func(kind, id string, payload []byte) {
		mu.Lock()
		received = append(received, kind+"|"+id+"|"+string(payload))
		mu.Unlock()
		wg.Done()
	})
	t.Cleanup(func() {
		_ = ps.Shutdown(context.Background())
	})

	if err := ps.Subscribe(ctx, ChannelKindMatch, channelID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	// Allow Redis subscription to propagate.
	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(map[string]any{
		"event_id": eventID,
		"type":     "match.round_started",
	})
	if err := ps.Publish(ctx, ChannelKindMatch, channelID, payload); err != nil {
		t.Fatalf("publish: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for fanout delivery")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received count = %d, want 1; got %v", len(received), received)
	}
	if received[0] != ChannelKindMatch+"|"+channelID+"|"+string(payload) {
		t.Fatalf("unexpected payload: %q", received[0])
	}
}

func TestRealtimePubSubDuplicateSuppression(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	channelID := uuid.NewString()
	eventID := uuid.NewString()

	var calls atomic.Int64
	var first sync.WaitGroup
	first.Add(1)
	ps := NewRealtimePubSub(client, func(kind, id string, payload []byte) {
		if calls.Add(1) == 1 {
			first.Done()
		}
	})
	t.Cleanup(func() {
		_ = ps.Shutdown(context.Background())
	})

	if err := ps.Subscribe(ctx, ChannelKindParty, channelID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	payload, _ := json.Marshal(map[string]any{"event_id": eventID, "type": "party.roster_updated"})
	if err := ps.Publish(ctx, ChannelKindParty, channelID, payload); err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	// Wait for first delivery so the event_id is cached before the duplicate.
	done := make(chan struct{})
	go func() {
		first.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for first delivery")
	}

	if err := ps.Publish(ctx, ChannelKindParty, channelID, payload); err != nil {
		t.Fatalf("publish 2: %v", err)
	}
	// Brief window for a potential second delivery that must not occur.
	time.Sleep(200 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1 (duplicate suppressed)", got)
	}

	// A different event_id must still be delivered.
	var second sync.WaitGroup
	second.Add(1)
	// Replace handler observation via a second pubsub is hard; re-publish with new id
	// and rely on call count.
	otherID := uuid.NewString()
	otherPayload, _ := json.Marshal(map[string]any{"event_id": otherID, "type": "party.roster_updated"})
	// Bump expected: next call should fire.
	prev := calls.Load()
	if err := ps.Publish(ctx, ChannelKindParty, channelID, otherPayload); err != nil {
		t.Fatalf("publish other: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() == prev+1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("handler calls = %d, want %d for new event_id", calls.Load(), prev+1)
}

func TestRealtimePubSubShutdown(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	channelID := uuid.NewString()

	var calls atomic.Int64
	ps := NewRealtimePubSub(client, func(kind, id string, payload []byte) {
		calls.Add(1)
	})
	if err := ps.Subscribe(ctx, ChannelKindMatch, channelID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if got := ps.RefCount(ChannelKindMatch, channelID); got != 1 {
		t.Fatalf("refcount before shutdown = %d", got)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ps.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if got := ps.RefCount(ChannelKindMatch, channelID); got != 0 {
		t.Fatalf("refcount after shutdown = %d", got)
	}

	// Second shutdown is a no-op.
	if err := ps.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}

	// Subscribe after shutdown must fail.
	if err := ps.Subscribe(ctx, ChannelKindMatch, channelID); err == nil {
		t.Fatal("expected subscribe after shutdown to fail")
	}

	// Publish still works at Redis level, but no local handler should run.
	payload, _ := json.Marshal(map[string]any{"event_id": uuid.NewString()})
	_ = ps.Publish(ctx, ChannelKindMatch, channelID, payload)
	time.Sleep(150 * time.Millisecond)
	if got := calls.Load(); got != 0 {
		t.Fatalf("handler calls after shutdown = %d, want 0", got)
	}
}

func TestParseRealtimePubSubChannel(t *testing.T) {
	kind, id, ok := parseRealtimePubSubChannel("realtime:v1:pubsub:party:abc")
	if !ok || kind != "party" || id != "abc" {
		t.Fatalf("got kind=%q id=%q ok=%v", kind, id, ok)
	}
	if _, _, ok := parseRealtimePubSubChannel("other:channel"); ok {
		t.Fatal("expected non-prefix channel to fail")
	}
	if _, _, ok := parseRealtimePubSubChannel("realtime:v1:pubsub:nocolon"); ok {
		t.Fatal("expected missing separator to fail")
	}
}

func TestExtractEventID(t *testing.T) {
	if got := extractEventID([]byte(`{"event_id":"e1","type":"x"}`)); got != "e1" {
		t.Fatalf("event id = %q", got)
	}
	if got := extractEventID([]byte(`{"event":{"event_id":"nested-e1","type":"x"}}`)); got != "nested-e1" {
		t.Fatalf("nested event id = %q", got)
	}
	if got := extractEventID([]byte(`not-json`)); got != "" {
		t.Fatalf("non-json event id = %q", got)
	}
	if got := extractEventID(nil); got != "" {
		t.Fatalf("nil payload event id = %q", got)
	}
}
