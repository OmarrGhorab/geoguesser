package competitive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Season rollover advisory lock key (stable int64 pair for pg_advisory_xact_lock).
const (
	seasonRolloverLockClass = 712045 // arbitrary fixed class id
	seasonRolloverLockObj   = 1
)

// RolloverConfig configures automatic season closure.
type RolloverConfig struct {
	// Duration is the length of the next season.
	Duration time.Duration
	// InitialRating / ResetFactorBPS / Top500Min / EloK seed the next season.
	InitialRating    int
	ResetFactorBPS   int
	Top500MinMatches int
	EloK             int
}

// DefaultRolloverConfig returns v1 84-day season constants.
func DefaultRolloverConfig() RolloverConfig {
	return RolloverConfig{
		Duration:         84 * 24 * time.Hour,
		InitialRating:    DefaultInitialRating,
		ResetFactorBPS:   DefaultResetFactorBPS,
		Top500MinMatches: DefaultTop500Min,
		EloK:             DefaultEloK,
	}
}

// RolloverOutcome describes the result of a rollover attempt.
type RolloverOutcome struct {
	// Applied is true when a season was closed and a new one created.
	Applied bool
	// Skipped is true when no active season had ended yet.
	Skipped bool
	// Replay is true when another worker already rolled over (idempotent).
	Replay       bool
	ClosedSeason *Season
	NewSeason    *Season
}

// RollOverIfDue closes an expired active season under an advisory transaction
// lock, freezes final/ending/peak ranks for eligible standings, and opens the
// next season. Concurrent callers either wait and observe a replay or skip.
func (r *Repository) RollOverIfDue(ctx context.Context, cfg RolloverConfig, now time.Time) (*RolloverOutcome, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	cfg = normalizeRolloverConfig(cfg)
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var out RolloverOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := acquireSeasonRolloverLock(tx); err != nil {
			return err
		}

		var active Season
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", SeasonStatusActive).
			Take(&active).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			out.Skipped = true
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock active season for rollover: %w", err)
		}

		// Not due yet.
		if active.EndsAt.After(now) {
			out.Skipped = true
			return nil
		}

		// Freeze eligible ordered standings (scan at most Top500Limit eligible for
		// final_position assignment; all standings get ending/peak codes).
		if err := freezeSeasonStandings(tx, active, now); err != nil {
			return err
		}

		closedAt := now
		if err := tx.Model(&Season{}).Where("id = ? AND status = ?", active.ID, SeasonStatusActive).
			Updates(map[string]any{
				"status":     SeasonStatusClosed,
				"closed_at":  closedAt,
				"updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("close season: %w", err)
		}
		active.Status = SeasonStatusClosed
		active.ClosedAt = &closedAt
		active.UpdatedAt = now

		// Create next season. Unique sequence/slug + one-active index make races fail closed.
		nextSeq := active.Sequence + 1
		startsAt := active.EndsAt
		if startsAt.Before(now) {
			// Catch-up: start immediately when the prior window already ended.
			startsAt = now
		}
		endsAt := startsAt.Add(cfg.Duration)
		next := Season{
			ID:               uuid.New(),
			Sequence:         nextSeq,
			Slug:             fmt.Sprintf("season-%d", nextSeq),
			Name:             fmt.Sprintf("Season %d", nextSeq),
			Status:           SeasonStatusActive,
			StartsAt:         startsAt,
			EndsAt:           endsAt,
			InitialRating:    cfg.InitialRating,
			EloK:             cfg.EloK,
			ResetFactorBPS:   cfg.ResetFactorBPS,
			Top500MinMatches: cfg.Top500MinMatches,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&next).Error; err != nil {
			if isUniqueViolation(err) {
				// Concurrent rollover created the next season; treat as replay.
				out.Replay = true
				out.ClosedSeason = &active
				return nil
			}
			return fmt.Errorf("create next season: %w", err)
		}

		out.Applied = true
		out.ClosedSeason = &active
		out.NewSeason = &next
		return nil
	})
	if err != nil {
		// Unique conflict outside the create path (one-active index) → reload.
		if isUniqueViolation(err) {
			return r.loadRolloverReplay(ctx)
		}
		return nil, err
	}
	if out.Replay && out.NewSeason == nil {
		// Best-effort attach current active season.
		if active, aerr := r.ActiveSeason(ctx); aerr == nil {
			out.NewSeason = active
		}
	}
	return &out, nil
}

func (r *Repository) loadRolloverReplay(ctx context.Context) (*RolloverOutcome, error) {
	active, err := r.ActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	return &RolloverOutcome{Replay: true, NewSeason: active}, nil
}

