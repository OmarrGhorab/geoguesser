package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/session"
)

// QueueCoordinator is the Redis-backed ephemeral queue boundary.
type QueueCoordinator interface {
	Join(ctx context.Context, userID uuid.UUID, mode string, now time.Time, leaseTTL time.Duration) (*QueueEntry, error)
	Leave(ctx context.Context, userID uuid.UUID) error
	GetEntry(ctx context.Context, userID uuid.UUID) (*QueueEntry, error)
	RenewLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueEntry, error)
	ClaimPair(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int) (*PairClaim, error)
	GetClaim(ctx context.Context, claimID string) (*PairClaim, error)
	FinalizeClaim(ctx context.Context, claim *PairClaim) error
	// ReleaseClaim requeues players independently via requeueA/requeueB.
	ReleaseClaim(ctx context.Context, claim *PairClaim, requeueA, requeueB bool, leaseTTL time.Duration) error
	// ListExpiredClaims returns claim IDs past recover_after; durable-first reconciliation is service-owned.
	ListExpiredClaims(ctx context.Context, now time.Time, limit int) ([]string, error)
	DropClaimIndex(ctx context.Context, claimID string) error
}

// LocationSelector selects distinct active locations for ranked game formation.
type LocationSelector interface {
	SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]uuid.UUID, error)
}

// Clock provides injectable time for tests.
type Clock interface {
	Now() time.Time
}

// DurableStore is the PostgreSQL-backed matchmaking store.
type DurableStore interface {
	FindActiveUser(ctx context.Context, userID uuid.UUID) (*ActiveUser, error)
	FindActiveAssignment(ctx context.Context, userID uuid.UUID) (*ActiveAssignment, error)
	HasConflictingActiveGame(ctx context.Context, userID uuid.UUID) (bool, error)
	FindMatchByFormationKey(ctx context.Context, formationKey string) (*Match, error)
	CreateFormationBundle(ctx context.Context, input FormationInput) (*FormationResult, error)
}

// Config holds runtime matchmaking parameters.
type Config struct {
	DefaultMapID       uuid.UUID
	QueueLease         time.Duration
	ClaimTTL           time.Duration
	StartDelay         time.Duration
	RoundCount         int
	TimerSeconds       int
	CandidateScanLimit int
}

// Service orchestrates matchmaking queue and status operations.
type Service struct {
	store     DurableStore
	queue     QueueCoordinator
	locations LocationSelector
	clock     Clock
	logger    *slog.Logger
	metrics   MetricsRecorder
	cfg       Config
}

// systemClock is the default wall clock.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// NewService constructs a matchmaking service with required dependencies.
func NewService(store DurableStore, queue QueueCoordinator, cfg Config, logger *slog.Logger, metrics MetricsRecorder) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &Service{
		store:   store,
		queue:   queue,
		clock:   systemClock{},
		logger:  logger,
		metrics: metrics,
		cfg:     cfg,
	}
}

// WithLocations attaches a location selector used during formation (US2).
func (s *Service) WithLocations(selector LocationSelector) *Service {
	s.locations = selector
	return s
}

// WithClock overrides the wall clock (tests).
func (s *Service) WithClock(clock Clock) *Service {
	if clock != nil {
		s.clock = clock
	}
	return s
}

