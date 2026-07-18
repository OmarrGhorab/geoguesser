package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/session"
)

// QueueCoordinator is the Redis-backed ephemeral v1 pair-queue boundary.
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

// TicketQueueCoordinator is the Redis-backed v2 team-ticket boundary.
type TicketQueueCoordinator interface {
	JoinTicket(ctx context.Context, input TicketJoinInput, now time.Time, leaseTTL time.Duration) (*QueueTicket, error)
	LeaveTicket(ctx context.Context, userID uuid.UUID) error
	GetTicketByUser(ctx context.Context, userID uuid.UUID) (*QueueTicket, error)
	GetTicket(ctx context.Context, ticketID string) (*QueueTicket, error)
	RenewTicketLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueTicket, error)
	ClaimTickets(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int, maxRatingDelta int) (*TeamClaim, error)
	GetTeamClaim(ctx context.Context, claimID string) (*TeamClaim, error)
	FinalizeTeamClaim(ctx context.Context, claim *TeamClaim) error
	ReleaseTeamClaim(ctx context.Context, claim *TeamClaim, requeueA, requeueB bool, leaseTTL time.Duration) error
	ListExpiredTeamClaims(ctx context.Context, now time.Time, limit int) ([]string, error)
	DropTeamClaimIndex(ctx context.Context, claimID string) error
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
	CreateTeamFormationBundle(ctx context.Context, input TeamFormationInput) (*FormationResult, error)
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
	// CasualMatchmakingEnabled gates casual_* queues (default false).
	CasualMatchmakingEnabled bool
	// RankedTeamModesEnabled gates ranked duo/squad and v2 solo tickets (default false).
	// When false, ranked solo (including ranked_standard) uses the legacy v1 pair path.
	RankedTeamModesEnabled bool
	// MaxRatingDelta bounds ranked ticket claims; <=0 disables proximity filter (Casual).
	MaxRatingDelta int
}

