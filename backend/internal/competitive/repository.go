package competitive

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository persists seasons, standings, and exact-once rating changes.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a competitive repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB exposes the underlying handle for tests.
func (r *Repository) DB() *gorm.DB {
	if r == nil {
		return nil
	}
	return r.db
}

// ActiveSeason returns the single active competitive season, or nil when none.
func (r *Repository) ActiveSeason(ctx context.Context) (*Season, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	var season Season
	err := r.db.WithContext(ctx).
		Where("status = ?", SeasonStatusActive).
		Take(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active season: %w", err)
	}
	return &season, nil
}

// ActiveSeasonID returns the active season id or uuid.Nil when none.
func (r *Repository) ActiveSeasonID(ctx context.Context) (uuid.UUID, error) {
	season, err := r.ActiveSeason(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if season == nil {
		return uuid.Nil, nil
	}
	return season.ID, nil
}

// EnsureStandings creates missing standings at initialRating for the given users.
// Existing rows are left unchanged. Returns standings in stable user_id order.
func (r *Repository) EnsureStandings(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID, initialRating int, now time.Time) ([]Standing, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if seasonID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	ids := uniqueSortedUserIDs(userIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	if initialRating < 0 {
		initialRating = 0
	}
	now = now.UTC()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, uid := range ids {
			res := tx.Exec(`
				INSERT INTO competitive_standings (
					season_id, user_id, rating, placements_completed,
					matches_played, wins, losses, draws, abandons,
					peak_rating, rating_reached_at, created_at, updated_at
				) VALUES (?, ?, ?, 0, 0, 0, 0, 0, 0, ?, ?, ?, ?)
				ON CONFLICT (season_id, user_id) DO NOTHING
			`, seasonID, uid, initialRating, initialRating, now, now, now)
			if res.Error != nil {
				return fmt.Errorf("ensure standing: %w", res.Error)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.listStandings(ctx, seasonID, ids)
}

// StandingsForUsers returns standings for the given users (missing rows omitted).
// Order is stable by user_id ascending.
func (r *Repository) StandingsForUsers(ctx context.Context, seasonID uuid.UUID, userIDs []uuid.UUID) ([]Standing, error) {
	ids := uniqueSortedUserIDs(userIDs)
	return r.listStandings(ctx, seasonID, ids)
}

func (r *Repository) listStandings(ctx context.Context, seasonID uuid.UUID, sortedUserIDs []uuid.UUID) ([]Standing, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if len(sortedUserIDs) == 0 {
		return nil, nil
	}
	var rows []Standing
	err := r.db.WithContext(ctx).
		Where("season_id = ? AND user_id IN ?", seasonID, sortedUserIDs).
		Order("user_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list standings: %w", err)
	}
	return rows, nil
}

// LoadRatingChanges returns stored rating changes for a match in user_id order.
func (r *Repository) LoadRatingChanges(ctx context.Context, matchID uuid.UUID) ([]RatingChange, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	var rows []RatingChange
	err := r.db.WithContext(ctx).
		Where("match_id = ?", matchID).
		Order("user_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load rating changes: %w", err)
	}
	return rows, nil
}

// ListPendingRankedMatchIDs returns completed Ranked matches lacking progression finalization.
// Bounded by limit; ordered by completed_at ascending for fairness.
func (r *Repository) ListPendingRankedMatchIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if limit <= 0 {
		limit = 50
	}
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT id
		FROM matches
		WHERE status = 'completed'
		  AND playlist = 'ranked'
		  AND progression_finalized_at IS NULL
		  AND result IS NOT NULL
		ORDER BY completed_at ASC NULLS LAST, id ASC
		LIMIT ?
	`, limit).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("list pending ranked matches: %w", err)
	}
	return ids, nil
}

// FinalizeInput configures exact-once rating finalization.
type FinalizeInput struct {
	MatchID        uuid.UUID
	EloK           int
	AbandonPenalty int // positive magnitude
	InitialRating  int
	Now            time.Time
}

// FinalizeOutcome is the durable result of finalization (applied or replay).
type FinalizeOutcome struct {
	Match       MatchTerminal
	Season      Season
	Changes     []RatingChange
	Standings   []Standing
	Replay      bool
	FinalizedAt time.Time
}

// FinalizeMatchProgression locks the match, active season, and standings in sorted
// user order; inserts unique rating changes; updates standings; and sets
// progression_finalized_at atomically. Concurrent callers either wait on the
// match lock and replay, or treat unique conflicts as idempotent replay.
func (r *Repository) FinalizeMatchProgression(ctx context.Context, in FinalizeInput) (*FinalizeOutcome, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if in.MatchID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if in.EloK <= 0 {
		in.EloK = DefaultEloK
	}
	if in.AbandonPenalty < 0 {
		in.AbandonPenalty = -in.AbandonPenalty
	}
	if in.AbandonPenalty == 0 {
		in.AbandonPenalty = DefaultAbandonPenalty
	}
	if in.InitialRating < 0 {
		in.InitialRating = DefaultInitialRating
	}
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var out FinalizeOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match, err := lockMatch(tx, in.MatchID)
		if err != nil {
			return err
		}
		if err := validateMatchForProgression(match); err != nil {
			return err
		}

		// Idempotent replay: already finalized.
		if match.ProgressionFinalizedAt != nil {
			changes, loadErr := loadChangesTx(tx, match.ID)
			if loadErr != nil {
				return loadErr
			}
			standings, stErr := loadStandingsForChanges(tx, changes)
			if stErr != nil {
				return stErr
			}
			season, sErr := loadSeasonTx(tx, *match.SeasonID)
			if sErr != nil {
				return sErr
			}
			out = FinalizeOutcome{
				Match:       *match,
				Season:      *season,
				Changes:     changes,
				Standings:   standings,
				Replay:      true,
				FinalizedAt: match.ProgressionFinalizedAt.UTC(),
			}
			return nil
		}

		// Lock active season (prevents rollover races during finalization).
		season, err := lockActiveSeason(tx)
		if err != nil {
			return err
		}
		if match.SeasonID == nil || *match.SeasonID != season.ID {
			// Prefer the match's season if it is still valid and active was swapped;
			// for v1 require match season == active.
			return ErrSeasonMismatch
		}

		parts, err := loadParticipantsTx(tx, match.ID)
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return ErrMatchNotReady
		}

		userIDs := make([]uuid.UUID, len(parts))
		for i, p := range parts {
			userIDs[i] = p.UserID
		}
		sortedIDs := uniqueSortedUserIDs(userIDs)

		// Ensure standings exist, then lock in stable user order.
		if err := ensureStandingsTx(tx, season.ID, sortedIDs, in.InitialRating, now); err != nil {
			return err
		}
		standings, err := lockStandingsInOrder(tx, season.ID, sortedIDs)
		if err != nil {
			return err
		}
		standingByUser := make(map[uuid.UUID]*Standing, len(standings))
		for i := range standings {
			standingByUser[standings[i].UserID] = &standings[i]
		}
		for _, id := range sortedIDs {
			if _, ok := standingByUser[id]; !ok {
				return fmt.Errorf("%w: missing standing for %s", ErrMatchNotReady, id)
			}
		}

		// Team averages from pre-match ratings.
		var team1, team2 []int
		for _, p := range parts {
			st := standingByUser[p.UserID]
			switch p.TeamSlot {
			case TeamSlotOne:
				team1 = append(team1, st.Rating)
			case TeamSlotTwo:
				team2 = append(team2, st.Rating)
			}
		}
		if len(team1) == 0 || len(team2) == 0 {
			return ErrMatchNotReady
		}
		avg1 := TeamAverage(team1)
		avg2 := TeamAverage(team2)

		result := ""
		if match.Result != nil {
			result = *match.Result
		}

		// Shared base delta per team (symmetric Elo from team averages).
		outcome1, ok1 := OutcomeForTeam(result, TeamSlotOne, match.WinnerTeamSlot)
		outcome2, ok2 := OutcomeForTeam(result, TeamSlotTwo, match.WinnerTeamSlot)
		if !ok1 || !ok2 {
			return ErrInvalidResult
		}
		base1 := BaseDeltaForAverages(in.EloK, avg1, avg2, outcome1)
		base2 := BaseDeltaForAverages(in.EloK, avg2, avg1, outcome2)
		exp1 := ExpectedBPS(avg1, avg2)
		exp2 := ExpectedBPS(avg2, avg1)

		changes := make([]RatingChange, 0, len(parts))
		// Process participants in sorted user order for deterministic inserts.
		sort.Slice(parts, func(i, j int) bool {
			return bytesLessUUID(parts[i].UserID, parts[j].UserID)
		})

		for _, p := range parts {
			st := standingByUser[p.UserID]
			var (
				outcome string
				ownAvg  int
				oppAvg  int
				base    int
				expBPS  int
			)
			if p.TeamSlot == TeamSlotOne {
				outcome, ownAvg, oppAvg, base, expBPS = outcome1, avg1, avg2, base1, exp1
			} else {
				outcome, ownAvg, oppAvg, base, expBPS = outcome2, avg2, avg1, base2, exp2
			}
			abandoned := p.AbandonedAt != nil
			penalty := AbandonPenaltyValue(abandoned, in.AbandonPenalty)
			oldRating := st.Rating
			newRating, total := ApplyRating(oldRating, base, penalty)
			placementsAfter := NextPlacements(st.PlacementsCompleted)
			// Old rank visible only if already past placements before this match.
			oldCode := OptionalRankCode(oldRating, st.PlacementsCompleted)
			newCode := OptionalRankCode(newRating, placementsAfter)

			change := RatingChange{
				ID:                  uuid.New(),
				SeasonID:            season.ID,
				MatchID:             match.ID,
				UserID:              p.UserID,
				TeamSlot:            p.TeamSlot,
				Outcome:             outcome,
				OwnTeamAverage:      ownAvg,
				OpponentTeamAverage: oppAvg,
				ExpectedBPS:         expBPS,
				BaseDelta:           base,
				AbandonPenalty:      penalty,
				OldRating:           oldRating,
				NewRating:           newRating,
				TotalDelta:          total,
				OldRankCode:         oldCode,
				NewRankCode:         newCode,
				CreatedAt:           now,
			}
			if err := tx.Create(&change).Error; err != nil {
				// Unique (match_id, user_id) → concurrent finalizer won; treat as replay.
				if isUniqueViolation(err) {
					return errIdempotentReplay
				}
				return fmt.Errorf("insert rating change: %w", err)
			}
			changes = append(changes, change)

			// Update standing in memory then persist.
			st.Rating = newRating
			st.PlacementsCompleted = placementsAfter
			st.MatchesPlayed++
			switch outcome {
			case OutcomeWin:
				st.Wins++
			case OutcomeLoss:
				st.Losses++
			case OutcomeDraw:
				st.Draws++
			}
			if abandoned {
				st.Abandons++
			}
			if newRating > st.PeakRating {
				st.PeakRating = newRating
			}
			if total != 0 {
				st.RatingReachedAt = now
			}
			matchID := match.ID
			st.LastMatchID = &matchID
			st.UpdatedAt = now

			if err := tx.Model(&Standing{}).
				Where("season_id = ? AND user_id = ?", st.SeasonID, st.UserID).
				Updates(map[string]any{
					"rating":               st.Rating,
					"placements_completed": st.PlacementsCompleted,
					"matches_played":       st.MatchesPlayed,
					"wins":                 st.Wins,
					"losses":               st.Losses,
					"draws":                st.Draws,
					"abandons":             st.Abandons,
					"peak_rating":          st.PeakRating,
					"rating_reached_at":    st.RatingReachedAt,
					"last_match_id":        st.LastMatchID,
					"updated_at":           st.UpdatedAt,
				}).Error; err != nil {
				return fmt.Errorf("update standing: %w", err)
			}
		}

		// Atomic match finalization marker.
		if err := tx.Exec(`
			UPDATE matches
			SET progression_finalized_at = ?, updated_at = ?
			WHERE id = ? AND progression_finalized_at IS NULL
		`, now, now, match.ID).Error; err != nil {
			return fmt.Errorf("set progression_finalized_at: %w", err)
		}

		// Confirm the marker stuck (lost race → replay path).
		var finalizedAt *time.Time
		if err := tx.Raw(`SELECT progression_finalized_at FROM matches WHERE id = ?`, match.ID).
			Scan(&finalizedAt).Error; err != nil {
			return fmt.Errorf("reload progression_finalized_at: %w", err)
		}
		if finalizedAt == nil {
			return errIdempotentReplay
		}

		match.ProgressionFinalizedAt = finalizedAt
		out = FinalizeOutcome{
			Match:       *match,
			Season:      *season,
			Changes:     changes,
			Standings:   standingsSnapshot(standingByUser, sortedIDs),
			Replay:      false,
			FinalizedAt: finalizedAt.UTC(),
		}
		return nil
	})

	if errors.Is(err, errIdempotentReplay) {
		// Another worker finalized; return stored state.
		return r.loadReplayOutcome(ctx, in.MatchID)
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// errIdempotentReplay is an internal signal for unique-conflict / lost races.
var errIdempotentReplay = errors.New("idempotent progression replay")

func (r *Repository) loadReplayOutcome(ctx context.Context, matchID uuid.UUID) (*FinalizeOutcome, error) {
	var match MatchTerminal
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, game_id, mode, playlist, format, team_size, season_id, status, result,
		       winner_team_slot, progression_finalized_at, completed_at
		FROM matches WHERE id = ?
	`, matchID).Scan(&match).Error
	if err != nil {
		return nil, fmt.Errorf("reload match for replay: %w", err)
	}
	if match.ID == uuid.Nil {
		return nil, ErrMatchNotFound
	}
	if match.ProgressionFinalizedAt == nil {
		// Still not finalized — surface as pending retry.
		return nil, ErrProgressionPending
	}
	changes, err := r.LoadRatingChanges(ctx, matchID)
	if err != nil {
		return nil, err
	}
	var season Season
	if match.SeasonID != nil {
		if err := r.db.WithContext(ctx).Where("id = ?", *match.SeasonID).Take(&season).Error; err != nil {
			return nil, fmt.Errorf("reload season for replay: %w", err)
		}
	}
	standings, err := loadStandingsForChangeList(r.db.WithContext(ctx), changes)
	if err != nil {
		return nil, err
	}
	return &FinalizeOutcome{
		Match:       match,
		Season:      season,
		Changes:     changes,
		Standings:   standings,
		Replay:      true,
		FinalizedAt: match.ProgressionFinalizedAt.UTC(),
	}, nil
}

