package matchplay

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store is the durable matchplay persistence surface.
// Implementations may be PostgreSQL-backed or in-memory fakes for unit tests.
type Store interface {
	// LoadSnapshotBundle returns the raw data needed for an authorized snapshot.
	// Returns nil bundle (no error) when the match does not exist.
	LoadSnapshotBundle(ctx context.Context, matchID uuid.UUID) (*SnapshotBundle, error)

	// LoadRoundResultBundle returns revealed round data. round must be completed.
	// Returns ErrRoundNotRevealed when the round is still active.
	// Returns nil,nil when match/round missing.
	LoadRoundResultBundle(ctx context.Context, matchID, roundID uuid.UUID) (*RoundResultBundle, error)

	// LoadTerminalResultBundle returns the terminal match outcome.
	// Returns nil,nil when missing; ErrMatchNotActive when not terminal.
	LoadTerminalResultBundle(ctx context.Context, matchID uuid.UUID) (*TerminalResultBundle, error)

	// FindParticipant returns the caller's participant row, or nil when not on the match.
	FindParticipant(ctx context.Context, matchID, userID uuid.UUID) (*MatchParticipant, error)

	// ExplicitLeaveTx marks the caller abandoned and forfeits the match for Casual/Ranked.
	// Idempotent when already terminal or the caller already abandoned.
	ExplicitLeaveTx(ctx context.Context, matchID, userID uuid.UUID, now time.Time) (*LeaveOutcome, error)

	// ForfeitDisconnectTx forfeits a single participant past reconnect grace.
	// Idempotent when already terminal or already abandoned.
	ForfeitDisconnectTx(ctx context.Context, matchID, userID uuid.UUID, now time.Time) (*LeaveOutcome, error)

	// CloseInactiveCasualTx ends a casual match with no activity past the inactivity window.
	CloseInactiveCasualTx(ctx context.Context, matchID uuid.UUID, now time.Time) (*LeaveOutcome, error)

	// ListDisconnectCandidates returns active participants marked disconnected past grace.
	ListDisconnectCandidates(ctx context.Context, now time.Time, grace time.Duration, limit int) ([]DisconnectCandidate, error)

	// ListInactiveCasualMatches returns active casual matches with stale last_activity_at.
	ListInactiveCasualMatches(ctx context.Context, cutoff time.Time, limit int) ([]InactivityCandidate, error)

	// TouchActivity updates last_activity_at for a match (gameplay/collab activity).
	TouchActivity(ctx context.Context, matchID uuid.UUID, at time.Time) error
}

// connectionStateStore is an optional durable extension implemented by the
// PostgreSQL repository. Keeping it separate preserves narrow Store test fakes.
type connectionStateStore interface {
	MarkConnectionState(ctx context.Context, matchID, userID uuid.UUID, connected bool, at time.Time) (changed bool, err error)
}

// PartyRestorer restores party state after a terminal match so members can requeue.
// Implementations live in the parties package; matchplay only depends on this surface.
// When nil, party restoration is skipped (solo or parties not yet wired).
type PartyRestorer interface {
	// RestoreAfterTerminalMatch moves parties linked to the match from in_match
	// back to forming and clears active_match_id. Must be idempotent.
	RestoreAfterTerminalMatch(ctx context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error
}

// VersionStore provides monotonic match-channel versions.
type VersionStore interface {
	NextVersion(ctx context.Context, matchID uuid.UUID) (int64, error)
	CurrentVersion(ctx context.Context, matchID uuid.UUID) (int64, error)
}

// PresenceGraceStore tracks reconnect grace windows for disconnect sweeps.
// Implementations typically wrap platform/redis RealtimeStore.
type PresenceGraceStore interface {
	// HasReconnectWindow reports whether a reconnect grace key still exists.
	HasReconnectWindow(ctx context.Context, matchID, userID uuid.UUID) (bool, error)
	// ClearPresence removes presence and reconnect state after forfeit.
	ClearPresence(ctx context.Context, matchID, userID uuid.UUID) error
}

// EventSink publishes versioned match lifecycle events after durable commits.
// Implementations wrap realtime.ChannelPublisher without creating package cycles.
type EventSink interface {
	PublishMatchEvent(ctx context.Context, matchID uuid.UUID, eventType string, version int64, gameID *uuid.UUID, roundID *uuid.UUID, payload any) error
}

// MediaResolver turns provider/provider_ref into safe media DTOs for current rounds.
type MediaResolver interface {
	ResolveRoundMedia(provider, providerRef string, attribution *string) *RoundMediaDTO
}

// TxRunner is an optional seam for repository methods that need an external transaction.
// Not required by the Store interface; exposed for advanced adapters.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error
}
