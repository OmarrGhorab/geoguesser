package matchmaking

import (
	"context"
	"time"

	"github.com/google/uuid"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

// RedisQueueAdapter adapts the platform Redis coordinator to QueueCoordinator.
type RedisQueueAdapter struct {
	inner *redisplatform.MatchmakingCoordinator
}

// NewRedisQueueAdapter wraps a platform matchmaking coordinator.
func NewRedisQueueAdapter(inner *redisplatform.MatchmakingCoordinator) *RedisQueueAdapter {
	return &RedisQueueAdapter{inner: inner}
}

func (a *RedisQueueAdapter) Join(ctx context.Context, userID uuid.UUID, mode string, now time.Time, leaseTTL time.Duration) (*QueueEntry, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	entry, err := a.inner.Join(ctx, userID, mode, now, leaseTTL)
	if err != nil {
		return nil, err
	}
	return toQueueEntry(entry), nil
}

func (a *RedisQueueAdapter) Leave(ctx context.Context, userID uuid.UUID) error {
	if a == nil || a.inner == nil {
		return ErrUnavailable
	}
	err := a.inner.Leave(ctx, userID)
	if redisplatform.ClaimInProgress(err) {
		return ErrClaimInProgress
	}
	return err
}

func (a *RedisQueueAdapter) GetEntry(ctx context.Context, userID uuid.UUID) (*QueueEntry, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	entry, err := a.inner.GetEntry(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toQueueEntry(entry), nil
}

func (a *RedisQueueAdapter) RenewLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueEntry, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	entry, err := a.inner.RenewLease(ctx, userID, leaseTTL, now)
	if err != nil {
		return nil, err
	}
	return toQueueEntry(entry), nil
}

func (a *RedisQueueAdapter) ClaimPair(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int) (*PairClaim, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	claim, err := a.inner.ClaimPair(ctx, mode, now, claimTTL, scanLimit)
	if err != nil {
		return nil, err
	}
	return toPairClaim(claim), nil
}

func (a *RedisQueueAdapter) GetClaim(ctx context.Context, claimID string) (*PairClaim, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	claim, err := a.inner.GetClaim(ctx, claimID)
	if err != nil {
		return nil, err
	}
	return toPairClaim(claim), nil
}

func (a *RedisQueueAdapter) FinalizeClaim(ctx context.Context, claim *PairClaim) error {
	if a == nil || a.inner == nil {
		return ErrUnavailable
	}
	return a.inner.FinalizeClaim(ctx, fromPairClaim(claim))
}

func (a *RedisQueueAdapter) ReleaseClaim(ctx context.Context, claim *PairClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	if a == nil || a.inner == nil {
		return ErrUnavailable
	}
	return a.inner.ReleaseClaim(ctx, fromPairClaim(claim), requeueA, requeueB, leaseTTL)
}

func (a *RedisQueueAdapter) ListExpiredClaims(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if a == nil || a.inner == nil {
		return nil, ErrUnavailable
	}
	return a.inner.ListExpiredClaims(ctx, now, limit)
}

func (a *RedisQueueAdapter) DropClaimIndex(ctx context.Context, claimID string) error {
	if a == nil || a.inner == nil {
		return ErrUnavailable
	}
	return a.inner.DropClaimIndex(ctx, claimID)
}

func toQueueEntry(entry *redisplatform.QueueEntry) *QueueEntry {
	if entry == nil {
		return nil
	}
	return &QueueEntry{
		EntryID:          entry.EntryID,
		UserID:           entry.UserID,
		Mode:             entry.Mode,
		State:            entry.State,
		EnqueuedAtMs:     entry.EnqueuedAtMs,
		LeaseExpiresAtMs: entry.LeaseExpiresAtMs,
		ClaimID:          entry.ClaimID,
	}
}

func toPairClaim(claim *redisplatform.PairClaim) *PairClaim {
	if claim == nil {
		return nil
	}
	return &PairClaim{
		ClaimID:        claim.ClaimID,
		FormationKey:   claim.FormationKey,
		Mode:           claim.Mode,
		EntryIDA:       claim.EntryIDA,
		UserIDA:        claim.UserIDA,
		EnqueuedAtMsA:  claim.EnqueuedAtMsA,
		EntryIDB:       claim.EntryIDB,
		UserIDB:        claim.UserIDB,
		EnqueuedAtMsB:  claim.EnqueuedAtMsB,
		ClaimedAtMs:    claim.ClaimedAtMs,
		RecoverAfterMs: claim.RecoverAfterMs,
	}
}

func fromPairClaim(claim *PairClaim) *redisplatform.PairClaim {
	if claim == nil {
		return nil
	}
	return &redisplatform.PairClaim{
		ClaimID:        claim.ClaimID,
		FormationKey:   claim.FormationKey,
		Mode:           claim.Mode,
		EntryIDA:       claim.EntryIDA,
		UserIDA:        claim.UserIDA,
		EnqueuedAtMsA:  claim.EnqueuedAtMsA,
		EntryIDB:       claim.EntryIDB,
		UserIDB:        claim.UserIDB,
		EnqueuedAtMsB:  claim.EnqueuedAtMsB,
		ClaimedAtMs:    claim.ClaimedAtMs,
		RecoverAfterMs: claim.RecoverAfterMs,
	}
}
