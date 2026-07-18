package parties

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

// Party channel event type constants (aligned with realtime package).
const (
	EventPartySnapshot         = realtime.EventPartySnapshot
	EventPartyMemberJoined     = realtime.EventPartyMemberJoined
	EventPartyMemberLeft       = realtime.EventPartyMemberLeft
	EventPartyMemberRemoved    = realtime.EventPartyMemberRemoved
	EventPartyLeaderChanged    = realtime.EventPartyLeaderChanged
	EventPartyReadinessChanged = realtime.EventPartyReadinessChanged
	EventPartyQueueChanged     = realtime.EventPartyQueueChanged
	EventPartyMatchAssigned    = realtime.EventPartyMatchAssigned
	EventPartyRosterUpdated    = realtime.EventPartyRosterUpdated
	EventPartyClosed           = realtime.EventPartyClosed
)

// EventPublisher adapts realtime.ChannelPublisher for versioned party events.
// Thin residual surface for T042: roster, readiness, queue, and assignment fanout.
type EventPublisher struct {
	inner  realtime.ChannelPublisher
	logger *slog.Logger
	now    func() time.Time
}

// NewEventPublisher constructs a party event publisher.
func NewEventPublisher(inner realtime.ChannelPublisher, logger *slog.Logger) *EventPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventPublisher{
		inner:  inner,
		logger: logger,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Publish delivers a versioned party-channel event to all active party participants.
func (p *EventPublisher) Publish(
	ctx context.Context,
	partyID uuid.UUID,
	eventType string,
	version int64,
	payload any,
) error {
	return p.publish(ctx, partyID, eventType, version, realtime.Audience{Kind: realtime.AudienceAllParticipants}, payload)
}

// PublishToUsers limits fanout to the durable active roster (plus any explicitly
// removed member that must receive the transition event).
func (p *EventPublisher) PublishToUsers(ctx context.Context, partyID uuid.UUID, eventType string, version int64, userIDs []uuid.UUID, payload any) error {
	return p.publish(ctx, partyID, eventType, version, realtime.Audience{Kind: realtime.AudienceUsers, UserIDs: userIDs}, payload)
}

func (p *EventPublisher) publish(ctx context.Context, partyID uuid.UUID, eventType string, version int64, audience realtime.Audience, payload any) error {
	if p == nil || p.inner == nil {
		return nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	evt, err := realtime.NewChannelEvent(
		uuid.NewString(),
		eventType,
		realtime.ChannelKindParty,
		partyID.String(),
		nil,
		nil,
		p.now(),
		version,
		payload,
	)
	if err != nil {
		p.logger.WarnContext(ctx, "party event build failed",
			slog.String("type", eventType),
			slog.String("party_id", partyID.String()),
			slog.Any("error", err),
		)
		return err
	}
	channel := realtime.ChannelRef{Kind: realtime.ChannelKindParty, ID: partyID.String()}
	if err := p.inner.Publish(ctx, channel, audience, evt); err != nil {
		p.logger.WarnContext(ctx, "party event publish failed",
			slog.String("type", eventType),
			slog.String("party_id", partyID.String()),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

// PublishRosterUpdated notifies members that the party roster changed.
func (p *EventPublisher) PublishRosterUpdated(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartyRosterUpdated, version, payload)
}

// PublishReadinessChanged notifies members that readiness state changed.
func (p *EventPublisher) PublishReadinessChanged(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartyReadinessChanged, version, payload)
}

// PublishQueueChanged notifies members that queue state changed.
func (p *EventPublisher) PublishQueueChanged(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartyQueueChanged, version, payload)
}

// PublishMatchAssigned notifies members that a match was assigned.
func (p *EventPublisher) PublishMatchAssigned(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartyMatchAssigned, version, payload)
}

// PublishSnapshot delivers a full party snapshot event (reconnect / version gap).
func (p *EventPublisher) PublishSnapshot(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartySnapshot, version, payload)
}

// PublishClosed notifies members that the party was closed/disbanded.
func (p *EventPublisher) PublishClosed(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return p.Publish(ctx, partyID, EventPartyClosed, version, payload)
}
