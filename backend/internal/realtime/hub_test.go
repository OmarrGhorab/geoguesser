package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubPartyAndMatchChannelsAreIsolated(t *testing.T) {
	t.Parallel()

	hub := NewHubWithQueueSize(8)
	ctx := context.Background()

	partyID := uuid.New().String()
	matchID := uuid.New().String()

	partyClient := NewClient(ChannelKindParty, partyID, uuid.New(), nil, 8)
	matchClient := NewClient(ChannelKindMatch, matchID, uuid.New(), nil, 8)
	hub.Subscribe(partyClient)
	hub.Subscribe(matchClient)

	partyEvt, err := NewChannelEvent("p1", EventPartySnapshot, ChannelKindParty, partyID, nil, nil, time.Now().UTC(), 1, map[string]any{"n": 1})
	if err != nil {
		t.Fatalf("party event: %v", err)
	}
	matchEvt, err := NewChannelEvent("m1", EventMatchSnapshot, ChannelKindMatch, matchID, nil, nil, time.Now().UTC(), 1, map[string]any{"n": 1})
	if err != nil {
		t.Fatalf("match event: %v", err)
	}

	hub.Publish(ctx, ChannelKey{Kind: ChannelKindParty, ID: partyID}, partyEvt)
	hub.Publish(ctx, ChannelKey{Kind: ChannelKindMatch, ID: matchID}, matchEvt)

	select {
	case got := <-partyClient.Send:
		if got.EventID != "p1" {
			t.Fatalf("party got %q", got.EventID)
		}
	case <-time.After(time.Second):
		t.Fatal("party client did not receive event")
	}
	select {
	case got := <-matchClient.Send:
		if got.EventID != "m1" {
			t.Fatalf("match got %q", got.EventID)
		}
	case <-time.After(time.Second):
		t.Fatal("match client did not receive event")
	}

	// Cross-channel silence.
	select {
	case got := <-partyClient.Send:
		t.Fatalf("party received unexpected extra event %q", got.EventID)
	default:
	}
	select {
	case got := <-matchClient.Send:
		t.Fatalf("match received unexpected extra event %q", got.EventID)
	default:
	}
}

func TestHubPublishToUsers(t *testing.T) {
	t.Parallel()

	hub := NewHubWithQueueSize(4)
	ctx := context.Background()
	matchID := uuid.New().String()
	userA := uuid.New()
	userB := uuid.New()
	userC := uuid.New()

	clientA := NewClient(ChannelKindMatch, matchID, userA, nil, 4)
	clientB := NewClient(ChannelKindMatch, matchID, userB, nil, 4)
	clientC := NewClient(ChannelKindMatch, matchID, userC, nil, 4)
	hub.Subscribe(clientA)
	hub.Subscribe(clientB)
	hub.Subscribe(clientC)

	evt, err := NewChannelEvent("u1", EventMatchPlayerDisconnected, ChannelKindMatch, matchID, nil, nil, time.Now().UTC(), 2, map[string]any{"user": userA.String()})
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	hub.PublishToUsers(ctx, ChannelKey{Kind: ChannelKindMatch, ID: matchID}, []uuid.UUID{userA, userB}, evt)

	assertReceived(t, clientA, "u1")
	assertReceived(t, clientB, "u1")
	assertNotReceived(t, clientC)
}

func TestHubPublishToTeam(t *testing.T) {
	t.Parallel()

	hub := NewHubWithQueueSize(4)
	ctx := context.Background()
	matchID := uuid.New().String()
	team1 := 1
	team2 := 2

	ally := NewClient(ChannelKindMatch, matchID, uuid.New(), &team1, 4)
	ally2 := NewClient(ChannelKindMatch, matchID, uuid.New(), &team1, 4)
	foe := NewClient(ChannelKindMatch, matchID, uuid.New(), &team2, 4)
	hub.Subscribe(ally)
	hub.Subscribe(ally2)
	hub.Subscribe(foe)

	evt, err := NewChannelEvent("t1", EventRoundMarkerChanged, ChannelKindMatch, matchID, nil, nil, time.Now().UTC(), 3, map[string]any{"lat": 1.0})
	if err != nil {
		t.Fatalf("event: %v", err)
	}
	hub.PublishToTeam(ctx, ChannelKey{Kind: ChannelKindMatch, ID: matchID}, 1, evt)

	assertReceived(t, ally, "t1")
	assertReceived(t, ally2, "t1")
	assertNotReceived(t, foe)
}