func lockMatch(tx *gorm.DB, matchID uuid.UUID) (*MatchTerminal, error) {
	var match MatchTerminal
	err := tx.Raw(`
		SELECT id, game_id, mode, playlist, format, team_size, season_id, status, result,
		       winner_team_slot, progression_finalized_at, completed_at
		FROM matches
		WHERE id = ?
		FOR UPDATE
	`, matchID).Scan(&match).Error
	if err != nil {
		return nil, fmt.Errorf("lock match: %w", err)
	}
	if match.ID == uuid.Nil {
		return nil, ErrMatchNotFound
	}
	return &match, nil
}

func validateMatchForProgression(m *MatchTerminal) error {
	if m == nil {
		return ErrMatchNotFound
	}
	if m.Playlist != "ranked" && !isRankedMode(m.Mode) {
		return ErrMatchNotRanked
	}
	if m.Status != "completed" {
		return ErrMatchNotTerminal
	}
	if m.Result == nil || *m.Result == "" || *m.Result == "cancelled" {
		return ErrMatchNotReady
	}
	if m.SeasonID == nil {
		return ErrMatchNotReady
	}
	return nil
}

func isRankedMode(mode string) bool {
	switch mode {
	case "ranked_solo", "ranked_duo", "ranked_squad", "ranked_standard":
		return true
	default:
		return false
	}
}

