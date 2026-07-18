package matchmaking

import (
	"context"
	"time"

	"github.com/google/uuid"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

// RedisQueueAdapter adapts the platform Redis coordinators to the matchmaking package.
// v1 (pair) remains available for rollout; v2 (team tickets) is additive.
type RedisQueueAdapter struct {
	inner   *redisplatform.MatchmakingCoordinator
	innerV2 *redisplatform.MatchmakingV2Coordinator
}

// NewRedisQueueAdapter wraps a platform matchmaking coordinator (v1 only).
func NewRedisQueueAdapter(inner *redisplatform.MatchmakingCoordinator) *RedisQueueAdapter {
	return &RedisQueueAdapter{inner: inner}
}

// NewRedisQueueAdapterWithV2 wraps both v1 and v2 platform coordinators.
func NewRedisQueueAdapterWithV2(v1 *redisplatform.MatchmakingCoordinator, v2 *redisplatform.MatchmakingV2Coordinator) *RedisQueueAdapter {
	return &RedisQueueAdapter{inner: v1, innerV2: v2}
}

// WithV2 attaches a v2 team-ticket coordinator for staged rollout.
func (a *RedisQueueAdapter) WithV2(v2 *redisplatform.MatchmakingV2Coordinator) *RedisQueueAdapter {
	if a == nil {
		return nil
	}
	a.innerV2 = v2
	return a
}

// V2 returns the underlying v2 coordinator when configured.
func (a *RedisQueueAdapter) V2() *redisplatform.MatchmakingV2Coordinator {
	if a == nil {
		return nil
	}
	return a.innerV2
}

// --- v1 pair queue surface (unchanged for rollout) ---

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

// --- v2 team-ticket surface ---

// JoinTicket enqueues an immutable roster ticket (Solo/Duo/Squad).
func (a *RedisQueueAdapter) JoinTicket(ctx context.Context, input TicketJoinInput, now time.Time, leaseTTL time.Duration) (*QueueTicket, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	ticket, err := a.innerV2.JoinTicket(ctx, toPlatformJoinInput(input), now, leaseTTL)
	if err != nil {
		return nil, mapV2Error(err)
	}
	return toQueueTicket(ticket), nil
}

// LeaveTicket removes the searching ticket that contains userID (whole roster).
func (a *RedisQueueAdapter) LeaveTicket(ctx context.Context, userID uuid.UUID) error {
	if a == nil || a.innerV2 == nil {
		return ErrUnavailable
	}
	err := a.innerV2.LeaveTicket(ctx, userID)
	return mapV2Error(err)
}

// GetTicketByUser loads the v2 ticket currently bound to userID, if any.
func (a *RedisQueueAdapter) GetTicketByUser(ctx context.Context, userID uuid.UUID) (*QueueTicket, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	ticket, err := a.innerV2.GetTicketByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toQueueTicket(ticket), nil
}

// GetTicket loads a v2 ticket by id.
func (a *RedisQueueAdapter) GetTicket(ctx context.Context, ticketID string) (*QueueTicket, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	ticket, err := a.innerV2.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	return toQueueTicket(ticket), nil
}

// RenewTicketLease extends a searching ticket lease without changing queue priority.
func (a *RedisQueueAdapter) RenewTicketLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueTicket, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	ticket, err := a.innerV2.RenewTicketLease(ctx, userID, leaseTTL, now)
	if err != nil {
		return nil, err
	}
	return toQueueTicket(ticket), nil
}

// ClaimTickets claims two equal-roster tickets under the production scan bound.
// maxRatingDelta <= 0 disables rating proximity filtering (Casual).
func (a *RedisQueueAdapter) ClaimTickets(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int, maxRatingDelta int) (*TeamClaim, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	claim, err := a.innerV2.ClaimTickets(ctx, mode, now, claimTTL, scanLimit, maxRatingDelta)
	if err != nil {
		return nil, err
	}
	return toTeamClaim(claim), nil
}

// GetTeamClaim loads a v2 claim record by id.
func (a *RedisQueueAdapter) GetTeamClaim(ctx context.Context, claimID string) (*TeamClaim, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	claim, err := a.innerV2.GetTeamClaim(ctx, claimID)
	if err != nil {
		return nil, err
	}
	return toTeamClaim(claim), nil
}

// FinalizeTeamClaim clears claimed ticket state after durable match commit.
func (a *RedisQueueAdapter) FinalizeTeamClaim(ctx context.Context, claim *TeamClaim) error {
	if a == nil || a.innerV2 == nil {
		return ErrUnavailable
	}
	return a.innerV2.FinalizeTeamClaim(ctx, fromTeamClaim(claim))
}

// ReleaseTeamClaim releases a claim and independently requeues each ticket when eligible.
// Callers must perform durable-first formation lookup before requeueing.
func (a *RedisQueueAdapter) ReleaseTeamClaim(ctx context.Context, claim *TeamClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	if a == nil || a.innerV2 == nil {
		return ErrUnavailable
	}
	return a.innerV2.ReleaseTeamClaim(ctx, fromTeamClaim(claim), requeueA, requeueB, leaseTTL)
}

