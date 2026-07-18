package games

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MatchLifecycleHook advances durable match/competitive state when a multiplayer
// game transitions. Implementations live outside games (matchmaking/matchplay/
// competitive adapters) so games never imports those packages.
//
// Tx methods run inside the game write transaction and must roll back with it.
// Non-tx methods are best-effort post-commit paths.
type MatchLifecycleHook interface {
	OnMatchActive(ctx context.Context, gameID uuid.UUID, at time.Time) error
	OnGameCompleted(ctx context.Context, gameID uuid.UUID, at time.Time) error
	OnGameCancelled(ctx context.Context, gameID uuid.UUID, at time.Time, failureCode string) error
	ApplyMatchActiveInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
	ApplyGameCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
	ApplyGameCancelledInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error
}

// MultiplayerEventSink publishes post-commit round/match transition events.
type MultiplayerEventSink interface {
	PublishMultiplayerOutcome(ctx context.Context, gameID uuid.UUID, outcome MultiplayerGuessOutcome) error
}

// HostedRoomLifecycleHook keeps a hosted room's durable lifecycle in the same
// PostgreSQL transaction as its Party Lobby game. Realtime publication remains
// post-commit and reloadable from this authoritative state.
type HostedRoomLifecycleHook interface {
	ApplyRoomStartedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
	ApplyRoomCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error
}

// MultiplayerEventFanout delivers outcomes to independent room and match
// channels. Every sink is attempted so one degraded transport cannot suppress
// the other; joined errors remain observable to the caller.
type MultiplayerEventFanout []MultiplayerEventSink

func (f MultiplayerEventFanout) PublishMultiplayerOutcome(ctx context.Context, gameID uuid.UUID, outcome MultiplayerGuessOutcome) error {
	var errs []error
	for _, sink := range f {
		if sink == nil {
			continue
		}
		if err := sink.PublishMultiplayerOutcome(ctx, gameID, outcome); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// TerminalResultApplier is optionally implemented by MatchLifecycleHook adapters
// to receive computed team totals when a multiplayer game ends.
type TerminalResultApplier interface {
	ApplyTerminalResultInTx(ctx context.Context, tx *gorm.DB, result TerminalMatchResult) error
}

// TerminalMatchResult describes the game-side outcome passed to progression hooks.
type TerminalMatchResult struct {
	GameID          uuid.UUID
	Result          string // team_one_win | team_two_win | draw | forfeit | abandoned | cancelled
	WinnerTeamSlot  *int
	TeamOneScore    int
	TeamTwoScore    int
	CompletedAt     time.Time
	IsRanked        bool
	AbandonedUserID *uuid.UUID
	AbandonReason   string
}

// ProgressionFinalizer applies exact-once competitive rating changes after a terminal ranked game.
// Implementations live in the competitive package.
type ProgressionFinalizer interface {
	// FinalizeInTx applies rating changes inside the caller's transaction.
	// A unique-constraint conflict is treated as an idempotent replay.
	FinalizeInTx(ctx context.Context, tx *gorm.DB, result TerminalMatchResult) error
}

// RoundRevealPolicy decides whether multiplayer answer/guess details may leave the service.
// Default multiplayer behavior withholds answers until the shared round closes.
type RoundRevealPolicy interface {
	// MayRevealAnswer reports whether answer coordinates and full guess details
	// may be included in a submit/snapshot payload for the given mode and round status.
	MayRevealAnswer(mode, roundStatus string, roundCompleted bool) bool
}

// DelayedRevealPolicy is the standard multiplayer reveal policy: answers are hidden
// until the round is completed.
type DelayedRevealPolicy struct{}

// MayRevealAnswer implements RoundRevealPolicy.
func (DelayedRevealPolicy) MayRevealAnswer(mode, roundStatus string, roundCompleted bool) bool {
	if !IsMultiplayerMode(mode) {
		return true
	}
	if roundCompleted {
		return true
	}
	return roundStatus == RoundStatusCompleted
}

// CollaborationMarkerGate authorizes proposed-marker writes without games importing matchplay.
type CollaborationMarkerGate interface {
	// CanUpdateMarker reports whether the player may set a proposed marker for the round.
	CanUpdateMarker(ctx context.Context, gameID, roundID, userID uuid.UUID) (bool, error)
}
