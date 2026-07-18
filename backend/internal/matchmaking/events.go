package matchmaking

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

// Queue / assignment event types published on party and match channels.
const (
	EventPartyQueueChanged  = realtime.EventPartyQueueChanged
	EventPartyMatchAssigned = realtime.EventPartyMatchAssigned
	EventMatchStarted       = realtime.EventMatchStarted
	EventMatchSnapshot      = realtime.EventMatchSnapshot
)

// EventPublisherAdapter adapts realtime.ChannelPublisher to matchmaking.EventPublisher
// and MatchFormationNotifier for versioned queue/assignment fanout.
type EventPublisherAdapter struct {
	inner  realtime.ChannelPublisher
	logger *slog.Logger
	now    func() time.Time
}

// NewEventPublisherAdapter constructs a shared publisher adapter.
func NewEventPublisherAdapter(inner realtime.ChannelPublisher, logger *slog.Logger) *EventPublisherAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventPublisherAdapter{
		inner:  inner,
		logger: logger,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Publish implements EventPublisher.
func (a *EventPublisherAdapter) Publish(
	ctx context.Context,
	channelKind, channelID, eventType string,
	version int64,
	audienceUserIDs []uuid.UUID,
	payload any,
) error {
	if a == nil || a.inner == nil {
		return nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	evt, err := realtime.NewChannelEvent(
		uuid.NewString(),
		eventType,
		channelKind,
		channelID,
		nil,
		nil,
		a.now(),
		version,
		payload,
	)
	if err != nil {
		a.logger.WarnContext(ctx, "matchmaking event build failed",
			slog.String("type", eventType),
			slog.String("channel_kind", channelKind),
			slog.Any("error", err),
		)
		return err
	}
	channel := realtime.ChannelRef{Kind: channelKind, ID: channelID}
	audience := realtime.Audience{Kind: realtime.AudienceAllParticipants}
	if len(audienceUserIDs) > 0 {
		audience = realtime.Audience{Kind: realtime.AudienceUsers, UserIDs: audienceUserIDs}
	}
	return a.inner.Publish(ctx, channel, audience, evt)
}

// NotifyMatchFormed implements MatchFormationNotifier.
// Publishes party.match_assigned on each party channel and a privacy-safe
// match.started on the match channel to all participants.
func (a *EventPublisherAdapter) NotifyMatchFormed(ctx context.Context, event MatchAssignmentEvent) error {
	if a == nil || a.inner == nil {
		return nil
	}
	payload := map[string]any{
		"match_id":    event.MatchID.String(),
		"game_id":     event.GameID.String(),
		"mode":        event.Mode,
		"playlist":    event.Playlist,
		"format":      event.Format,
		"formed_at":   event.FormedAt.UTC().Format(time.RFC3339Nano),
		"destination": event.Destination,
	}
	// Party channels: assignment notice for requeue/UI routing.
	for _, partyID := range event.PartyIDs {
		if partyID == uuid.Nil {
			continue
		}
		_ = a.Publish(ctx, realtime.ChannelKindParty, partyID.String(), EventPartyMatchAssigned, 0, nil, payload)
	}
	// Match channel: started for all participants.
	return a.Publish(ctx, realtime.ChannelKindMatch, event.MatchID.String(), EventMatchStarted, 0, event.UserIDs, payload)
}

// PublishQueueChanged notifies party members that queue state changed.
func (a *EventPublisherAdapter) PublishQueueChanged(ctx context.Context, partyID uuid.UUID, version int64, payload any) error {
	return a.Publish(ctx, realtime.ChannelKindParty, partyID.String(), EventPartyQueueChanged, version, nil, payload)
}
