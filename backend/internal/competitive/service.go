package competitive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchmaking"
)

// Config holds Elo / placement constants for the competitive service.
type Config struct {
	EloK               int
	AbandonPenalty     int // positive magnitude (stored negative on rows)
	InitialRating      int
	PlacementsRequired int
	// WorkerBatchSize bounds pending finalization retries per tick.
	WorkerBatchSize int
	// SeasonDuration / ResetFactorBPS / Top500MinMatches feed rollover.
	SeasonDuration   time.Duration
	ResetFactorBPS   int
	Top500MinMatches int
}

// DefaultConfig returns v1 competitive constants.
func DefaultConfig() Config {
	return Config{
		EloK:               DefaultEloK,
		AbandonPenalty:     DefaultAbandonPenalty,
		InitialRating:      DefaultInitialRating,
		PlacementsRequired: PlacementsRequired,
		WorkerBatchSize:    50,
		SeasonDuration:     84 * 24 * time.Hour,
		ResetFactorBPS:     DefaultResetFactorBPS,
		Top500MinMatches:   DefaultTop500Min,
	}
}

// store is the persistence surface used by Service.
type store interface {
	ActiveSeason(ctx context.Context) (*Season, error)
	ActiveSeasonID(ctx context.Context) (uuid.UUID, error)
	EnsureStandings(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID, initialRating int, now time.Time) ([]Standing, error)
	StandingsForUsers(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID) ([]Standing, error)
	LoadRatingChanges(ctx context.Context, matchID uuid.UUID) ([]RatingChange, error)
	ListPendingRankedMatchIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	FinalizeMatchProgression(ctx context.Context, in FinalizeInput) (*FinalizeOutcome, error)
}

// MetricsRecorder is the service-facing metrics surface (no user IDs).
type MetricsRecorder interface {
	ObserveFinalization(outcome, format string, duration time.Duration)
	ObserveWorkerSweep(outcome string, processed int)
	ObserveAbandonPenalty()
	ObserveProfileRead(outcome string, duration time.Duration)
	ObserveLeaderboardRead(outcome string, duration time.Duration)
	ObserveRollover(outcome string, seasonSequence int, duration time.Duration)
	ObserveCacheInvalidate(scope string)
}

// NoopMetrics is a no-op MetricsRecorder.
type NoopMetrics struct{}

func (NoopMetrics) ObserveFinalization(string, string, time.Duration) {}
func (NoopMetrics) ObserveWorkerSweep(string, int)                    {}
func (NoopMetrics) ObserveAbandonPenalty()                            {}
func (NoopMetrics) ObserveProfileRead(string, time.Duration)          {}
func (NoopMetrics) ObserveLeaderboardRead(string, time.Duration)      {}
func (NoopMetrics) ObserveRollover(string, int, time.Duration)        {}
func (NoopMetrics) ObserveCacheInvalidate(string)                     {}

// Service implements seasonal Elo finalization and standing reads.
type Service struct {
	repo     store
	cfg      Config
	metrics  MetricsRecorder
	logger   *slog.Logger
	now      func() time.Time
	cache    pageCache
	rollover RolloverConfig
}

// NewService constructs a competitive service.
func NewService(repo store, cfg Config, logger *slog.Logger) *Service {
	if cfg.EloK <= 0 {
		cfg.EloK = DefaultEloK
	}
	if cfg.AbandonPenalty < 0 {
		cfg.AbandonPenalty = -cfg.AbandonPenalty
	}
	if cfg.AbandonPenalty == 0 {
		cfg.AbandonPenalty = DefaultAbandonPenalty
	}
	if cfg.InitialRating < 0 {
		cfg.InitialRating = DefaultInitialRating
	}
	if cfg.PlacementsRequired <= 0 {
		cfg.PlacementsRequired = PlacementsRequired
	}
	if cfg.WorkerBatchSize <= 0 {
		cfg.WorkerBatchSize = 50
	}
	if cfg.SeasonDuration <= 0 {
		cfg.SeasonDuration = 84 * 24 * time.Hour
	}
	if cfg.ResetFactorBPS < 0 {
		cfg.ResetFactorBPS = DefaultResetFactorBPS
	}
	if cfg.Top500MinMatches < 1 {
		cfg.Top500MinMatches = DefaultTop500Min
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:    repo,
		cfg:     cfg,
		metrics: NoopMetrics{},
		logger:  logger,
		now:     func() time.Time { return time.Now().UTC() },
		rollover: RolloverConfig{
			Duration:         cfg.SeasonDuration,
			InitialRating:    cfg.InitialRating,
			ResetFactorBPS:   cfg.ResetFactorBPS,
			Top500MinMatches: cfg.Top500MinMatches,
			EloK:             cfg.EloK,
		},
	}
}

