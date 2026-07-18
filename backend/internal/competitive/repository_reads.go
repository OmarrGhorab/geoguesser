package competitive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StandingRow is a standing joined with user profile/status for leaderboard reads.
type StandingRow struct {
	Standing
	DisplayName string `gorm:"column:display_name"`
	UserStatus  string `gorm:"column:user_status"`
}

// HistoryRow is a rating change joined with match playlist/format.
type HistoryRow struct {
	RatingChange
	Playlist string `gorm:"column:playlist"`
	Format   string `gorm:"column:format"`
}

// GetSeason returns a season by id.
func (r *Repository) GetSeason(ctx context.Context, seasonID uuid.UUID) (*Season, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if seasonID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	var season Season
	err := r.db.WithContext(ctx).Where("id = ?", seasonID).Take(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSeasonNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load season: %w", err)
	}
	return &season, nil
}

// ListSeasons returns seasons ordered by sequence descending (newest first).
func (r *Repository) ListSeasons(ctx context.Context, limit int) ([]Season, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if limit <= 0 {
		limit = 50
	}
	var rows []Season
	err := r.db.WithContext(ctx).
		Order("sequence DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list seasons: %w", err)
	}
	return rows, nil
}

// GetStanding returns a single standing or nil when missing.
func (r *Repository) GetStanding(ctx context.Context, seasonID, userID uuid.UUID) (*Standing, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	var st Standing
	err := r.db.WithContext(ctx).
		Where("season_id = ? AND user_id = ?", seasonID, userID).
		Take(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get standing: %w", err)
	}
	return &st, nil
}

// LastRatingChange returns the newest rating change for a user in a season, if any.
func (r *Repository) LastRatingChange(ctx context.Context, seasonID, userID uuid.UUID) (*RatingChange, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	var change RatingChange
	err := r.db.WithContext(ctx).
		Where("season_id = ? AND user_id = ?", seasonID, userID).
		Order("created_at DESC, id DESC").
		Limit(1).
		Take(&change).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("last rating change: %w", err)
	}
	return &change, nil
}

