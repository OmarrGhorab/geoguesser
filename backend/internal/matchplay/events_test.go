package matchplay_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/realtime"
)

type capturePublisher struct {
	events []realtime.Event
}

func (c *capturePublisher) Publish(_ context.Context, _ realtime.ChannelRef, _ realtime.Audience, envelope realtime.Event) error {
	c.events = append(c.events, envelope)
	return nil
}

func TestPublisherEmitsVersionedMatchEvents(t *testing.T) {
	t.Parallel()
	cap := &capturePublisher{}
	pub := matchplay.NewPublisher(cap, nil)
	matchID := uuid.New()
	gameID := uuid.New()

	if err := pub.PublishMatchCompleted(context.Background(), matchID, gameID, 7, map[string]any{
		"result": matchplay.MatchResultForfeit,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(cap.events) != 1 {
		t.Fatalf("events = %d", len(cap.events))
	}
	evt := cap.events[0]
	if evt.Type != matchplay.EventMatchCompleted {
		t.Fatalf("type = %s", evt.Type)
	}
	if evt.Version != 7 {
		t.Fatalf("version = %d", evt.Version)
	}
	if evt.ChannelKind != realtime.ChannelKindMatch || evt.ChannelID != matchID.String() {
		t.Fatalf("channel = %s/%s", evt.ChannelKind, evt.ChannelID)
	}
	if evt.OccurredAt.IsZero() {
		t.Fatal("occurred_at required")
	}
	if time.Since(evt.OccurredAt) > time.Minute {
		t.Fatal("occurred_at unreasonable")
	}
}

func TestPartyPublisherStub(t *testing.T) {
	t.Parallel()
	cap := &capturePublisher{}
	pub := matchplay.NewPartyPublisher(cap, nil)
	partyID := uuid.New()
	if err := pub.PublishPartyEvent(context.Background(), partyID, matchplay.EventPartyQueueChanged, 3, map[string]any{
		"status": "searching",
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(cap.events) != 1 || cap.events[0].Type != matchplay.EventPartyQueueChanged {
		t.Fatalf("events = %+v", cap.events)
	}
}