// ListExpiredTeamClaims returns claim IDs past recover_after for durable-first reconciliation.
func (a *RedisQueueAdapter) ListExpiredTeamClaims(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if a == nil || a.innerV2 == nil {
		return nil, ErrUnavailable
	}
	return a.innerV2.ListExpiredTeamClaims(ctx, now, limit)
}

// DropTeamClaimIndex removes a stale claim id from the recovery index.
func (a *RedisQueueAdapter) DropTeamClaimIndex(ctx context.Context, claimID string) error {
	if a == nil || a.innerV2 == nil {
		return ErrUnavailable
	}
	return a.innerV2.DropTeamClaimIndex(ctx, claimID)
}

// TicketJoinInput is the matchmaking-package view of a v2 join roster snapshot.
type TicketJoinInput struct {
	Mode          string
	UserIDs       []uuid.UUID
	PartyID       uuid.UUID
	PartyVersion  int64
	TeamSize      int
	TeamAvgRating int
}

func mapV2Error(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case redisplatform.V2ClaimInProgress(err):
		return ErrClaimInProgress
	case redisplatform.V2UserConflict(err):
		// User already holds a different queue ticket; service (T035) can specialize.
		return ErrInvalidRequest
	case redisplatform.V2PartyVersionMismatch(err):
		return ErrInvalidRequest
	default:
		return err
	}
}

func toPlatformJoinInput(input TicketJoinInput) redisplatform.JoinTicketInput {
	return redisplatform.JoinTicketInput{
		Mode:          input.Mode,
		UserIDs:       input.UserIDs,
		PartyID:       input.PartyID,
		PartyVersion:  input.PartyVersion,
		TeamSize:      input.TeamSize,
		TeamAvgRating: input.TeamAvgRating,
	}
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

func toQueueTicket(ticket *redisplatform.QueueTicket) *QueueTicket {
	if ticket == nil {
		return nil
	}
	userIDs := make([]uuid.UUID, len(ticket.UserIDs))
	copy(userIDs, ticket.UserIDs)
	return &QueueTicket{
		TicketID:         ticket.TicketID,
		PartyID:          ticket.PartyID,
		Mode:             ticket.Mode,
		UserIDs:          userIDs,
		TeamSize:         ticket.TeamSize,
		TeamAvgRating:    ticket.TeamAvgRating,
		EnqueuedAtMs:     ticket.EnqueuedAtMs,
		LeaseExpiresAtMs: ticket.LeaseExpiresAtMs,
		State:            ticket.State,
		ClaimID:          ticket.ClaimID,
		PartyVersion:     ticket.PartyVersion,
	}
}

func toTeamClaim(claim *redisplatform.TeamClaim) *TeamClaim {
	if claim == nil {
		return nil
	}
	usersA := make([]uuid.UUID, len(claim.UserIDsA))
	copy(usersA, claim.UserIDsA)
	usersB := make([]uuid.UUID, len(claim.UserIDsB))
	copy(usersB, claim.UserIDsB)
	return &TeamClaim{
		ClaimID:        claim.ClaimID,
		FormationKey:   claim.FormationKey,
		Mode:           claim.Mode,
		TicketIDA:      claim.TicketIDA,
		TicketIDB:      claim.TicketIDB,
		UserIDsA:       usersA,
		UserIDsB:       usersB,
		PartyIDA:       claim.PartyIDA,
		PartyIDB:       claim.PartyIDB,
		PartyVersionA:  claim.PartyVersionA,
		PartyVersionB:  claim.PartyVersionB,
		TeamSize:       claim.TeamSize,
		TeamAvgRatingA: claim.TeamAvgRatingA,
		TeamAvgRatingB: claim.TeamAvgRatingB,
		EnqueuedAtMsA:  claim.EnqueuedAtMsA,
		EnqueuedAtMsB:  claim.EnqueuedAtMsB,
		ClaimedAtMs:    claim.ClaimedAtMs,
		RecoverAfterMs: claim.RecoverAfterMs,
	}
}

func fromTeamClaim(claim *TeamClaim) *redisplatform.TeamClaim {
	if claim == nil {
		return nil
	}
	usersA := make([]uuid.UUID, len(claim.UserIDsA))
	copy(usersA, claim.UserIDsA)
	usersB := make([]uuid.UUID, len(claim.UserIDsB))
	copy(usersB, claim.UserIDsB)
	return &redisplatform.TeamClaim{
		ClaimID:        claim.ClaimID,
		FormationKey:   claim.FormationKey,
		Mode:           claim.Mode,
		TicketIDA:      claim.TicketIDA,
		TicketIDB:      claim.TicketIDB,
		UserIDsA:       usersA,
		UserIDsB:       usersB,
		PartyIDA:       claim.PartyIDA,
		PartyIDB:       claim.PartyIDB,
		PartyVersionA:  claim.PartyVersionA,
		PartyVersionB:  claim.PartyVersionB,
		TeamSize:       claim.TeamSize,
		TeamAvgRatingA: claim.TeamAvgRatingA,
		TeamAvgRatingB: claim.TeamAvgRatingB,
		EnqueuedAtMsA:  claim.EnqueuedAtMsA,
		EnqueuedAtMsB:  claim.EnqueuedAtMsB,
		ClaimedAtMs:    claim.ClaimedAtMs,
		RecoverAfterMs: claim.RecoverAfterMs,
	}
}
