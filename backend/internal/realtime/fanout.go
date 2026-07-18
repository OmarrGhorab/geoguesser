package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
)

// PubSubTransport is the multi-instance fanout surface (Redis Pub/Sub adapter).
type PubSubTransport interface {
	Subscribe(ctx context.Context, channelKind, channelID string) error
	Unsubscribe(ctx context.Context, channelKind, channelID string) error
	Publish(ctx context.Context, channelKind, channelID string, payload []byte) error
}

// FanoutEnvelope is the inter-instance Pub/Sub payload. Audience is never
// forwarded to WebSocket clients; only local hubs apply it.
type FanoutEnvelope struct {
	Event    Event    `json:"event"`
	Audience Audience `json:"audience"`
}

// FanoutPublisher publishes to local hub subscribers and multi-instance Pub/Sub.
// Local delivery uses the hub immediately; remote instances receive via Pub/Sub
// and apply the same audience filter. Event_id dedupe is owned by Pub/Sub transport.
type FanoutPublisher struct {
	Hub    *Hub
	PubSub PubSubTransport
	Logger *slog.Logger
}

// NewFanoutPublisher constructs a multi-instance aware ChannelPublisher.
func NewFanoutPublisher(hub *Hub, pubsub PubSubTransport, logger *slog.Logger) *FanoutPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &FanoutPublisher{Hub: hub, PubSub: pubsub, Logger: logger}
}

// Publish implements ChannelPublisher.
func (p *FanoutPublisher) Publish(ctx context.Context, channel ChannelRef, audience Audience, envelope Event) error {
	if p == nil {
		return nil
	}
	// When Pub/Sub is configured, deliver only via Redis so each instance
	// (including the publisher) applies audience filtering once through
	// HandlePubSubMessage. This avoids local+echo double delivery.
	// Single-process/test setups without Pub/Sub fall back to the local hub.
	if p.PubSub == nil {
		if p.Hub != nil {
			return HubChannelPublisher{Hub: p.Hub}.Publish(ctx, channel, audience, envelope)
		}
		return nil
	}
	body, err := json.Marshal(FanoutEnvelope{Event: envelope, Audience: audience})
	if err != nil {
		return err
	}
	if err := p.PubSub.Publish(ctx, channel.Kind, channel.ID, body); err != nil {
		p.Logger.WarnContext(ctx, "realtime pubsub publish failed",
			slog.String("channel_kind", channel.Kind),
			slog.String("channel_id", channel.ID),
			slog.Any("error", err),
		)
		// Best-effort local fallback so in-process clients still see the event
		// when Redis Pub/Sub is unavailable.
		if p.Hub != nil {
			_ = HubChannelPublisher{Hub: p.Hub}.Publish(ctx, channel, audience, envelope)
		}
		return err
	}
	return nil
}

// HandlePubSubMessage is the MessageHandler for RealtimePubSub. It fans out to
// the local hub with audience filtering. Callers should register this once.
func (p *FanoutPublisher) HandlePubSubMessage(channelKind, channelID string, payload []byte) {
	if p == nil || p.Hub == nil || len(payload) == 0 {
		return
	}
	var env FanoutEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		// Accept bare Event payloads for simpler publishers/tests.
		var evt Event
		if err2 := json.Unmarshal(payload, &evt); err2 != nil {
			return
		}
		env.Event = evt
		env.Audience = Audience{Kind: AudienceAllParticipants}
	}
	if env.Event.EventID == "" {
		return
	}
	// Prefer envelope channel fields when present.
	kind := env.Event.ChannelKind
	id := env.Event.ChannelID
	if kind == "" {
		kind = channelKind
	}
	if id == "" {
		id = channelID
	}
	channel := ChannelRef{Kind: kind, ID: id}
	_ = HubChannelPublisher{Hub: p.Hub}.Publish(context.Background(), channel, env.Audience, env.Event)
}

// SubscribeChannel increments Pub/Sub refcount when the first local client joins.
func (p *FanoutPublisher) SubscribeChannel(ctx context.Context, channelKind, channelID string) error {
	if p == nil || p.PubSub == nil {
		return nil
	}
	return p.PubSub.Subscribe(ctx, channelKind, channelID)
}

// UnsubscribeChannel decrements Pub/Sub refcount when the last local client leaves.
func (p *FanoutPublisher) UnsubscribeChannel(ctx context.Context, channelKind, channelID string) error {
	if p == nil || p.PubSub == nil {
		return nil
	}
	return p.PubSub.Unsubscribe(ctx, channelKind, channelID)
}