func lockActiveSeason(tx *gorm.DB) (*Season, error) {
	var season Season
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status = ?", SeasonStatusActive).
		Take(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoActiveSeason
	}
	if err != nil {
		return nil, fmt.Errorf("lock active season: %w", err)
	}
	return &season, nil
}

func loadSeasonTx(tx *gorm.DB, seasonID uuid.UUID) (*Season, error) {
	var season Season
	err := tx.Where("id = ?", seasonID).Take(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoActiveSeason
	}
	if err != nil {
		return nil, fmt.Errorf("load season: %w", err)
	}
	return &season, nil
}

func loadParticipantsTx(tx *gorm.DB, matchID uuid.UUID) ([]MatchParticipantTerminal, error) {
	var parts []MatchParticipantTerminal
	err := tx.Raw(`
		SELECT match_id, user_id, team_slot, abandoned_at, abandon_reason, status
		FROM match_players
		WHERE match_id = ?
		ORDER BY user_id ASC
	`, matchID).Scan(&parts).Error
	if err != nil {
		return nil, fmt.Errorf("load participants: %w", err)
	}
	return parts, nil
}

func ensureStandingsTx(tx *gorm.DB, seasonID uuid.UUID, sortedUserIDs []uuid.UUID, initialRating int, now time.Time) error {
	for _, uid := range sortedUserIDs {
		if err := tx.Exec(`
			INSERT INTO competitive_standings (
				season_id, user_id, rating, placements_completed,
				matches_played, wins, losses, draws, abandons,
				peak_rating, rating_reached_at, created_at, updated_at
			) VALUES (?, ?, ?, 0, 0, 0, 0, 0, 0, ?, ?, ?, ?)
			ON CONFLICT (season_id, user_id) DO NOTHING
		`, seasonID, uid, initialRating, initialRating, now, now, now).Error; err != nil {
			return fmt.Errorf("ensure standing tx: %w", err)
		}
	}
	return nil
}