// Service orchestrates matchmaking queue and status operations.
type Service struct {
	store       DurableStore
	queue       QueueCoordinator
	tickets     TicketQueueCoordinator
	parties     PartySnapshotReader
	competitive CompetitiveStandingReader
	notifier    MatchFormationNotifier
	events      EventPublisher
	locations   LocationSelector
	clock       Clock
	logger      *slog.Logger
	metrics     MetricsRecorder
	cfg         Config
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

// WithTickets attaches the v2 team-ticket coordinator.
func (s *Service) WithTickets(tickets TicketQueueCoordinator) *Service {
	s.tickets = tickets
	return s
}

// WithParties attaches the party readiness reader for duo/squad queues.
func (s *Service) WithParties(parties PartySnapshotReader) *Service {
	s.parties = parties
	return s
}

// WithCompetitive attaches the competitive standing reader for ranked tickets.
func (s *Service) WithCompetitive(reader CompetitiveStandingReader) *Service {
	s.competitive = reader
	return s
}

// WithNotifier attaches post-commit match assignment fanout.
func (s *Service) WithNotifier(notifier MatchFormationNotifier) *Service {
	s.notifier = notifier
	return s
}

// WithEvents attaches post-commit queue lifecycle fanout.
func (s *Service) WithEvents(events EventPublisher) *Service {
	s.events = events
	return s
}

// JoinQueue enters matchmaking or returns the existing searching/matched state.
func (s *Service) JoinQueue(ctx context.Context, sess *session.Context, req JoinQueueRequest) (resp *StatusResponse, err error) {
	start := s.clock.Now()
	defer func() {
		s.metrics.ObserveCommand("join", commandOutcome(err), s.clock.Now().Sub(start))
	}()

	userID, err := registeredUserID(sess)
	if err != nil {
		return nil, err
	}

	parts, err := req.ResolveMode()
	if err != nil {
		if errors.Is(err, ErrUnsupportedMode) {
			return nil, ErrUnsupportedMode
		}
		if errors.Is(err, ErrInvalidModeSelection) {
			return nil, ErrInvalidModeSelection
		}
		return nil, ErrInvalidRequest
	}
	if err := s.ensureModeEnabled(parts); err != nil {
		return nil, err
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

	if s.useLegacyPairQueue(parts) {
		return s.joinLegacyPairQueue(ctx, userID, parts)
	}
	return s.joinTicketQueue(ctx, userID, parts, req)
}

func (s *Service) joinLegacyPairQueue(ctx context.Context, userID uuid.UUID, parts ModeParts) (*StatusResponse, error) {
	// Preserve ranked_standard wire value for v1 Redis keys / durable mode.
	mode := ModeRankedStandard
	if parts.Raw != "" && parts.Raw != ModeRankedStandard {
		// ranked_solo with team modes disabled still uses v1 under the ranked_standard queue.
		mode = ModeRankedStandard
	}

	entry, err := s.queue.Join(ctx, userID, mode, s.clock.Now(), s.cfg.QueueLease)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		return nil, ErrUnavailable
	}
	if entry.State == QueueStateClaimed {
		return s.recoverClaimedStatus(ctx, userID, entry)
	}

	if matched, formationErr := s.tryFormMatch(ctx, mode); formationErr != nil {
		return nil, formationErr
	} else if matched != nil {
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

func (s *Service) joinTicketQueue(ctx context.Context, userID uuid.UUID, parts ModeParts, req JoinQueueRequest) (*StatusResponse, error) {
	if s.tickets == nil {
		return nil, ErrUnavailable
	}

	roster, partyID, partyVersion, err := s.resolveTicketRoster(ctx, userID, parts, req)
	if err != nil {
		return nil, err
	}

	avgRating, err := s.teamAverageRating(ctx, parts, roster)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	ticket, err := s.tickets.JoinTicket(ctx, TicketJoinInput{
		Mode:          parts.Canonical,
		UserIDs:       roster,
		PartyID:       partyID,
		PartyVersion:  partyVersion,
		TeamSize:      parts.TeamSize,
		TeamAvgRating: avgRating,
	}, now, s.cfg.QueueLease)
	if err != nil {
		if errors.Is(err, ErrClaimInProgress) {
			return nil, ErrClaimInProgress
		}
		if errors.Is(err, ErrInvalidRequest) {
			return nil, ErrInvalidRequest
		}
		s.metrics.ObserveDependencyFailure("redis")
		return nil, ErrUnavailable
	}
	if ticket.State == QueueStateClaimed {
		return s.recoverTicketClaimedStatus(ctx, userID, ticket)
	}

	if _, formationErr := s.tryFormTeamMatch(ctx, parts.Canonical); formationErr != nil {
		return nil, formationErr
	}
	assignment, refreshErr := s.store.FindActiveAssignment(ctx, userID)
	if refreshErr != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		return nil, ErrUnavailable
	}
	if assignment != nil {
		s.metrics.ObserveStatus(PublicStatusMatched)
		return NewMatchedStatus(assignment.MatchID, assignment.GameID, assignment.Mode, assignment.MatchedAt), nil
	}

	// Refresh ticket after possible formation of other players.
	if renewed, renewErr := s.tickets.GetTicketByUser(ctx, userID); renewErr == nil && renewed != nil {
		ticket = renewed
	}

	s.metrics.ObserveStatus(PublicStatusSearching)
	s.publishPartyQueueChanged(ctx, ticket, PublicStatusSearching)
	return NewSearchingStatusFromTicket(ticket, s.ratingWindowForTicket(ticket)), nil
}

func (s *Service) publishPartyQueueChanged(ctx context.Context, ticket *QueueTicket, status string) {
	if s == nil || s.events == nil || ticket == nil || ticket.PartyID == uuid.Nil {
		return
	}
	_ = s.events.Publish(ctx, "party", ticket.PartyID.String(), EventPartyQueueChanged, ticket.PartyVersion, ticket.UserIDs, map[string]any{
		"status": status,
		"mode":   ticket.Mode,
	})
}

func (s *Service) resolveTicketRoster(ctx context.Context, userID uuid.UUID, parts ModeParts, req JoinQueueRequest) (roster []uuid.UUID, partyID uuid.UUID, partyVersion int64, err error) {
	if parts.Format == FormatSolo {
		if req.PartyID != nil {
			return nil, uuid.Nil, 0, ErrInvalidRequest
		}
		return []uuid.UUID{userID}, uuid.Nil, 0, nil
	}

	if req.PartyID == nil {
		return nil, uuid.Nil, 0, ErrInvalidRequest
	}
	if s.parties == nil {
		return nil, uuid.Nil, 0, ErrUnavailable
	}
	if req.PartyVersion == nil {
		return nil, uuid.Nil, 0, ErrInvalidRequest
	}

	snap, err := s.parties.ReadyPartyForQueue(ctx, userID, parts.Format, *req.PartyVersion)
	if err != nil {
		return nil, uuid.Nil, 0, mapPartyReaderError(err)
	}
	if snap == nil {
		return nil, uuid.Nil, 0, ErrInvalidRequest
	}
	if snap.PartyID != *req.PartyID {
		return nil, uuid.Nil, 0, ErrInvalidRequest
	}
	if len(snap.MemberIDs) != parts.TeamSize {
		return nil, uuid.Nil, 0, ErrInvalidRequest
	}
	// Revalidate every roster member under service eligibility rules.
	for _, memberID := range snap.MemberIDs {
		if err := s.ensureEligible(ctx, memberID); err != nil {
			return nil, uuid.Nil, 0, err
		}
		conflict, err := s.store.HasConflictingActiveGame(ctx, memberID)
		if err != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, uuid.Nil, 0, ErrUnavailable
		}
		if conflict {
			return nil, uuid.Nil, 0, ErrActiveGameConflict
		}
		assignment, err := s.store.FindActiveAssignment(ctx, memberID)
		if err != nil {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, uuid.Nil, 0, ErrUnavailable
		}
		if assignment != nil {
			return nil, uuid.Nil, 0, ErrAlreadyAssigned
		}
	}
	return append([]uuid.UUID(nil), snap.MemberIDs...), snap.PartyID, snap.Version, nil
}

func (s *Service) teamAverageRating(ctx context.Context, parts ModeParts, roster []uuid.UUID) (int, error) {
	if parts.Playlist != PlaylistRanked {
		return 0, nil
	}
	if s.competitive == nil {
		// Competitive not injected yet (parallel agent / local stubs): allow join with zero avg.
		return 0, nil
	}
	seasonID, err := s.competitive.ActiveSeasonID(ctx)
	if err != nil {
		s.metrics.ObserveDependencyFailure("competitive")
		return 0, ErrUnavailable
	}
	if seasonID == uuid.Nil {
		return 0, ErrContentUnavailable
	}
	standings, err := s.competitive.StandingsForUsers(ctx, seasonID, roster)
	if err != nil {
		s.metrics.ObserveDependencyFailure("competitive")
		return 0, ErrUnavailable
	}
	if len(standings) == 0 {
		// Lazy standing creation failed or empty season — treat as content unavailable for ranked.
		return 0, ErrContentUnavailable
	}
	// Index standings by user; require every roster member (placement uses hidden rating).
	byUser := make(map[uuid.UUID]CompetitiveStandingSnapshot, len(standings))
	for _, st := range standings {
		byUser[st.UserID] = st
	}
	ratings := make([]int, 0, len(roster))
	tiers := make([]int, 0, len(roster))
	for _, userID := range roster {
		st, ok := byUser[userID]
		if !ok {
			return 0, ErrContentUnavailable
		}
		ratings = append(ratings, st.Rating)
		tiers = append(tiers, EffectiveNamedRankTier(st.Rating, st.NamedRankTier, st.PlacementsCompleted, st.RankCode))
	}
	if !PartyNamedRankSpreadOK(tiers) {
		return 0, ErrInvalidRequest
	}
	return TeamAverageRating(ratings), nil
}

// ratingWindowForTicket builds the public ranked search window; nil for casual.
func (s *Service) ratingWindowForTicket(ticket *QueueTicket) *RatingWindow {
	if ticket == nil {
		return nil
	}
	parts, err := ParseMode(ticket.Mode)
	if err != nil || parts.Playlist != PlaylistRanked {
		return nil
	}
	waited := s.clock.Now().UTC().Sub(time.UnixMilli(ticket.EnqueuedAtMs).UTC())
	win := RatingWindowForAverage(ticket.TeamAvgRating, waited)
	return &win
}

// rankedClaimMaxDelta is the cap passed to Redis; Lua expands per-ticket up to this cap.
func (s *Service) rankedClaimMaxDelta() int {
	if s.cfg.MaxRatingDelta > 0 {
		return s.cfg.MaxRatingDelta
	}
	return RatingWindowCap
}

// tryFormMatch runs one bounded claim+formation attempt for the legacy v1 pair path.
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

	s.notifyFormation(ctx, &result.Match, []uuid.UUID{claim.UserIDA, claim.UserIDB}, nil, nil)
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

// tryFormTeamMatch runs one bounded v2 ticket claim + team formation attempt.
func (s *Service) tryFormTeamMatch(ctx context.Context, mode string) (*StatusResponse, error) {
	start := s.clock.Now()
	if s.tickets == nil {
		return nil, nil
	}
	s.reconcileExpiredTeamClaims(ctx)
	if s.locations == nil || s.cfg.DefaultMapID == uuid.Nil {
		return nil, nil
	}

	parts, err := ParseMode(mode)
	if err != nil {
		return nil, nil
	}
	maxDelta := 0
	if parts.Playlist == PlaylistRanked {
		// Cap for expanding windows (±100 → ±400). Lua applies per-ticket expansion up to this cap.
		maxDelta = s.rankedClaimMaxDelta()
	}

	claim, err := s.tickets.ClaimTickets(ctx, parts.Canonical, s.clock.Now(), s.cfg.ClaimTTL, s.cfg.CandidateScanLimit, maxDelta)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveFormation("claim_error", s.clock.Now().Sub(start))
		return nil, nil
	}
	if claim == nil {
		s.metrics.ObserveFormation("no_pair", s.clock.Now().Sub(start))
		return nil, nil
	}

	existing, err := s.store.FindMatchByFormationKey(ctx, claim.FormationKey)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
		return nil, ErrUnavailable
	}
	if existing != nil {
		if err := s.tickets.FinalizeTeamClaim(ctx, claim); err != nil {
			s.recordTeamRedisCleanupFailure(ctx, "finalize_replay", claim, err)
		}
		s.metrics.ObserveFormation("replay", s.clock.Now().Sub(start))
		s.metrics.ObserveRecovery("formation_key_hit")
		return NewMatchedStatus(existing.ID, existing.GameID, existing.Mode, existing.MatchedAt), nil
	}

	requeueA, reasonA := s.rosterRequeue(ctx, claim.UserIDsA)
	requeueB, reasonB := s.rosterRequeue(ctx, claim.UserIDsB)
	if reasonA == "db_error" || reasonB == "db_error" {
		s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
		return nil, ErrUnavailable
	}
	if reasonA != "" || reasonB != "" {
		if err := s.tickets.ReleaseTeamClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
			s.recordTeamRedisCleanupFailure(ctx, "release_ineligible", claim, err)
			return nil, ErrUnavailable
		}
		s.metrics.ObserveFormation("ineligible", s.clock.Now().Sub(start))
		return nil, nil
	}

	locationIDs, err := s.locations.SelectLocations(ctx, s.cfg.DefaultMapID, s.cfg.RoundCount)
	if err != nil || len(locationIDs) < s.cfg.RoundCount {
		if releaseErr := s.tickets.ReleaseTeamClaim(ctx, claim, true, true, s.cfg.QueueLease); releaseErr != nil {
			s.recordTeamRedisCleanupFailure(ctx, "release_locations", claim, releaseErr)
			return nil, ErrUnavailable
		}
		s.metrics.ObserveFormation("locations_unavailable", s.clock.Now().Sub(start))
		return nil, nil
	}

	var timer *int
	if parts.Playlist == PlaylistRanked {
		t := s.cfg.TimerSeconds
		timer = &t
	}
	var partyOne, partyTwo *uuid.UUID
	if claim.PartyIDA != uuid.Nil {
		id := claim.PartyIDA
		partyOne = &id
	}
	if claim.PartyIDB != uuid.Nil {
		id := claim.PartyIDB
		partyTwo = &id
	}

	result, err := s.store.CreateTeamFormationBundle(ctx, TeamFormationInput{
		FormationKey: claim.FormationKey,
		Mode:         parts.Canonical,
		Playlist:     parts.Playlist,
		Format:       parts.Format,
		TeamSize:     parts.TeamSize,
		MapID:        s.cfg.DefaultMapID,
		RoundCount:   s.cfg.RoundCount,
		TimerSeconds: timer,
		StartDelay:   s.cfg.StartDelay,
		TeamOne:      append([]uuid.UUID(nil), claim.UserIDsA...),
		TeamTwo:      append([]uuid.UUID(nil), claim.UserIDsB...),
		PartyOneID:   partyOne,
		PartyTwoID:   partyTwo,
		LocationIDs:  locationIDs,
		MatchedAt:    s.clock.Now(),
	})
	if err != nil {
		if errors.Is(err, ErrAccountIneligible) || errors.Is(err, ErrAlreadyAssigned) || errors.Is(err, ErrActiveGameConflict) || errors.Is(err, ErrInvalidRequest) {
			requeueA, reasonA = s.rosterRequeue(ctx, claim.UserIDsA)
			requeueB, reasonB = s.rosterRequeue(ctx, claim.UserIDsB)
			if reasonA == "db_error" || reasonB == "db_error" {
				s.metrics.ObserveFormation("db_error", s.clock.Now().Sub(start))
				return nil, ErrUnavailable
			}
			if releaseErr := s.tickets.ReleaseTeamClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); releaseErr != nil {
				s.recordTeamRedisCleanupFailure(ctx, "release_formation_rejected", claim, releaseErr)
				return nil, ErrUnavailable
			}
			s.metrics.ObserveFormation("formation_failed", s.clock.Now().Sub(start))
			return nil, nil
		}
		s.metrics.ObserveDependencyFailure("postgres")
		s.metrics.ObserveFormation("formation_failed", s.clock.Now().Sub(start))
		s.logger.WarnContext(ctx, "team match formation failed",
			slog.String("formation_key", claim.FormationKey),
			slog.String("outcome", "formation_failed"),
		)
		return nil, ErrUnavailable
	}

	if err := s.tickets.FinalizeTeamClaim(ctx, claim); err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveRecovery("finalize_failed")
		s.logger.WarnContext(ctx, "team match finalize redis failed after durable commit",
			slog.String("formation_key", claim.FormationKey),
			slog.String("match_id", result.Match.ID.String()),
		)
	}

	allUsers := append(append([]uuid.UUID{}, claim.UserIDsA...), claim.UserIDsB...)
	s.notifyFormation(ctx, &result.Match, allUsers, partyOne, partyTwo)
	elapsed := s.clock.Now().Sub(start)
	s.metrics.ObserveFormation("formed", elapsed)
	if parts.Playlist == PlaylistRanked {
		s.metrics.ObserveRankedFormation(parts.Playlist, parts.Format, "formed", elapsed)
	}
	s.logger.InfoContext(ctx, "team match formed",
		slog.String("match_id", result.Match.ID.String()),
		slog.String("game_id", result.Match.GameID.String()),
		slog.String("mode", result.Match.Mode),
		slog.String("outcome", "formed"),
	)
	return NewMatchedStatus(result.Match.ID, result.Match.GameID, result.Match.Mode, result.Match.MatchedAt), nil
}