// JoinQueue enters ranked matchmaking or returns the existing searching/matched state.
func (s *Service) JoinQueue(ctx context.Context, sess *session.Context, req JoinQueueRequest) (resp *StatusResponse, err error) {
	start := s.clock.Now()
	defer func() {
		s.metrics.ObserveCommand("join", commandOutcome(err), s.clock.Now().Sub(start))
	}()

	userID, err := registeredUserID(sess)
	if err != nil {
		return nil, err
	}

	if !SupportedMode(req.Mode) {
		if req.Mode == "" {
			return nil, ErrInvalidRequest
		}
		return nil, ErrUnsupportedMode
	}

	if assignment, err := s.store.FindActiveAssignment(ctx, userID); err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return nil, ErrUnavailable
	} else if assignment != nil {
		return NewMatchedStatus(assignment.MatchID, assignment.GameID, assignment.Mode, assignment.MatchedAt), nil
	}

	if err := s.ensureEligible(ctx, userID); err != nil {
		return nil, err
	}

	conflict, err := s.store.HasConflictingActiveGame(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return nil, ErrUnavailable
	}
	if conflict {
		return nil, ErrActiveGameConflict
	}

	if err := s.ensureContentAvailable(ctx); err != nil {
		return nil, err
	}

	entry, err := s.queue.Join(ctx, userID, req.Mode, s.clock.Now(), s.cfg.QueueLease)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		return nil, ErrUnavailable
	}
	if entry.State == QueueStateClaimed {
		return s.recoverClaimedStatus(ctx, userID, entry)
	}

	// Request-driven formation: a second compatible join may form a match synchronously.
	if matched, formationErr := s.tryFormMatch(ctx, req.Mode); formationErr != nil {
		return nil, formationErr
	} else if matched != nil {
		// If this player was assigned, return matched; otherwise keep searching.
		assignment, refreshErr := s.store.FindActiveAssignment(ctx, userID)
		if refreshErr != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, ErrUnavailable
		}
		if assignment != nil {
			s.metrics.ObserveStatus(PublicStatusMatched)
			return NewMatchedStatus(assignment.MatchID, assignment.GameID, assignment.Mode, assignment.MatchedAt), nil
		}
	}

	s.metrics.ObserveStatus(PublicStatusSearching)
	return statusFromEntry(entry), nil
}