func lockStandingsInOrder(tx *gorm.DB, seasonID uuid.UUID, sortedUserIDs []uuid.UUID) ([]Standing, error) {
	out := make([]Standing, 0, len(sortedUserIDs))
	for _, uid := range sortedUserIDs {
		var st Standing
		err := tx.Raw(`
			SELECT season_id, user_id, rating, placements_completed, matches_played,
			       wins, losses, draws, abandons, peak_rating, rating_reached_at,
			       last_match_id, final_position, ending_rank_code, peak_rank_code,
			       created_at, updated_at
			FROM competitive_standings
			WHERE season_id = ? AND user_id = ?
			FOR UPDATE
		`, seasonID, uid).Scan(&st).Error
		if err != nil {
			return nil, fmt.Errorf("lock standing: %w", err)
		}
		if st.UserID == uuid.Nil {
			return nil, fmt.Errorf("%w: standing missing for %s", ErrMatchNotReady, uid)
		}
		out = append(out, st)
	}
	return out, nil
}

func loadChangesTx(tx *gorm.DB, matchID uuid.UUID) ([]RatingChange, error) {
	var rows []RatingChange
	err := tx.Where("match_id = ?", matchID).Order("user_id ASC").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load changes tx: %w", err)
	}
	return rows, nil
}