// WithMetrics attaches a metrics recorder.
func (s *Service) WithMetrics(m MetricsRecorder) *Service {
	if s == nil {
		return s
	}
	if m == nil {
		s.metrics = NoopMetrics{}
	} else {
		s.metrics = m
	}
	return s
}

// WithClock overrides the wall clock (tests).
func (s *Service) WithClock(now func() time.Time) *Service {
	if s == nil {
		return s
	}
	if now != nil {
		s.now = now
	}
	return s
}

// Config returns a copy of the service configuration.
func (s *Service) Config() Config {
	if s == nil {
		return DefaultConfig()
	}
	return s.cfg
}

// ActiveSeasonID implements matchmaking.CompetitiveStandingReader.
func (s *Service) ActiveSeasonID(ctx context.Context) (uuid.UUID, error) {
	if s == nil || s.repo == nil {
		return uuid.Nil, ErrDependencyFailure
	}
	return s.repo.ActiveSeasonID(ctx)
}

// StandingsForUsers implements matchmaking.CompetitiveStandingReader.
// Missing standings are created lazily at the season initial rating.
func (s *Service) StandingsForUsers(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID) ([]matchmaking.CompetitiveStandingSnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDependencyFailure
	}
	if seasonID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if len(userIDs) == 0 {
		return nil, nil
	}

	initial := s.cfg.InitialRating
	// Prefer season constants when available.
	if season, err := s.repo.ActiveSeason(ctx); err == nil && season != nil && season.ID == seasonID {
		initial = season.InitialRating
	}

	standings, err := s.repo.EnsureStandings(ctx, seasonID, userIDs, initial, s.now())
	if err != nil {
		return nil, err
	}

	out := make([]matchmaking.CompetitiveStandingSnapshot, 0, len(standings))
	for _, st := range standings {
		snap := matchmaking.CompetitiveStandingSnapshot{
			UserID:              st.UserID,
			SeasonID:            st.SeasonID,
			Rating:              st.Rating,
			PlacementsCompleted: st.PlacementsCompleted,
			NamedRankTier:       NamedRankTier(st.Rating, st.PlacementsCompleted),
		}
		if st.RankVisible() {
			snap.RankCode = RankCodeFromRating(st.Rating)
		}
		out = append(out, snap)
	}
	return out, nil
}

// EnsureReader documents that *Service satisfies matchmaking.CompetitiveStandingReader.
var _ matchmaking.CompetitiveStandingReader = (*Service)(nil)

// FinalizeMatch applies exact-once Elo finalization for a completed Ranked match.
// Idempotent: concurrent/repeated calls return the same stored changes (Replay=true).
// Returns Pending=true with ErrProgressionPending when the match is not yet ready.
func (s *Service) FinalizeMatch(ctx context.Context, matchID uuid.UUID) (*MatchProgressionResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDependencyFailure
	}
	if matchID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	start := s.now()

	outcome, err := s.repo.FinalizeMatchProgression(ctx, FinalizeInput{
		MatchID:        matchID,
		EloK:           s.cfg.EloK,
		AbandonPenalty: s.cfg.AbandonPenalty,
		InitialRating:  s.cfg.InitialRating,
		Now:            start,
	})
	duration := s.now().Sub(start)

	if err != nil {
		outcomeLabel := "error"
		switch {
		case errors.Is(err, ErrMatchNotRanked):
			outcomeLabel = "not_ranked"
		case errors.Is(err, ErrMatchNotTerminal):
			outcomeLabel = "not_terminal"
		case errors.Is(err, ErrMatchNotReady), errors.Is(err, ErrInvalidResult):
			outcomeLabel = "not_ready"
		case errors.Is(err, ErrNoActiveSeason), errors.Is(err, ErrSeasonMismatch):
			outcomeLabel = "season_unavailable"
		case errors.Is(err, ErrProgressionPending):
			outcomeLabel = "pending"
		case errors.Is(err, ErrMatchNotFound):
			outcomeLabel = "not_found"
		}
		s.metrics.ObserveFinalization(outcomeLabel, "", duration)
		if errors.Is(err, ErrMatchNotReady) || errors.Is(err, ErrInvalidResult) {
			return &MatchProgressionResult{MatchID: matchID, Pending: true}, ErrProgressionPending
		}
		return nil, err
	}

	format := outcome.Match.Format
	if outcome.Replay {
		s.metrics.ObserveFinalization("replay", format, duration)
	} else {
		s.metrics.ObserveFinalization("applied", format, duration)
		for _, c := range outcome.Changes {
			if c.AbandonPenalty != 0 {
				s.metrics.ObserveAbandonPenalty()
			}
		}
		// Invalidate leaderboard/profile cache after a newly applied rating change.
		if outcome.Season.ID != uuid.Nil {
			s.InvalidateSeasonCache(ctx, outcome.Season.ID)
		}
	}

	return s.buildResult(outcome), nil
}