func (s *Service) notifyFormation(ctx context.Context, match *Match, userIDs []uuid.UUID, partyOne, partyTwo *uuid.UUID) {
	if s == nil || s.notifier == nil || match == nil {
		return
	}
	dest := DestinationPath(match.GameID)
	if match.Mode != ModeRankedStandard {
		dest = MatchDestinationPath(match.ID)
	}
	playlist, format := match.Playlist, match.Format
	if playlist == "" || format == "" {
		if parts, err := ParseMode(match.Mode); err == nil {
			playlist, format = parts.Playlist, parts.Format
		}
	}
	var partyIDs []uuid.UUID
	if partyOne != nil && *partyOne != uuid.Nil {
		partyIDs = append(partyIDs, *partyOne)
	}
	if partyTwo != nil && *partyTwo != uuid.Nil {
		partyIDs = append(partyIDs, *partyTwo)
	}
	_ = s.notifier.NotifyMatchFormed(ctx, MatchAssignmentEvent{
		MatchID:     match.ID,
		GameID:      match.GameID,
		Mode:        match.Mode,
		Playlist:    playlist,
		Format:      format,
		FormedAt:    match.MatchedAt,
		Destination: dest,
		UserIDs:     userIDs,
		PartyIDs:    partyIDs,
	})
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

func (s *Service) recordTeamRedisCleanupFailure(ctx context.Context, operation string, claim *TeamClaim, err error) {
	s.metrics.ObserveDependencyFailure("redis")
	s.metrics.ObserveRecovery(operation + "_failed")
	attrs := []any{slog.String("operation", operation), slog.Any("error", err)}
	if claim != nil {
		attrs = append(attrs, slog.String("formation_key", claim.FormationKey))
	}
	s.logger.WarnContext(ctx, "matchmaking team redis cleanup failed", attrs...)
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

func (s *Service) reconcileExpiredTeamClaims(ctx context.Context) {
	if s.tickets == nil {
		return
	}
	ids, err := s.tickets.ListExpiredTeamClaims(ctx, s.clock.Now(), s.cfg.CandidateScanLimit)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		return
	}
	for _, id := range ids {
		claim, err := s.tickets.GetTeamClaim(ctx, id)
		if err != nil {
			s.metrics.ObserveDependencyFailure("redis")
			return
		}
		if claim == nil {
			if err := s.tickets.DropTeamClaimIndex(ctx, id); err != nil {
				s.recordTeamRedisCleanupFailure(ctx, "drop_orphan_claim_index", nil, err)
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
			if err := s.tickets.FinalizeTeamClaim(ctx, claim); err != nil {
				s.recordTeamRedisCleanupFailure(ctx, "reconcile_finalize", claim, err)
				continue
			}
			s.metrics.ObserveRecovery("reconcile_finalize")
			continue
		}
		requeueA, reasonA := s.rosterRequeue(ctx, claim.UserIDsA)
		requeueB, reasonB := s.rosterRequeue(ctx, claim.UserIDsB)
		if reasonA == "db_error" || reasonB == "db_error" {
			continue
		}
		if err := s.tickets.ReleaseTeamClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
			s.recordTeamRedisCleanupFailure(ctx, "reconcile_release", claim, err)
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

// rosterRequeue is true only when every member of the roster may re-enter the queue.
func (s *Service) rosterRequeue(ctx context.Context, userIDs []uuid.UUID) (requeue bool, reason string) {
	for _, userID := range userIDs {
		ok, why := s.candidateRequeue(ctx, userID)
		if why == "db_error" {
			return false, "db_error"
		}
		if !ok {
			return false, why
		}
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

	// Prefer v2 ticket leave when a ticket is present.
	if s.tickets != nil {
		ticket, err := s.tickets.GetTicketByUser(ctx, userID)
		if err != nil {
			s.metrics.ObserveDependencyFailure("redis")
			return ErrUnavailable
		}
		if ticket != nil {
			if ticket.State == QueueStateClaimed {
				return ErrClaimInProgress
			}
			if err := s.tickets.LeaveTicket(ctx, userID); err != nil {
				if errors.Is(err, ErrClaimInProgress) {
					return ErrClaimInProgress
				}
				s.metrics.ObserveDependencyFailure("redis")
				return ErrUnavailable
			}
			s.publishPartyQueueChanged(ctx, ticket, "idle")
			return nil
		}
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
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrUnsupportedMode), errors.Is(err, ErrInvalidModeSelection):
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
	if _, err := s.tryFormMatch(ctx, ModeRankedStandard); err != nil {
		return nil, err
	}
	// When a v2 ticket exists, attempt formation for that mode.
	if s.tickets != nil {
		if ticket, tErr := s.tickets.GetTicketByUser(ctx, userID); tErr == nil && ticket != nil && ticket.State == QueueStateSearching {
			if _, formErr := s.tryFormTeamMatch(ctx, ticket.Mode); formErr != nil {
				return nil, formErr
			}
		}
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

	return s.statusWithoutDurable(ctx, userID)
}

func (s *Service) statusWithoutDurable(ctx context.Context, userID uuid.UUID) (*StatusResponse, error) {
	// Prefer v2 ticket lease renewal when present.
	if s.tickets != nil {
		ticket, err := s.tickets.RenewTicketLease(ctx, userID, s.cfg.QueueLease, s.clock.Now())
		if err != nil {
			s.metrics.ObserveDependencyFailure("redis")
			// Fall through to v1 — ticket path may be partially degraded.
		} else if ticket != nil {
			if ticket.State == QueueStateClaimed {
				return s.recoverTicketClaimedStatus(ctx, userID, ticket)
			}
			s.metrics.ObserveStatus(PublicStatusSearching)
			return NewSearchingStatusFromTicket(ticket, s.ratingWindowForTicket(ticket)), nil
		}
	}

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

// recoverClaimedStatus resolves durable-first status for a claimed v1 player and cleans expired claims.
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

func (s *Service) recoverTicketClaimedStatus(ctx context.Context, userID uuid.UUID, ticket *QueueTicket) (*StatusResponse, error) {
	if ticket == nil {
		return NewTemporarilyUnavailableStatus(), nil
	}
	if ticket.ClaimID == "" {
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}

	// Durable-first by formation key (claim id is the formation key in v2).
	match, err := s.store.FindMatchByFormationKey(ctx, ticket.ClaimID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if match != nil {
		claim, claimErr := s.tickets.GetTeamClaim(ctx, ticket.ClaimID)
		if claimErr != nil {
			s.recordTeamRedisCleanupFailure(ctx, "get_claim_after_durable_hit", nil, claimErr)
		} else if claim != nil {
			if finalizeErr := s.tickets.FinalizeTeamClaim(ctx, claim); finalizeErr != nil {
				s.recordTeamRedisCleanupFailure(ctx, "finalize_durable_hit", claim, finalizeErr)
			}
		}
		s.metrics.ObserveRecovery("claimed_durable_hit")
		s.metrics.ObserveStatus(PublicStatusMatched)
		return NewMatchedStatus(match.ID, match.GameID, match.Mode, match.MatchedAt), nil
	}

	claim, err := s.tickets.GetTeamClaim(ctx, ticket.ClaimID)
	if err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if claim == nil {
		s.metrics.ObserveRecovery("claim_payload_missing")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	nowMs := s.clock.Now().UTC().UnixMilli()
	if claim.RecoverAfterMs > nowMs {
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}

	match, err = s.store.FindMatchByFormationKey(ctx, claim.FormationKey)
	if err != nil {
		s.metrics.ObserveDependencyFailure("postgres")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if match != nil {
		if finalizeErr := s.tickets.FinalizeTeamClaim(ctx, claim); finalizeErr != nil {
			s.recordTeamRedisCleanupFailure(ctx, "finalize_expired_durable_hit", claim, finalizeErr)
		}
		s.metrics.ObserveRecovery("claimed_durable_hit")
		s.metrics.ObserveStatus(PublicStatusMatched)
		return NewMatchedStatus(match.ID, match.GameID, match.Mode, match.MatchedAt), nil
	}

	requeueA, reasonA := s.rosterRequeue(ctx, claim.UserIDsA)
	requeueB, reasonB := s.rosterRequeue(ctx, claim.UserIDsB)
	if reasonA == "db_error" || reasonB == "db_error" {
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if err := s.tickets.ReleaseTeamClaim(ctx, claim, requeueA, requeueB, s.cfg.QueueLease); err != nil {
		s.metrics.ObserveDependencyFailure("redis")
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	s.metrics.ObserveRecovery("expired_claim_requeued")
	renewed, renewErr := s.tickets.RenewTicketLease(ctx, userID, s.cfg.QueueLease, s.clock.Now())
	if renewErr != nil {
		s.recordTeamRedisCleanupFailure(ctx, "renew_after_release", claim, renewErr)
		s.metrics.ObserveStatus(PublicStatusTemporarilyUnavailable)
		return NewTemporarilyUnavailableStatus(), nil
	}
	if renewed != nil && renewed.State == QueueStateSearching {
		s.metrics.ObserveStatus(PublicStatusSearching)
		return NewSearchingStatusFromTicket(renewed, s.ratingWindowForTicket(renewed)), nil
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

func (s *Service) ensureModeEnabled(parts ModeParts) error {
	if parts.Playlist == PlaylistCasual && !s.cfg.CasualMatchmakingEnabled {
		return ErrUnsupportedMode
	}
	if parts.Playlist == PlaylistRanked && parts.Format != FormatSolo && !s.cfg.RankedTeamModesEnabled {
		return ErrUnsupportedMode
	}
	return nil
}

// useLegacyPairQueue reports whether join should use the v1 pair coordinator.
func (s *Service) useLegacyPairQueue(parts ModeParts) bool {
	if parts.Playlist != PlaylistRanked || parts.Format != FormatSolo {
		return false
	}
	// Ranked solo stays on v1 until team modes (v2 tickets) are enabled and wired.
	if !s.cfg.RankedTeamModesEnabled || s.tickets == nil {
		return true
	}
	return false
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

// mapPartyReaderError converts parties package errors without importing parties (cycle-safe).
func mapPartyReaderError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "dependency") || strings.Contains(msg, "unavailable"):
		return ErrUnavailable
	default:
		return ErrInvalidRequest
	}
}