func loadStandingsForChanges(tx *gorm.DB, changes []RatingChange) ([]Standing, error) {
	return loadStandingsForChangeList(tx, changes)
}

func loadStandingsForChangeList(db *gorm.DB, changes []RatingChange) ([]Standing, error) {
	if len(changes) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(changes))
	seasonID := changes[0].SeasonID
	for i, c := range changes {
		ids[i] = c.UserID
	}
	ids = uniqueSortedUserIDs(ids)
	var rows []Standing
	err := db.Where("season_id = ? AND user_id IN ?", seasonID, ids).
		Order("user_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load standings for changes: %w", err)
	}
	return rows, nil
}

func standingsSnapshot(byUser map[uuid.UUID]*Standing, sortedIDs []uuid.UUID) []Standing {
	out := make([]Standing, 0, len(sortedIDs))
	for _, id := range sortedIDs {
		if st, ok := byUser[id]; ok && st != nil {
			out = append(out, *st)
		}
	}
	return out
}

func uniqueSortedUserIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return bytesLessUUID(out[i], out[j])
	})
	return out
}

func bytesLessUUID(a, b uuid.UUID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// Avoid hard dependency on pgconn; match common driver messages / codes.
	msg := err.Error()
	return containsAny(msg, "duplicate key", "unique constraint", "SQLSTATE 23505", "23505")
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if stringContainsFold(s, p) {
			return true
		}
	}
	return false
}

func stringContainsFold(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		indexFold(s, substr) >= 0)
}

func indexFold(s, substr string) int {
	// Simple case-sensitive contains is enough for SQLSTATE/driver strings.
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