// GetMatchProgression returns stored progression for a match, or pending when not finalized.
// Does not apply rating changes; callers that want try-on-read should call FinalizeMatch first.
func (s *Service) GetMatchProgression(ctx context.Context, matchID, userID uuid.UUID) (*PlayerProgressionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDependencyFailure
	}
	changes, err := s.repo.LoadRatingChanges(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		pending := PendingProgression(userID)
		return &pending, ErrProgressionPending
	}
	// Load standings for placement visibility after the match.
	seasonID := changes[0].SeasonID
	userIDs := make([]uuid.UUID, len(changes))
	for i, c := range changes {
		userIDs[i] = c.UserID
	}
	standings, err := s.repo.StandingsForUsers(ctx, seasonID, userIDs)
	if err != nil {
		return nil, err
	}
	placements := map[uuid.UUID]int{}
	for _, st := range standings {
		placements[st.UserID] = st.PlacementsCompleted
	}

	for _, c := range changes {
		if c.UserID == userID {
			dto := ProjectPlayerProgression(c, placements[userID])
			return &dto, nil
		}
	}
	pending := PendingProgression(userID)
	return &pending, ErrProgressionPending
}

// GetMatchProgressionMap returns visibility-aware progression for every participant.
func (s *Service) GetMatchProgressionMap(ctx context.Context, matchID uuid.UUID) (map[uuid.UUID]PlayerProgressionDTO, error) {
	if s == nil || s.repo == nil {
		return nil, ErrDependencyFailure
	}
	changes, err := s.repo.LoadRatingChanges(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, ErrProgressionPending
	}
	seasonID := changes[0].SeasonID
	userIDs := make([]uuid.UUID, len(changes))
	for i, c := range changes {
		userIDs[i] = c.UserID
	}
	standings, err := s.repo.StandingsForUsers(ctx, seasonID, userIDs)
	if err != nil {
		return nil, err
	}
	placements := map[uuid.UUID]int{}
	for _, st := range standings {
		placements[st.UserID] = st.PlacementsCompleted
	}
	out := make(map[uuid.UUID]PlayerProgressionDTO, len(changes))
	for _, c := range changes {
		out[c.UserID] = ProjectPlayerProgression(c, placements[c.UserID])
	}
	return out, nil
}

// RetryPendingFinalizations processes a bounded batch of completed Ranked matches
// missing progression_finalized_at. Errors on individual matches are logged and counted.
func (s *Service) RetryPendingFinalizations(ctx context.Context) (processed, applied, failed int, err error) {
	if s == nil || s.repo == nil {
		return 0, 0, 0, ErrDependencyFailure
	}
	ids, err := s.repo.ListPendingRankedMatchIDs(ctx, s.cfg.WorkerBatchSize)
	if err != nil {
		s.metrics.ObserveWorkerSweep("list_error", 0)
		return 0, 0, 0, err
	}
	if len(ids) == 0 {
		s.metrics.ObserveWorkerSweep("empty", 0)
		return 0, 0, 0, nil
	}
	for _, id := range ids {
		processed++
		res, ferr := s.FinalizeMatch(ctx, id)
		if ferr != nil {
			failed++
			if s.logger != nil {
				s.logger.Warn("competitive progression retry failed",
					"match_id", id.String(),
					"error", ferr.Error(),
				)
			}
			continue
		}
		if res != nil && !res.Pending {
			applied++
		}
	}
	outcome := "ok"
	if failed > 0 && applied == 0 {
		outcome = "all_failed"
	} else if failed > 0 {
		outcome = "partial"
	}
	s.metrics.ObserveWorkerSweep(outcome, processed)
	return processed, applied, failed, nil
}