// PreviousClosedSeasonStanding returns the user's standing from the most recent
// closed season excluding excludeSeasonID, if any.
func (r *Repository) PreviousClosedSeasonStanding(ctx context.Context, excludeSeasonID, userID uuid.UUID) (*Standing, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	var st Standing
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.season_id, s.user_id, s.rating, s.placements_completed, s.matches_played,
		       s.wins, s.losses, s.draws, s.abandons, s.peak_rating, s.rating_reached_at,
		       s.last_match_id, s.final_position, s.ending_rank_code, s.peak_rank_code,
		       s.created_at, s.updated_at
		FROM competitive_standings s
		INNER JOIN competitive_seasons cs ON cs.id = s.season_id
		WHERE s.user_id = ?
		  AND cs.status = ?
		  AND cs.id <> ?
		ORDER BY cs.sequence DESC
		LIMIT 1
	`, userID, SeasonStatusClosed, excludeSeasonID).Scan(&st).Error
	if err != nil {
		return nil, fmt.Errorf("previous closed standing: %w", err)
	}
	if st.UserID == uuid.Nil {
		return nil, nil
	}
	return &st, nil
}

// EligibleStandingPosition returns the 1-based position of a user among eligible
// active-season standings, or nil when not eligible / not found.
func (r *Repository) EligibleStandingPosition(ctx context.Context, season Season, userID uuid.UUID) (*int, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	minMatches := season.Top500MinMatches
	if minMatches < 1 {
		minMatches = DefaultTop500Min
	}
	var pos *int
	err := r.db.WithContext(ctx).Raw(`
		WITH eligible_limited AS (
			SELECT s.user_id, s.rating, s.wins, s.rating_reached_at
			FROM competitive_standings s
			INNER JOIN users u ON u.id = s.user_id
			WHERE s.season_id = ?
			  AND s.placements_completed >= ?
			  AND s.matches_played >= ?
			  AND s.rating >= ?
			  AND u.status = 'active'
			ORDER BY s.rating DESC, s.wins DESC, s.rating_reached_at ASC, s.user_id ASC
			LIMIT ?
		), eligible AS (
			SELECT user_id,
			       ROW_NUMBER() OVER (
			           ORDER BY rating DESC, wins DESC, rating_reached_at ASC, user_id ASC
			       ) AS position
			FROM eligible_limited
		)
		SELECT position FROM eligible WHERE user_id = ?
	`, season.ID, PlacementsRequired, minMatches, GeoMaster1MinRating, Top500Limit+1, userID).Scan(&pos).Error
	if err != nil {
		return nil, fmt.Errorf("eligible standing position: %w", err)
	}
	return pos, nil
}

// ListEligibleStandings returns a page of eligible standings in top-500 order.
// limit is the page size (caller may pass limit+1 for hasNext). Cursor is exclusive.
// maxScan caps the absolute number of eligible rows considered (≤501 for top-500).
func (r *Repository) ListEligibleStandings(
	ctx context.Context,
	season Season,
	limit int,
	cursor *leaderboardCursor,
	maxScan int,
) ([]StandingRow, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if limit <= 0 {
		limit = defaultLeaderboardLimit
	}
	if maxScan <= 0 {
		maxScan = Top500Limit + 1
	}
	minMatches := season.Top500MinMatches
	if minMatches < 1 {
		minMatches = DefaultTop500Min
	}

	// Bound the ordered scan to maxScan rows, then apply keyset pagination + limit.
	// This keeps top-500 reads within a ≤501 ordered scan budget.
	sql := `
		SELECT s.season_id, s.user_id, s.rating, s.placements_completed, s.matches_played,
		       s.wins, s.losses, s.draws, s.abandons, s.peak_rating, s.rating_reached_at,
		       s.last_match_id, s.final_position, s.ending_rank_code, s.peak_rank_code,
		       s.created_at, s.updated_at,
		       COALESCE(p.display_name, '') AS display_name,
		       u.status AS user_status
		FROM (
			SELECT s.*
			FROM competitive_standings s
			INNER JOIN users u ON u.id = s.user_id
			WHERE s.season_id = ?
			  AND s.placements_completed >= ?
			  AND s.matches_played >= ?
			  AND s.rating >= ?
			  AND u.status = 'active'
			ORDER BY s.rating DESC, s.wins DESC, s.rating_reached_at ASC, s.user_id ASC
			LIMIT ?
		) s
		INNER JOIN users u ON u.id = s.user_id
		LEFT JOIN user_profiles p ON p.user_id = s.user_id
	`
	args := []any{season.ID, PlacementsRequired, minMatches, GeoMaster1MinRating, maxScan}
	if cursor != nil {
		// Keyset "after" in rating DESC, wins DESC, rating_reached_at ASC, user_id ASC order.
		sql += `
		WHERE (
			s.rating < ?
			OR (s.rating = ? AND s.wins < ?)
			OR (s.rating = ? AND s.wins = ? AND s.rating_reached_at > ?)
			OR (s.rating = ? AND s.wins = ? AND s.rating_reached_at = ? AND s.user_id > ?)
		)`
		args = append(args,
			cursor.Rating,
			cursor.Rating, cursor.Wins,
			cursor.Rating, cursor.Wins, cursor.RatingReachedAt,
			cursor.Rating, cursor.Wins, cursor.RatingReachedAt, cursor.UserID,
		)
	}
	sql += `
		ORDER BY s.rating DESC, s.wins DESC, s.rating_reached_at ASC, s.user_id ASC
		LIMIT ?
	`
	args = append(args, limit)

	var rows []StandingRow
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list eligible standings: %w", err)
	}
	return rows, nil
}

// ListClosedTop500 returns frozen final_position standings for a closed season (positions 1–500).
func (r *Repository) ListClosedTop500(
	ctx context.Context,
	seasonID uuid.UUID,
	limit int,
	cursor *leaderboardCursor,
) ([]StandingRow, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if limit <= 0 {
		limit = defaultLeaderboardLimit
	}
	args := []any{seasonID, Top500Limit}
	sql := `
		SELECT s.season_id, s.user_id, s.rating, s.placements_completed, s.matches_played,
		       s.wins, s.losses, s.draws, s.abandons, s.peak_rating, s.rating_reached_at,
		       s.last_match_id, s.final_position, s.ending_rank_code, s.peak_rank_code,
		       s.created_at, s.updated_at,
		       COALESCE(p.display_name, '') AS display_name,
		       u.status AS user_status
		FROM competitive_standings s
		INNER JOIN users u ON u.id = s.user_id
		LEFT JOIN user_profiles p ON p.user_id = s.user_id
		WHERE s.season_id = ?
		  AND s.final_position IS NOT NULL
		  AND s.final_position <= ?
	`
	if cursor != nil {
		sql += `
		  AND (
			s.rating < ?
			OR (s.rating = ? AND s.wins < ?)
			OR (s.rating = ? AND s.wins = ? AND s.rating_reached_at > ?)
			OR (s.rating = ? AND s.wins = ? AND s.rating_reached_at = ? AND s.user_id > ?)
		  )`
		args = append(args,
			cursor.Rating,
			cursor.Rating, cursor.Wins,
			cursor.Rating, cursor.Wins, cursor.RatingReachedAt,
			cursor.Rating, cursor.Wins, cursor.RatingReachedAt, cursor.UserID,
		)
	}
	sql += `
		ORDER BY s.final_position ASC, s.rating DESC, s.wins DESC, s.rating_reached_at ASC, s.user_id ASC
		LIMIT ?
	`
	args = append(args, limit)

	var rows []StandingRow
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list closed top500: %w", err)
	}
	return rows, nil
}

// ListRatingHistory returns the caller's rating changes newest first.
func (r *Repository) ListRatingHistory(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	cursor *historyCursor,
) ([]HistoryRow, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	args := []any{userID}
	sql := `
		SELECT c.id, c.season_id, c.match_id, c.user_id, c.team_slot, c.outcome,
		       c.own_team_average, c.opponent_team_average, c.expected_bps,
		       c.base_delta, c.abandon_penalty, c.old_rating, c.new_rating, c.total_delta,
		       c.old_rank_code, c.new_rank_code, c.created_at,
		       COALESCE(m.playlist, 'ranked') AS playlist,
		       COALESCE(m.format, 'solo') AS format
		FROM competitive_rating_changes c
		LEFT JOIN matches m ON m.id = c.match_id
		WHERE c.user_id = ?
	`
	if cursor != nil {
		sql += `
		  AND (
			c.created_at < ?
			OR (c.created_at = ? AND c.id < ?)
		  )`
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	sql += `
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT ?
	`
	args = append(args, limit)

	var rows []HistoryRow
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list rating history: %w", err)
	}
	return rows, nil
}

// UserIsActive reports whether the user account is active (good standing for top 500).
func (r *Repository) UserIsActive(ctx context.Context, userID uuid.UUID) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrDependencyFailure
	}
	var status string
	err := r.db.WithContext(ctx).Raw(`SELECT status FROM users WHERE id = ?`, userID).Scan(&status).Error
	if err != nil {
		return false, fmt.Errorf("load user status: %w", err)
	}
	return status == "active", nil
}

// EnsureStandingWithSoftReset creates a standing for user if missing, using soft-reset
// from the previous closed season when available.
func (r *Repository) EnsureStandingWithSoftReset(ctx context.Context, season Season, userID uuid.UUID, now time.Time) (*Standing, error) {
	if r == nil || r.db == nil {
		return nil, ErrDependencyFailure
	}
	if season.ID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	now = now.UTC()
	existing, err := r.GetStanding(ctx, season.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	startRating := season.InitialRating
	if startRating < 0 {
		startRating = DefaultInitialRating
	}
	prev, err := r.PreviousClosedSeasonStanding(ctx, season.ID, userID)
	if err != nil {
		return nil, err
	}
	if prev != nil {
		startRating = SoftResetRating(prev.Rating, season.InitialRating, season.ResetFactorBPS)
	}

	err = r.db.WithContext(ctx).Exec(`
		INSERT INTO competitive_standings (
			season_id, user_id, rating, placements_completed,
			matches_played, wins, losses, draws, abandons,
			peak_rating, rating_reached_at, created_at, updated_at
		) VALUES (?, ?, ?, 0, 0, 0, 0, 0, 0, ?, ?, ?, ?)
		ON CONFLICT (season_id, user_id) DO NOTHING
	`, season.ID, userID, startRating, startRating, now, now, now).Error
	if err != nil {
		return nil, fmt.Errorf("ensure soft-reset standing: %w", err)
	}
	return r.GetStanding(ctx, season.ID, userID)
}