func acquireSeasonRolloverLock(tx *gorm.DB) error {
	// Transaction-scoped advisory lock; released on commit/rollback.
	if err := tx.Exec(`SELECT pg_advisory_xact_lock(?, ?)`, seasonRolloverLockClass, seasonRolloverLockObj).Error; err != nil {
		return fmt.Errorf("season rollover advisory lock: %w", err)
	}
	return nil
}

func freezeSeasonStandings(tx *gorm.DB, season Season, now time.Time) error {
	minMatches := season.Top500MinMatches
	if minMatches < 1 {
		minMatches = DefaultTop500Min
	}

	// Load all standings with user status for this season.
	type freezeRow struct {
		UserID              uuid.UUID
		Rating              int
		PeakRating          int
		PlacementsCompleted int
		MatchesPlayed       int
		Wins                int
		RatingReachedAt     time.Time
		UserStatus          string
	}
	var rows []freezeRow
	if err := tx.Raw(`
		SELECT s.user_id, s.rating, s.peak_rating, s.placements_completed, s.matches_played,
		       s.wins, s.rating_reached_at, u.status AS user_status
		FROM competitive_standings s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.season_id = ?
		ORDER BY s.rating DESC, s.wins DESC, s.rating_reached_at ASC, s.user_id ASC
	`, season.ID).Scan(&rows).Error; err != nil {
		return fmt.Errorf("load standings for freeze: %w", err)
	}

	// Assign final positions only to eligible good-standing users in order.
	position := 0
	for _, row := range rows {
		var finalPos *int
		eligible := row.UserStatus == "active" &&
			EligibleForTop500(row.Rating, row.PlacementsCompleted, row.MatchesPlayed, minMatches)
		if eligible {
			position++
			p := position
			finalPos = &p
		}
		ending := EndingRankCode(row.Rating, row.PlacementsCompleted, finalPos)
		peak := PeakRankCode(row.PeakRating, row.PlacementsCompleted, finalPos)

		updates := map[string]any{
			"updated_at": now,
		}
		if finalPos != nil {
			updates["final_position"] = *finalPos
		}
		if ending != nil {
			updates["ending_rank_code"] = *ending
		}
		if peak != nil {
			updates["peak_rank_code"] = *peak
		}
		if err := tx.Model(&Standing{}).
			Where("season_id = ? AND user_id = ?", season.ID, row.UserID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("freeze standing: %w", err)
		}
	}
	return nil
}

func normalizeRolloverConfig(cfg RolloverConfig) RolloverConfig {
	if cfg.Duration <= 0 {
		cfg.Duration = 84 * 24 * time.Hour
	}
	if cfg.InitialRating < 0 {
		cfg.InitialRating = DefaultInitialRating
	}
	if cfg.ResetFactorBPS < 0 {
		cfg.ResetFactorBPS = 0
	}
	if cfg.ResetFactorBPS > 10000 {
		cfg.ResetFactorBPS = 10000
	}
	if cfg.Top500MinMatches < 1 {
		cfg.Top500MinMatches = DefaultTop500Min
	}
	if cfg.EloK < 1 {
		cfg.EloK = DefaultEloK
	}
	return cfg
}

// SeasonSummary projects a Season into the API summary shape.
func SeasonSummary(s Season) SeasonSummaryDTO {
	factorPct := s.ResetFactorBPS / 100
	return SeasonSummaryDTO{
		ID:       s.ID,
		Sequence: s.Sequence,
		Slug:     s.Slug,
		Name:     s.Name,
		Status:   s.Status,
		StartsAt: s.StartsAt.UTC(),
		EndsAt:   s.EndsAt.UTC(),
		ClosedAt: s.ClosedAt,
		Reset: SeasonResetDTO{
			Anchor:        s.InitialRating,
			FactorPercent: factorPct,
		},
		Top500MinMatches: s.Top500MinMatches,
	}
}

// RankDetailFromCode builds a RankDetailDTO from a stored rank code.
func RankDetailFromCode(code string) *RankDetailDTO {
	if code == "" {
		return nil
	}
	if code == WorldLegendCode {
		return &RankDetailDTO{Code: WorldLegendCode, NameKey: WorldLegendNameKey, Division: nil}
	}
	if d, ok := DivisionByCode(code); ok {
		div := d.Division
		return &RankDetailDTO{Code: d.Code, NameKey: d.NameKey, Division: &div}
	}
	return &RankDetailDTO{Code: code, NameKey: code, Division: nil}
}