func (s *Service) buildResult(outcome *FinalizeOutcome) *MatchProgressionResult {
	if outcome == nil {
		return nil
	}
	placements := map[uuid.UUID]int{}
	for _, st := range outcome.Standings {
		placements[st.UserID] = st.PlacementsCompleted
	}
	// When standings were not reloaded with updated placement counts (replay path
	// loads current standings), prefer inferring from changes if missing.
	byUser := make(map[uuid.UUID]PlayerProgressionDTO, len(outcome.Changes))
	for _, c := range outcome.Changes {
		p := placements[c.UserID]
		if p == 0 {
			// Infer: if new rank code present, placements complete; else use old+1 heuristic.
			if c.NewRankCode != nil {
				p = PlacementsRequired
			} else if c.OldRankCode == nil {
				// Still hidden — unknown exact count; treat as incomplete for projection.
				p = PlacementsRequired - 1
			} else {
				p = PlacementsRequired
			}
		}
		byUser[c.UserID] = ProjectPlayerProgression(c, p)
	}
	var seasonID uuid.UUID
	if outcome.Match.SeasonID != nil {
		seasonID = *outcome.Match.SeasonID
	} else {
		seasonID = outcome.Season.ID
	}
	return &MatchProgressionResult{
		MatchID:     outcome.Match.ID,
		SeasonID:    seasonID,
		Replay:      outcome.Replay,
		Pending:     false,
		FinalizedAt: outcome.FinalizedAt,
		Changes:     outcome.Changes,
		ByUser:      byUser,
	}
}

// StandingSnapshots converts domain standings to package snapshots (non-matchmaking).
func StandingSnapshots(standings []Standing) []StandingSnapshot {
	out := make([]StandingSnapshot, 0, len(standings))
	for _, st := range standings {
		snap := StandingSnapshot{
			UserID:              st.UserID,
			SeasonID:            st.SeasonID,
			Rating:              st.Rating,
			PlacementsCompleted: st.PlacementsCompleted,
			NamedRankTier:       NamedRankTier(st.Rating, st.PlacementsCompleted),
		}
		if st.RankVisible() {
			snap.RankCode = RankCodeFromRating(st.Rating)
		}
		out = append(out, snap)
	}
	return out
}

// ComputeTeamRatingChanges is a pure helper for unit tests and dry-run diagnostics.
// It does not persist. abandoners maps userID → abandoned.
func ComputeTeamRatingChanges(
	k, abandonMagnitude int,
	teamOne, teamTwo map[uuid.UUID]int, // user → pre-match rating
	teamOneOutcome, teamTwoOutcome string,
	abandoners map[uuid.UUID]bool,
) (map[uuid.UUID]struct {
	BaseDelta, AbandonPenalty, OldRating, NewRating, TotalDelta int
}, error) {
	if len(teamOne) == 0 || len(teamTwo) == 0 {
		return nil, fmt.Errorf("%w: empty team", ErrInvalidInput)
	}
	ratings1 := make([]int, 0, len(teamOne))
	for _, r := range teamOne {
		ratings1 = append(ratings1, r)
	}
	ratings2 := make([]int, 0, len(teamTwo))
	for _, r := range teamTwo {
		ratings2 = append(ratings2, r)
	}
	avg1 := TeamAverage(ratings1)
	avg2 := TeamAverage(ratings2)
	base1 := BaseDeltaForAverages(k, avg1, avg2, teamOneOutcome)
	base2 := BaseDeltaForAverages(k, avg2, avg1, teamTwoOutcome)

	out := make(map[uuid.UUID]struct {
		BaseDelta, AbandonPenalty, OldRating, NewRating, TotalDelta int
	}, len(teamOne)+len(teamTwo))

	apply := func(users map[uuid.UUID]int, base int) {
		for uid, old := range users {
			pen := AbandonPenaltyValue(abandoners[uid], abandonMagnitude)
			newR, total := ApplyRating(old, base, pen)
			out[uid] = struct {
				BaseDelta, AbandonPenalty, OldRating, NewRating, TotalDelta int
			}{base, pen, old, newR, total}
		}
	}
	apply(teamOne, base1)
	apply(teamTwo, base2)
	return out, nil
}