// tryFormMatch runs one bounded claim+formation attempt. Returns matched status for either player when successful.
func (s *Service) tryFormMatch(ctx context.Context, mode string) (*StatusResponse, error) {
	start := s.clock.Now()
	// Durable-first cleanup of expired claims before attempting a new pair claim.
	s.reconcileExpiredClaims(ctx)
	if s.locations == nil || s.cfg.DefaultMapID == uuid.Nil {
		return nil, nil
	}

	claim, err := s.queue.ClaimPair(ctx, mode, s.clock.Now(), s.cfg.ClaimTTL, s.cfg.CandidateScanLimit)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveFormation("claim_error", s.clock.Now().Sub(start))
		return nil, nil
	}
	if claim == nil {
		s.metrics.ObserveFormation("no_pair", s.clock.Now().Sub(start))
		return nil, nil
	}

	// Durable-first: if formation already committed for this claim, finalize Redis and return.
	existing, err := s.store.FindMatchByFormationKey(ctx, claim.FormationKey)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
		return nil, ErrUnavailable
	}
	if existing != nil {
		if err := s.queue.FinalizeClaim(ctx, claim); err != nil {
			s.recordRedisCleanupFailure(ctx, "finalize_replay", claim, err)
		}
		s.metrics.ObserveFormation("replay", s.clock.Now().Sub(start))
		s.metrics.ObserveRecovery("formation_key_hit")
		return NewMatchedStatus(existing.ID, existing.GameID, existing.Mode, existing.MatchedAt), nil
	}

	// Revalidate candidates independently so an ineligible player does not discard a valid opponent.
	requeueA, reasonA := s.candidateRequeue(ctx, claim.UserIDA)
	requeueB, reasonB := s.candidateRequeue(ctx, claim.UserIDB)
	if reasonA == "db_error" || reasonB == "db_error" {
		// Preserve the exact claim until PostgreSQL can determine whether a
		// formation committed. Requeueing under uncertainty can resurrect a
		// durable assignment after a commit/finalize crash window.
		s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
		return nil, ErrUnavailable
	}
	if reasonA != "" || reasonB != "" {
		if err := s.queue.ReleaseClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
			s.recordRedisCleanupFailure(ctx, "release_ineligible", claim, err)
			return nil, ErrUnavailable
		}
		outcome := "ineligible"
		if reasonA == "already_assigned" || reasonB == "already_assigned" {
			outcome = "already_assigned"
		}
		s.metrics.ObserveFormation(outcome, s.clock.Now().Sub(start))
		return nil, nil
	}

	locationIDs, err := s.locations.SelectLocations(ctx, s.cfg.DefaultMapID, s.cfg.RoundCount)
	if err != nil || len(locationIDs) < s.cfg.RoundCount {
		if releaseErr := s.queue.ReleaseClaim(ctx, claim, true, true, s.cfg.QueueLease); releaseErr != nil {
			s.recordRedisCleanupFailure(ctx, "release_locations", claim, releaseErr)
			return nil, ErrUnavailable
		}
		s.metrics.ObserveFormation("locations_unavailable", s.clock.Now().Sub(start))
		return nil, nil
	}

	result, err := s.store.CreateFormationBundle(ctx, FormationInput{
		FormationKey: claim.FormationKey,
		Mode:         mode,
		MapID:        s.cfg.DefaultMapID,
		RoundCount:   s.cfg.RoundCount,
		TimerSeconds: s.cfg.TimerSeconds,
		StartDelay:   s.cfg.StartDelay,
		UserIDs:      [2]uuid.UUID{claim.UserIDA, claim.UserIDB},
		LocationIDs:  locationIDs,
		MatchedAt:    s.clock.Now(),
	})
	if err != nil {
		if errors.Is(err, ErrAccountIneligible) || errors.Is(err, ErrAlreadyAssigned) || errors.Is(err, ErrActiveGameConflict) {
			// Bundle rejected for eligibility/assignment: re-evaluate each player.
			requeueA, reasonA = s.candidateRequeue(ctx, claim.UserIDA)
			requeueB, reasonB = s.candidateRequeue(ctx, claim.UserIDB)
			if reasonA == "db_error" || reasonB == "db_error" {
				s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
				return nil, ErrUnavailable
			}
			if releaseErr := s.queue.ReleaseClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); releaseErr != nil {
				s.recordRedisCleanupFailure(ctx, "release_formation_rejected", claim, releaseErr)
				return nil, ErrUnavailable
			}
		} else {
			// An unexpected write error is ambiguous: PostgreSQL may have committed
			// while the connection failed. Keep the claim for durable-first recovery.
			s.metrics.ObserveDependencyFailure("postgres")
			s.metrics.ObserveFormation("formation_failed", s.clock.Now().Sub(start))
			s.logger.WarnContext(ctx, "match formation failed",
				slog.String("formation_key", claim.FormationKey),
				slog.String("outcome", "formation_failed"),
			)
			return nil, ErrUnavailable
		}
		s.metrics.ObserveFormation("formation_failed", s.clock.Now().Sub(start))
		s.logger.WarnContext(ctx, "match formation failed",
			slog.String("formation_key", claim.FormationKey),
			slog.String("outcome", "formation_failed"),
		)
		return nil, nil
	}

	// Durable commit succeeded; best-effort Redis finalize. Status recovery is durable-first.
	if err := s.queue.FinalizeClaim(ctx, claim); err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveRecovery("finalize_failed")
		s.logger.WarnContext(ctx, "match finalize redis failed after durable commit",
			slog.String("formation_key", claim.FormationKey),
			slog.String("match_id", result.Match.ID.String()),
		)
	}

	s.metrics.ObserveFormation("formed", s.clock.Now().Sub(start))
	// Safe operational log: match/game IDs only — never player IDs, tokens, Redis keys, or coordinates.
	s.logger.InfoContext(ctx, "ranked match formed",
		slog.String("match_id", result.Match.ID.String()),
		slog.String("game_id", result.Match.GameID.String()),
		slog.String("mode", result.Match.Mode),
		slog.String("outcome", "formed"),
	)
	return NewMatchedStatus(result.Match.ID, result.Match.GameID, result.Match.Mode, result.Match.MatchedAt), nil
}

