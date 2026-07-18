package matchplay

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

// Match channel event type constants (aligned with realtime package).
const (
	EventMatchSnapshot              = realtime.EventMatchSnapshot
	EventMatchStarted               = realtime.EventMatchStarted
	EventMatchPlayerDisconnected    = realtime.EventMatchPlayerDisconnected
	EventMatchPlayerReconnected     = realtime.EventMatchPlayerReconnected
	EventMatchPlayerForfeited       = realtime.EventMatchPlayerForfeited
	EventRoundStarted               = realtime.EventRoundStarted
	EventRoundEnded                 = realtime.EventRoundEnded
	EventRoundResultsRevealed       = realtime.EventRoundResultsRevealed
	EventRoundMarkerChanged         = realtime.EventRoundMarkerChanged
	EventRoundSpectateTargetChanged = realtime.EventRoundSpectateTargetChanged
	EventRoundViewChanged           = realtime.EventRoundViewChanged
	EventChatMessageCreated         = realtime.EventChatMessageCreated
	EventChatMessageRemoved         = realtime.EventChatMessageRemoved
	EventChatMuteChanged            = realtime.EventChatMuteChanged
	EventMatchCompleted             = realtime.EventMatchCompleted
	EventMatchRoundClosed           = realtime.EventMatchRoundClosed
)

// Publisher adapts realtime.ChannelPublisher to the matchplay EventSink surface.
// Events are versioned and audience-filtered server-side (all participants by default).
// Call PublishAfterCommit helpers only after durable PostgreSQL commits succeed.
type Publisher struct {
	inner   realtime.ChannelPublisher
	logger  *slog.Logger
	metrics MetricsRecorder
	now     func() time.Time
}