func TestHubBoundedQueueAndSlowConsumerClose(t *testing.T) {
	t.Parallel()

	const queueSize = 2
	hub := NewHubWithQueueSize(queueSize)
	ctx := context.Background()
	partyID := uuid.New().String()

	slow := NewClient(ChannelKindParty, partyID, uuid.New(), nil, queueSize)
	fast := NewClient(ChannelKindParty, partyID, uuid.New(), nil, queueSize)
	hub.Subscribe(slow)
	hub.Subscribe(fast)

	// Fill the slow client's queue without reading.
	for i := 0; i < queueSize; i++ {
		evt, err := NewChannelEvent("fill", EventPartySnapshot, ChannelKindParty, partyID, nil, nil, time.Now().UTC(), int64(i+1), map[string]any{"i": i})
		if err != nil {
			t.Fatalf("event: %v", err)
		}
		// Deliver only to slow by user filter so fast stays empty.
		hub.PublishToUsers(ctx, ChannelKey{Kind: ChannelKindParty, ID: partyID}, []uuid.UUID{slow.UserID}, evt)
	}
	if len(slow.Send) != queueSize {
		t.Fatalf("slow queue len = %d, want %d", len(slow.Send), queueSize)
	}

	// Next publish should drop the slow consumer and still deliver to others.
	overflow, err := NewChannelEvent("overflow", EventPartyReadinessChanged, ChannelKindParty, partyID, nil, nil, time.Now().UTC(), 99, map[string]any{"ready": true})
	if err != nil {
		t.Fatalf("overflow event: %v", err)
	}
	hub.Publish(ctx, ChannelKey{Kind: ChannelKindParty, ID: partyID}, overflow)

	select {
	case <-slow.Done():
	case <-time.After(time.Second):
		t.Fatal("slow consumer was not closed")
	}
	if !slow.IsClosed() {
		t.Fatal("slow.IsClosed() = false, want true")
	}
	if hub.SubscriberCount(ChannelKey{Kind: ChannelKindParty, ID: partyID}) != 1 {
		t.Fatalf("subscriber count = %d, want 1", hub.SubscriberCount(ChannelKey{Kind: ChannelKindParty, ID: partyID}))
	}
	assertReceived(t, fast, "overflow")
}

func TestHubVersionOrderingNonDecreasing(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	ctx := context.Background()
	channel := ChannelKey{Kind: ChannelKindMatch, ID: uuid.New().String()}

	publish := func(version int64, id string) {
		t.Helper()
		evt, err := NewChannelEvent(id, EventMatchSnapshot, channel.Kind, channel.ID, nil, nil, time.Now().UTC(), version, map[string]any{"v": version})
		if err != nil {
			t.Fatalf("event %s: %v", id, err)
		}
		hub.Publish(ctx, channel, evt)
	}

	publish(1, "v1")
	if got := hub.LastVersion(channel); got != 1 {
		t.Fatalf("LastVersion = %d, want 1", got)
	}
	publish(3, "v3")
	if got := hub.LastVersion(channel); got != 3 {
		t.Fatalf("LastVersion = %d, want 3", got)
	}
	// Out-of-order older version must not decrease the stored watermark.
	publish(2, "v2")
	if got := hub.LastVersion(channel); got != 3 {
		t.Fatalf("LastVersion after older publish = %d, want 3", got)
	}
	publish(5, "v5")
	if got := hub.LastVersion(channel); got != 5 {
		t.Fatalf("LastVersion = %d, want 5", got)
	}
}

func TestHubLegacyRoomAddBroadcastRemove(t *testing.T) {
	t.Parallel()

	hub := NewHubWithQueueSize(4)
	ctx := context.Background()

	client := &Client{
		RoomCode: "ROOM01",
		Send:     make(chan Event, 4),
	}
	hub.Add(client)

	evt, err := NewEvent("r1", EventRoomPlayerJoined, "ROOM01", nil, time.Now().UTC(), 1, map[string]any{"joined": true})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	hub.Broadcast(ctx, "ROOM01", evt)
	assertReceived(t, client, "r1")

	hub.Remove(client)
	if hub.SubscriberCount(ChannelKey{Kind: ChannelKindRoom, ID: "ROOM01"}) != 0 {
		t.Fatal("expected no room subscribers after Remove")
	}
}

func assertReceived(t *testing.T, client *Client, eventID string) {
	t.Helper()
	select {
	case got := <-client.Send:
		if got.EventID != eventID {
			t.Fatalf("got event_id %q, want %q", got.EventID, eventID)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for event %q", eventID)
	}
}

func assertNotReceived(t *testing.T, client *Client) {
	t.Helper()
	select {
	case got := <-client.Send:
		t.Fatalf("unexpected event %q", got.EventID)
	default:
	}
}