func (s *Service) recordRedisCleanupFailure(ctx context.Context, operation string, claim *PairClaim, err error) {
	s.metrics.ObserveDependencyFailure("redis")
	s.metrics.ObserveRecovery(operation + "_failed")
	attrs := []any{slog.String("operation", operation), slog.Any("error", err)}
	if claim != nil {
		attrs = append(attrs, slog.String("formation_key", claim.FormationKey))
	}
	s.logger.WarnContext(ctx, "matchmaking redis cleanup failed", attrs...)
}

// reconcileExpiredClaims checks PostgreSQL formation first, finalizes committed claims,
// and only requeues uncommitted eligible players (independently).
func (s *Service) reconcileExpiredClaims(ctx context.Context) {
	ids, err := s.queue.ListExpiredClaims(ctx, s.clock.Now(), s.cfg.CandidateScanLimit)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		return
	}
	for _, id := range ids {
		claim, err := s.queue.GetClaim(ctx, id)
		if err != nil {
			s.metrics.ObserveDependencyFailure("redis")
			return
		}
		if claim == nil {
			if err := s.queue.DropClaimIndex(ctx, id); err != nil {
				s.recordRedisCleanupFailure(ctx, "drop_orphan_claim_index", nil, err)
				continue
			}
			s.metrics.ObserveStaleEntry()
			continue
		}
		existing, err := s.store.FindMatchByFormationKey(ctx, claim.FormationKey)
		if err != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			continue
		}
		if existing != nil {
			if err := s.queue.FinalizeClaim(ctx, claim); err != nil {
				s.recordRedisCleanupFailure(ctx, "reconcile_finalize", claim, err)
				continue
			}
			s.metrics.ObserveRecovery("reconcile_finalize")
			continue
		}
		requeueA, reasonA := s.candidateRequeue(ctx, claim.UserIDA)
		requeueB, reasonB := s.candidateRequeue(ctx, claim.UserIDB)
		if reasonA == "db_error" || reasonB == "db_error" {
			// Keep the claim until durable state can be established.
			continue
		}
		if err := s.queue.ReleaseClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
			s.recordRedisCleanupFailure(ctx, "reconcile_release", claim, err)
			continue
		}
		s.metrics.ObserveRecovery("reconcile_requeue")
		if !requeueA || !requeueB {
			s.metrics.ObserveStaleEntry()
		}
	}
}

// candidateRequeue reports whether a player should be requeued and a short rejection reason.
// reason is empty when eligible; "db_error" when durable checks fail transiently.
func (s *Service) candidateRequeue(ctx context.Context, userID uuid.UUID) (requeue bool, reason string) {
	if err := s.ensureEligible(ctx, userID); err != nil {
		if errors.Is(err, ErrUnavailable) {
			return false, "db_error"
		}
		return false, "ineligible"
	}
	assignment, err := s.store.FindActiveAssignment(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return false, "db_error"
	}
	if assignment != nil {
		return false, "already_assigned"
	}
	conflict, err := s.store.HasConflictingActiveGame(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return false, "db_error"
	}
	if conflict {
		return false, "active_game"
	}
	return true, ""
}

// LeaveQueue removes the caller's searching entry when safe.
func (s *Service) LeaveQueue(ctx context.Context, sess *session.Context) (err error) {
	start := s.clock.Now()
	defer func() {
		s.metrics.ObserveCommand("leave", commandOutcome(err), s.clock.Now().Sub(start))
	}()

	userID, err := registeredUserID(sess)
	if err != nil {
		return err
	}

	if assignment, err := s.store.FindActiveAssignment(ctx, userID); err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return ErrUnavailable
	} else if assignment != nil {
		// Already matched; leave is a no-op for durable assignment.
		return nil
	}

	entry, err := s.queue.GetEntry(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		return ErrUnavailable
	}
	if entry == nil {
		return nil
	}
	if entry.State == QueueStateClaimed {
		return ErrClaimInProgress
	}

	if err := s.queue.Leave(ctx, userID); err != nil {
		if errors.Is(err, ErrClaimInProgress) {
			return ErrClaimInProgress
		}
		s.metrics.ObserveDependencyFailure("redis")
		return ErrUnavailable
	}
	return nil
}