// NewPublisher constructs a match event publisher.
func NewPublisher(inner realtime.ChannelPublisher, logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{
		inner:   inner,
		logger:  logger,
		metrics: NoopMetrics{},
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// WithMetrics attaches publish metrics.
func (p *Publisher) WithMetrics(m MetricsRecorder) *Publisher {
	if p == nil {
		return p
	}
	if m == nil {
		m = NoopMetrics{}
	}
	p.metrics = m
	return p
}

// PublishMatchEvent implements EventSink (all participants).
func (p *Publisher) PublishMatchEvent(
	ctx context.Context,
	matchID uuid.UUID,
	eventType string,
	version int64,
	gameID *uuid.UUID,
	roundID *uuid.UUID,
	payload any,
) error {
	return p.PublishToAudience(ctx, matchID, eventType, version, gameID, roundID, ResolveMatchLifecycleAudience(), payload)
}

// PublishToAudience publishes a versioned match event to a server-resolved audience.
// Must be called only after the durable commit that the event describes.
func (p *Publisher) PublishToAudience(
	ctx context.Context,
	matchID uuid.UUID,
	eventType string,
	version int64,
	gameID *uuid.UUID,
	roundID *uuid.UUID,
	audience realtime.Audience,
	payload any,
) error {
	if p == nil || p.inner == nil {
		return nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if audience.Kind == "" {
		audience = ResolveMatchLifecycleAudience()
	}
	eventID := uuid.NewString()
	evt, err := realtime.NewChannelEvent(
		eventID,
		eventType,
		realtime.ChannelKindMatch,
		matchID.String(),
		gameID,
		roundID,
		p.now(),
		version,
		payload,
	)
	if err != nil {
		p.logger.WarnContext(ctx, "match event build failed",
			slog.String("type", eventType),
			slog.String("match_id", matchID.String()),
			slog.Any("error", err),
		)
		p.observePublish(eventType, audience.Kind, "error")
		return err
	}
	channel := realtime.ChannelRef{Kind: realtime.ChannelKindMatch, ID: matchID.String()}
	if err := p.inner.Publish(ctx, channel, audience, evt); err != nil {
		p.logger.WarnContext(ctx, "match event publish failed",
			slog.String("type", eventType),
			slog.String("match_id", matchID.String()),
			slog.Any("error", err),
		)
		p.observePublish(eventType, audience.Kind, "error")
		return err
	}
	p.observePublish(eventType, audience.Kind, "ok")
	return nil
}

// PublishMarkerChanged publishes a team-only marker event after live-state commit.
func (p *Publisher) PublishMarkerChanged(
	ctx context.Context,
	matchID, roundID uuid.UUID,
	teamSlot int,
	version int64,
	payload any,
) error {
	rid := roundID
	return p.PublishToAudience(ctx, matchID, EventRoundMarkerChanged, version, nil, &rid, ResolveMarkerAudience(teamSlot), payload)
}

// PublishChatMessageCreated publishes a team chat event after durable message commit.
// eligibleRecipients should already exclude recipients who muted the author.
func (p *Publisher) PublishChatMessageCreated(
	ctx context.Context,
	matchID uuid.UUID,
	teamSlot int,
	version int64,
	eligibleRecipients []uuid.UUID,
	payload any,
) error {
	return p.PublishToAudience(ctx, matchID, EventChatMessageCreated, version, nil, nil, ResolveChatAudience(teamSlot, eligibleRecipients), payload)
}

// PublishViewChanged publishes provider-safe view state to authorized spectators only.
func (p *Publisher) PublishViewChanged(
	ctx context.Context,
	matchID, roundID uuid.UUID,
	version int64,
	spectatorUserIDs []uuid.UUID,
	payload any,
) error {
	rid := roundID
	return p.PublishToAudience(ctx, matchID, EventRoundViewChanged, version, nil, &rid, ResolveViewAudience(spectatorUserIDs), payload)
}

// PublishSpectateTargetChanged notifies a single spectator of their resolved target
// (preferred or automatic fallback). Payload must not include private guesses/scores.
func (p *Publisher) PublishSpectateTargetChanged(
	ctx context.Context,
	matchID, roundID uuid.UUID,
	version int64,
	spectatorUserID uuid.UUID,
	payload any,
) error {
	if spectatorUserID == uuid.Nil {
		return nil
	}
	rid := roundID
	return p.PublishToAudience(
		ctx,
		matchID,
		EventRoundSpectateTargetChanged,
		version,
		nil,
		&rid,
		ResolveViewAudience([]uuid.UUID{spectatorUserID}),
		payload,
	)
}

func (p *Publisher) observePublish(eventType, audienceKind, outcome string) {
	if p == nil || p.metrics == nil {
		return
	}
	p.metrics.ObserveEventPublish(eventType, audienceKind, outcome)
}

// PublishRoundResults publishes a versioned round.results_revealed event to all participants.
// Payload must already be privacy-safe (only call after round reveal).
func (p *Publisher) PublishRoundResults(
	ctx context.Context,
	matchID, gameID, roundID uuid.UUID,
	version int64,
	result RevealedRoundResult,
) error {
	gid, rid := gameID, roundID
	return p.PublishMatchEvent(ctx, matchID, EventRoundResultsRevealed, version, &gid, &rid, result)
}

// PublishMatchStarted publishes match.started to all participants.
func (p *Publisher) PublishMatchStarted(ctx context.Context, matchID, gameID uuid.UUID, version int64, payload any) error {
	gid := gameID
	return p.PublishMatchEvent(ctx, matchID, EventMatchStarted, version, &gid, nil, payload)
}

// PublishMatchCompleted publishes match.completed to all participants.
func (p *Publisher) PublishMatchCompleted(ctx context.Context, matchID, gameID uuid.UUID, version int64, payload any) error {
	gid := gameID
	return p.PublishMatchEvent(ctx, matchID, EventMatchCompleted, version, &gid, nil, payload)
}

// Party event type stubs for when the parties package is not yet present.
// Prefer parties.Event* once backend/internal/parties exists (T027–T031 / residual T042).
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

// PartyPublisher is a thin helper for party-channel events via the shared ChannelPublisher.
// Used as a residual stub until parties.events.go owns this surface.
type PartyPublisher struct {
	inner  realtime.ChannelPublisher
	logger *slog.Logger
	now    func() time.Time
}

// NewPartyPublisher constructs a party event publisher stub.
func NewPartyPublisher(inner realtime.ChannelPublisher, logger *slog.Logger) *PartyPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &PartyPublisher{
		inner:  inner,
		logger: logger,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// PublishPartyEvent delivers a versioned party event to active party members.
func (p *PartyPublisher) PublishPartyEvent(
	ctx context.Context,
	partyID uuid.UUID,
	eventType string,
	version int64,
	payload any,
) error {
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
		return err
	}
	channel := realtime.ChannelRef{Kind: realtime.ChannelKindParty, ID: partyID.String()}
	return p.inner.Publish(ctx, channel, realtime.Audience{Kind: realtime.AudienceAllParticipants}, evt)
}
