package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Channel kinds for party/match transport. ChannelKindRoom is defined in event.go.
const (
	ChannelKindParty = "party"
	ChannelKindMatch = "match"
)

// Audience kinds control server-side fanout targeting. Audience is never sent to clients.
const (
	AudienceAllParticipants = "all_participants"
	AudienceTeam            = "team"
	AudienceUsers           = "users"
)

// Audience describes who may receive a published event on a channel.
type Audience struct {
	Kind     string      // all_participants | team | users
	TeamSlot *int        // required when Kind == AudienceTeam (1 or 2)
	UserIDs  []uuid.UUID // required when Kind == AudienceUsers; optional filter otherwise
}

// ChannelRef identifies a realtime channel without coupling to room-code semantics.
// Compatible with ChannelKey used by Hub routing.
type ChannelRef struct {
	Kind string // party | match | room
	ID   string // party UUID, match UUID, or room code
}

// Key converts the ref to the hub routing key.
func (c ChannelRef) Key() ChannelKey {
	return ChannelKey(c)
}

// ChannelRefFromKey builds a ChannelRef from a hub ChannelKey.
func ChannelRefFromKey(key ChannelKey) ChannelRef {
	return ChannelRef(key)
}

// ChannelEnvelope is the generalized party/match event shape used by publishers
// that do not need room_code. Convert to Event via EventFromEnvelope for hub fanout.
type ChannelEnvelope struct {
	EventID     string          `json:"event_id"`
	Type        string          `json:"type"`
	ChannelKind string          `json:"channel_kind"`
	ChannelID   string          `json:"channel_id"`
	GameID      *uuid.UUID      `json:"game_id,omitempty"`
	RoundID     *uuid.UUID      `json:"round_id,omitempty"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Version     int64           `json:"version"`
	Payload     json.RawMessage `json:"payload"`
}

// NewChannelEnvelope builds and validates a party/match envelope.
func NewChannelEnvelope(eventID, eventType, channelKind, channelID string, gameID, roundID *uuid.UUID, occurredAt time.Time, version int64, payload any) (ChannelEnvelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return ChannelEnvelope{}, err
	}
	env := ChannelEnvelope{
		EventID:     eventID,
		Type:        eventType,
		ChannelKind: channelKind,
		ChannelID:   channelID,
		GameID:      gameID,
		RoundID:     roundID,
		OccurredAt:  occurredAt,
		Version:     version,
		Payload:     raw,
	}
	return env, env.Validate()
}

// Validate checks required envelope fields for party/match channels.
func (e ChannelEnvelope) Validate() error {
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if e.Type == "" {
		return errors.New("type is required")
	}
	if e.ChannelKind != ChannelKindParty && e.ChannelKind != ChannelKindMatch {
		return errors.New("channel_kind is invalid")
	}
	if e.ChannelID == "" {
		return errors.New("channel_id is required")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("occurred_at is required")
	}
	if e.Version < 0 {
		return errors.New("version must be non-negative")
	}
	if len(e.Payload) == 0 {
		return errors.New("payload is required")
	}
	return nil
}

// IsChannelKind reports whether kind is a known channel kind (room, party, or match).
func IsChannelKind(kind string) bool {
	switch kind {
	case ChannelKindRoom, ChannelKindParty, ChannelKindMatch:
		return true
	default:
		return false
	}
}

// ChannelPublisher delivers versioned events to party, match, or room channels with audience targeting.
// Implementations may fan out locally (Hub) and/or via multi-instance Pub/Sub.
type ChannelPublisher interface {
	Publish(ctx context.Context, channel ChannelRef, audience Audience, envelope Event) error
}

// TicketClaims are the authorized binding for a one-time realtime connection ticket.
type TicketClaims struct {
	UserID      uuid.UUID
	ChannelKind string
	ChannelID   string
	ExpiresAt   time.Time
}

// TicketValidator issues and atomically consumes one-time realtime connection tickets.
// Implementations live under platform/redis; realtime only depends on this surface.
type TicketValidator interface {
	// Consume validates and invalidates a one-time ticket. Replay must fail.
	Consume(ctx context.Context, token string) (*TicketClaims, error)
}

// TicketIssuer creates one-time tickets bound to a user and channel.
type TicketIssuer interface {
	Issue(ctx context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (token string, expiresAt time.Time, err error)
}

// TicketStore combines issue and consume for HTTP + WebSocket ticket flows.
type TicketStore interface {
	TicketIssuer
	TicketValidator
}

// ChannelMembership is the authorized binding recovered after ticket consume
// and channel membership re-check.
type ChannelMembership struct {
	UserID   uuid.UUID
	TeamSlot *int // match only; party leaves nil
}

// ChannelAuthorizer re-checks durable membership before ticket issue and after consume.
// Missing and unauthorized resources must both return a privacy-safe not-found style error.
type ChannelAuthorizer interface {
	Authorize(ctx context.Context, userID uuid.UUID, channelKind, channelID string) (ChannelMembership, error)
}

// SnapshotProvider supplies the authorized initial channel snapshot after WS upgrade.
type SnapshotProvider interface {
	// Snapshot returns the caller-authorized payload and current channel version.
	Snapshot(ctx context.Context, userID uuid.UUID, channelKind, channelID string) (payload any, version int64, err error)
}

// SpectateSelectResult is the authorized outcome of round.spectate.select.
// Audience and target lists use game-player IDs for targets and never embed guesses.
type SpectateSelectResult struct {
	RoundID          uuid.UUID
	SelectedTargetID uuid.UUID // game_player_id; uuid.Nil when none available
	AllowedTargetIDs []uuid.UUID
	Fallback         bool
	Format           string
	Version          int64
}

// MatchCommandService applies match collaboration commands after WS validation.
type MatchCommandService interface {
	// SetMarker stores a team-scoped marker and returns the new version.
	SetMarker(ctx context.Context, matchID, roundID, userID uuid.UUID, teamSlot int, lat, lng float64) (version int64, err error)
	// SetView stores provider-safe view state and returns the new version.
	// Non-material updates may return ErrViewUnchanged-wrapped errors; handlers treat them as accepted no-ops.
	SetView(ctx context.Context, matchID, roundID, userID uuid.UUID, panoramaID string, heading, pitch, zoom float64) (version int64, err error)
	// SelectSpectate validates preferred target game_player_id and applies automatic fallback.
	SelectSpectate(ctx context.Context, matchID, viewerUserID, preferredTargetGamePlayerID uuid.UUID) (SpectateSelectResult, error)
	// ViewRecipients returns user IDs authorized to receive sourceUserID's view fanout for the round.
	ViewRecipients(ctx context.Context, matchID, roundID, sourceUserID uuid.UUID) ([]uuid.UUID, error)
	// Heartbeat refreshes presence for a connected participant.
	Heartbeat(ctx context.Context, matchID, userID uuid.UUID) error
	// CurrentVersion returns the match channel version.
	CurrentVersion(ctx context.Context, matchID uuid.UUID) (int64, error)
	// LoadCommandAck returns a prior ack for command replay when present.
	LoadCommandAck(ctx context.Context, matchID, userID uuid.UUID, commandID string) (ackJSON []byte, found bool, err error)
	// SaveCommandAck stores an ack for future replay.
	SaveCommandAck(ctx context.Context, matchID, userID uuid.UUID, commandID string, ackJSON []byte) error
}

// PresenceRecord is disposable connection presence for a channel member.
type PresenceRecord struct {
	UserID      uuid.UUID
	ChannelKind string
	ChannelID   string
	ConnectedAt time.Time
	LastSeenAt  time.Time
	ExpiresAt   time.Time
}

// PresenceStore tracks connection presence and reconnect grace windows.
type PresenceStore interface {
	Heartbeat(ctx context.Context, channel ChannelRef, userID uuid.UUID, now time.Time, ttl time.Duration) error
	IsPresent(ctx context.Context, channel ChannelRef, userID uuid.UUID) (bool, error)
	MarkDisconnected(ctx context.Context, channel ChannelRef, userID uuid.UUID, now time.Time, grace time.Duration) error
	Clear(ctx context.Context, channel ChannelRef, userID uuid.UUID) error
}

// ConnectionLifecycle records transport connection transitions after durable
// channel authorization. Implementations must be idempotent because a player
// may have more than one socket and cleanup can race reconnects.
type ConnectionLifecycle interface {
	Connected(ctx context.Context, channel ChannelRef, userID uuid.UUID, lastVersion int64) error
	Heartbeat(ctx context.Context, channel ChannelRef, userID uuid.UUID) error
	Disconnected(ctx context.Context, channel ChannelRef, userID uuid.UUID, lastVersion int64) error
}

// VersionStore provides monotonic channel versions for ordered fanout.
type VersionStore interface {
	// NextVersion increments and returns the channel version.
	NextVersion(ctx context.Context, channel ChannelRef) (int64, error)
	// CurrentVersion returns the latest version without incrementing.
	CurrentVersion(ctx context.Context, channel ChannelRef) (int64, error)
}

// HubChannelPublisher adapts Hub to ChannelPublisher.
type HubChannelPublisher struct {
	Hub *Hub
}

// Publish implements ChannelPublisher using hub audience helpers.
func (p HubChannelPublisher) Publish(ctx context.Context, channel ChannelRef, audience Audience, envelope Event) error {
	if p.Hub == nil {
		return nil
	}
	key := channel.Key()
	switch audience.Kind {
	case AudienceUsers:
		p.Hub.PublishToUsers(ctx, key, audience.UserIDs, envelope)
	case AudienceTeam:
		if audience.TeamSlot == nil {
			return nil
		}
		p.Hub.PublishToTeam(ctx, key, *audience.TeamSlot, envelope)
	default:
		// AudienceAllParticipants and empty kind broadcast to the full channel.
		if len(audience.UserIDs) > 0 {
			p.Hub.PublishToUsers(ctx, key, audience.UserIDs, envelope)
			return nil
		}
		p.Hub.Publish(ctx, key, envelope)
	}
	return nil
}