func commandOutcome(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrUnsupportedMode):
		return "invalid_request"
	case errors.Is(err, ErrClaimInProgress), errors.Is(err, ErrActiveGameConflict), errors.Is(err, ErrAlreadyAssigned):
		return "conflict"
	case errors.Is(err, ErrAccountIneligible), errors.Is(err, ErrContentUnavailable):
		return "ineligible"
	default:
		return "error"
	}
}

// GetStatus returns durable-first matchmaking status for the registered player.
func (s *Service) GetStatus(ctx context.Context, sess *session.Context) (*StatusResponse, error) {
	userID, err := registeredUserID(sess)
	if err != nil {
		return nil, err
	}

	assignment, err := s.store.FindActiveAssignment(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return nil, ErrUnavailable
	}
	if assignment != nil {
		s.metrics.ObserveStatus(PublicStatusMatched)
		return NewMatchedStatus(assignment.MatchID, assignment.GameID, assignment.Mode, assignment.MatchedAt), nil
	}

	// Request-driven formation/recovery while polling.
	if assignment == nil {
		if _, err := s.tryFormMatch(ctx, ModeRankedStandard); err != nil {
			return nil, err
		}
		refreshed, refreshErr := s.store.FindActiveAssignment(ctx, userID)
		if refreshErr != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, ErrUnavailable
		}
		if refreshed != nil {
			s.metrics.ObserveStatus(PublicStatusMatched)
			return NewMatchedStatus(refreshed.MatchID, refreshed.GameID, refreshed.Mode, refreshed.MatchedAt), nil
		}
	}

	return s.statusWithoutDurable(ctx, userID)
}

func (s *Service) statusWithoutDurable(ctx context.Context, userID uuid.UUID) (*StatusResponse, error) {
	entry, err := s.queue.RenewLease(ctx, userID, s.cfg.QueueLease, s.clock.Now())
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if entry == nil {
		s.metrics.ObserveStatus(PublicStatusNotQueued)
		return NewNotQueuedStatus(), nil
	}
	if entry.State == QueueStateClaimed {
		return s.recoverClaimedStatus(ctx, userID, entry)
	}
	s.metrics.ObserveStatus(PublicStatusSearching)
	return statusFromEntry(entry), nil
}

