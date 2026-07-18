package matchmaking

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PartyQueueSnapshot is a complete, queue-ready party view consumed by matchmaking.
// Owned fields are copied so matchmaking never depends on the parties package.
type PartyQueueSnapshot struct {
	PartyID   uuid.UUID
	Format    string
	Capacity  int
	Version   int64
	LeaderID  uuid.UUID
	Status    string
	MemberIDs []uuid.UUID
	AllReady  bool
}

// PartySnapshotReader loads a complete ready party for queue entry validation.
// Implementations live in the parties package; matchmaking only needs this surface.
type PartySnapshotReader interface {
	// ReadyPartyForQueue returns the caller's party when it is complete, all-ready,
	// leader-owned, and format-compatible with the requested mode. Returns nil when
	// no active party exists for the user.
	ReadyPartyForQueue(ctx context.Context, leaderUserID uuid.UUID, format string, expectedVersion int64) (*PartyQueueSnapshot, error)
}

// CompetitiveStandingSnapshot is the rating/placement view needed for ranked tickets.
type CompetitiveStandingSnapshot struct {
	UserID              uuid.UUID
	SeasonID            uuid.UUID
	Rating              int
	PlacementsCompleted int
	// RankCode is empty while placements are incomplete.
	RankCode string
	// NamedRankTier is a coarse tier index used for party spread checks (0 when hidden).
	NamedRankTier int
}

// CompetitiveStandingReader loads rating/placement facts for ranked ticket formation.
// Implementations live in the competitive package.
type CompetitiveStandingReader interface {
	// ActiveSeasonID returns the current active competitive season, or uuid.Nil when none.
	ActiveSeasonID(ctx context.Context) (uuid.UUID, error)
	// StandingsForUsers returns standings for the given users in the active season.
	// Missing standings are created lazily by the implementation when policy allows.
	StandingsForUsers(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID) ([]CompetitiveStandingSnapshot, error)
}

// MatchAssignmentEvent is a privacy-safe assignment notification for party/match channels.
type MatchAssignmentEvent struct {
	MatchID     uuid.UUID
	GameID      uuid.UUID
	Mode        string
	Playlist    string
	Format      string
	FormedAt    time.Time
	Destination string
	// UserIDs are the participants who must receive the assignment event.
	UserIDs []uuid.UUID
	// PartyIDs are optional party channels to notify (duo/squad).
	PartyIDs []uuid.UUID
}

// MatchFormationNotifier publishes post-commit assignment events after durable formation.
// Implementations typically wrap a realtime ChannelPublisher without creating package cycles.
type MatchFormationNotifier interface {
	NotifyMatchFormed(ctx context.Context, event MatchAssignmentEvent) error
}

// EventPublisher is a narrow post-commit event sink for matchmaking lifecycle signals.
// Kept smaller than a full realtime hub so matchmaking only depends on publish semantics.
type EventPublisher interface {
	// Publish delivers a typed event to a channel. channelKind is "party" or "match".
	// audienceUserIDs, when non-empty, restricts delivery to those users on the channel.
	Publish(ctx context.Context, channelKind, channelID, eventType string, version int64, audienceUserIDs []uuid.UUID, payload any) error
}