// recoverClaimedStatus resolves durable-first status for a claimed player and cleans expired claims.
func (s *Service) recoverClaimedStatus(ctx context.Context, userID uuid.UUID, entry *QueueEntry) (*StatusResponse, error) {
	if entry.ClaimID != "" {
		match, err := s.store.FindMatchByFormationKey(ctx, entry.ClaimID)
		if err != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
			return NewTemporarilyUnavailableStatus(), nil
		}
		if match != nil {
			claim, claimErr := s.queue.GetClaim(ctx, entry.ClaimID)
			if claimErr != nil {
				s.recordRedisCleanupFailure(ctx, "get_claim_after_durable_hit", nil, claimErr)
			} else if claim != nil {
				if finalizeErr := s.queue.FinalizeClaim(ctx, claim); finalizeErr != nil {
					s.recordRedisCleanupFailure(ctx, "finalize_durable_hit", claim, finalizeErr)
				}
			}
			s.metrics.ObserveRecovery("claimed_durable_hit")
			s.metrics.ObserveStatus(PublicStatusMatched)
			return NewMatchedStatus(match.ID, match.GameID, match.Mode, match.MatchedAt), nil
		}
		claim, err := s.queue.GetClaim(ctx, entry.ClaimID)
		if err != nil {
			s.metrics.ObserveDependencyFailure("redis")
			s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
			return NewTemporarilyUnavailableStatus(), nil
		}
		nowMs := s.clock.Now().UTC().UnixMilli()
		if claim == nil {
			s.metrics.ObserveRecovery("claim_payload_missing")
			s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
			return NewTemporarilyUnavailableStatus(), nil
		}
		if claim.RecoverAfterMs <= nowMs {
			// Durable-first before requeue: formation may have committed without Redis finalize.
			match, err := s.store.FindMatchByFormationKey(ctx, claim.FormationKey)
			if err != nil {
				s.metrics.ObserveDependencyFailure("postgres")
				s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
				return NewTemporarilyUnavailableStatus(), nil
			}
			if match != nil {
				if finalizeErr := s.queue.FinalizeClaim(ctx, claim); finalizeErr != nil {
					s.recordRedisCleanupFailure(ctx, "finalize_expired_durable_hit", claim, finalizeErr)
				}
				s.metrics.ObserveRecovery("claimed_durable_hit")
				s.metrics.ObserveStatus(PublicStatusMatched)
				return NewMatchedStatus(match.ID, match.GameID, match.Mode, match.MatchedAt), nil
			}
			requeueA, reasonA := s.candidateRequeue(ctx, claim.UserIDA)
			requeueB, reasonB := s.candidateRequeue(ctx, claim.UserIDB)
			if reasonA == "db_error" || reasonB == "db_error" {
				s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
				return NewTemporarilyUnavailableStatus(), nil
			}
			if err := s.queue.ReleaseClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
				s.metrics.ObserveDependencyFailure("redis")
				s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
				return NewTemporarilyUnavailableStatus(), nil
			}
			s.metrics.ObserveRecovery("expired_claim_requeued")
			// Player should be searching again after requeue.
			renewed, renewErr := s.queue.RenewLease(ctx, userID, s.cfg.QueueLease, s.clock.Now())
			if renewErr != nil {
				s.recordRedisCleanupFailure(ctx, "renew_after_release", claim, renewErr)
				s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
				return NewTemporarilyUnavailableStatus(), nil
			}
			if renewed != nil && renewed.State == QueueStateSearching {
				s.metrics.ObserveStatus(PublicStatusSearching)
				return statusFromEntry(renewed), nil
			}
			s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
			return NewTemporarilyUnavailableStatus(), nil
		}
	}
	s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
	return NewTemporarilyUnavailableStatus(), nil
}

func (s *Service) ensureContentAvailable(ctx context.Context) error {
	if s.locations == nil || s.cfg.DefaultMapID == uuid.Nil {
		return ErrContentUnavailable
	}
	ids, err := s.locations.SelectLocations(ctx, s.cfg.DefaultMapID, s.cfg.RoundCount)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return ErrContentUnavailable
	}
	if len(ids) < s.cfg.RoundCount {
		return ErrContentUnavailable
	}
	return nil
}

func (s *Service) ensureEligible(ctx context.Context, userID uuid.UUID) error {
	user, err := s.store.FindActiveUser(ctx, userID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return ErrUnavailable
	}
	if user == nil || user.Status != "active" {
		return ErrAccountIneligible
	}
	return nil
}

func registeredUserID(sess *session.Context) (uuid.UUID, error) {
	if sess == nil || !sess.IsRegistered() || sess.UserID == nil {
		return uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func statusFromEntry(entry *QueueEntry) *StatusResponse {
	if entry == nil {
		return NewNotQueuedStatus()
	}
	return NewSearchingStatus(
		entry.Mode,
		time.UnixMilli(entry.EnqueuedAtMs).UTC(),
		time.UnixMilli(entry.LeaseExpiresAtMs).UTC(),
	)
}
