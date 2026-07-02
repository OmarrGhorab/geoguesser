package leaderboards

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/challenges"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository owns leaderboard definitions, materialized entries, and source-fact reads.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureGlobalLeaderboard(ctx context.Context) (*Leaderboard, error) {
	board := Leaderboard{
		Kind:        KindGlobal,
		ScopeKey:    "all",
		DisplayName: "Global Leaderboard",
		Status:      StatusActive,
		RankingRule: RankingRuleBestScore,
	}
	if err := r.ensureLeaderboard(ctx, &board); err != nil {
		return nil, err
	}
	return r.getLeaderboard(ctx, KindGlobal, "all")
}

func (r *Repository) EnsureMapLeaderboard(ctx context.Context, mapID uuid.UUID) (*Leaderboard, error) {
	var exists struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id
		FROM maps
		WHERE id = ? AND status = 'active' AND visibility = 'public'
	`, mapID).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check map leaderboard scope: %w", err)
	}
	if exists.ID == uuid.Nil {
		return nil, nil
	}

	board := Leaderboard{
		Kind:        KindMap,
		ScopeKey:    mapID.String(),
		DisplayName: "Map Leaderboard",
		Status:      StatusActive,
		RankingRule: RankingRuleBestScore,
		MapID:       &mapID,
	}
	if err := r.ensureLeaderboard(ctx, &board); err != nil {
		return nil, err
	}
	return r.getLeaderboard(ctx, KindMap, mapID.String())
}

func (r *Repository) GetDailyChallengeByDate(ctx context.Context, date time.Time) (*challenges.Challenge, error) {
	var challenge challenges.Challenge
	if err := r.db.WithContext(ctx).Where("type = ? AND challenge_date = ?", challenges.TypeDaily, date.Format("2006-01-02")).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get daily challenge leaderboard scope: %w", err)
	}
	return &challenge, nil
}

func (r *Repository) ListGeneralEntries(ctx context.Context, leaderboardID uuid.UUID, limit int, cursor string) ([]Entry, error) {
	parsedCursor, err := decodeCursor(cursor)
	if err := wrapInvalidCursor(err); err != nil {
		return nil, err
	}
	rankOffset := 0
	if parsedCursor != nil {
		rankOffset, err = r.countGeneralEntriesThroughCursor(ctx, leaderboardID, *parsedCursor)
		if err != nil {
			return nil, err
		}
	}
	query := `
		SELECT
			id,
			leaderboard_id,
			game_id,
			user_id,
			display_name_snapshot,
			score,
			games_played,
			completion_duration_ms,
			completed_at,
			(? + ROW_NUMBER() OVER (
				ORDER BY score DESC, completion_duration_ms ASC NULLS LAST, completed_at ASC, user_id ASC
			))::INT AS rank,
			created_at,
			updated_at
		FROM leaderboard_entries
		WHERE leaderboard_id = ?
	`
	args := []any{rankOffset, leaderboardID}
	if parsedCursor != nil {
		query += ` AND ` + seekAfterPredicate("user_id")
		args = append(args, cursorSortValues(*parsedCursor)...)
	}
	query += ` ORDER BY rank ASC, user_id ASC LIMIT ?`
	args = append(args, limit)

	var entries []Entry
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&entries).Error; err != nil {
		return nil, fmt.Errorf("list leaderboard entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) ListDailyEntries(ctx context.Context, challengeID uuid.UUID, limit int, cursor string) ([]challenges.LeaderboardEntry, error) {
	parsedCursor, err := decodeCursor(cursor)
	if err := wrapInvalidCursor(err); err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Where("challenge_id = ?", challengeID)
	if parsedCursor != nil {
		query = query.Where(seekAfterPredicate("attempt_id"), cursorSortValues(*parsedCursor)...)
	}
	var entries []challenges.LeaderboardEntry
	if err := query.Order("score DESC, completion_duration_ms ASC NULLS LAST, completed_at ASC, attempt_id ASC").Limit(limit).Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list daily leaderboard entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) countGeneralEntriesThroughCursor(ctx context.Context, leaderboardID uuid.UUID, cursor leaderboardCursor) (int, error) {
	query := `SELECT COUNT(*) FROM leaderboard_entries WHERE leaderboard_id = ? AND ` + seekThroughPredicate("user_id")
	args := append([]any{leaderboardID}, cursorSortValues(cursor)...)
	var count int64
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("count leaderboard entries through cursor: %w", err)
	}
	return int(count), nil
}

func (r *Repository) MaterializeCompletedGame(ctx context.Context, gameID uuid.UUID) ([]uuid.UUID, error) {
	var touched []uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		candidates, err := r.completedCandidatesTx(tx, gameID)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return nil
		}

		global, err := r.ensureLeaderboardTx(tx, Leaderboard{
			Kind:        KindGlobal,
			ScopeKey:    "all",
			DisplayName: "Global Leaderboard",
			Status:      StatusActive,
			RankingRule: RankingRuleBestScore,
		})
		if err != nil {
			return err
		}
		touched = append(touched, global.ID)
		touchedSet := map[uuid.UUID]struct{}{global.ID: {}}

		for _, candidate := range candidates {
			if err := r.upsertBestEntryTx(tx, *global, candidate); err != nil {
				return err
			}
			mapBoard, err := r.ensureLeaderboardTx(tx, Leaderboard{
				Kind:        KindMap,
				ScopeKey:    candidate.MapID.String(),
				DisplayName: "Map Leaderboard",
				Status:      StatusActive,
				RankingRule: RankingRuleBestScore,
				MapID:       &candidate.MapID,
			})
			if err != nil {
				return err
			}
			if err := r.upsertBestEntryTx(tx, *mapBoard, candidate); err != nil {
				return err
			}
			if _, ok := touchedSet[mapBoard.ID]; !ok {
				touched = append(touched, mapBoard.ID)
				touchedSet[mapBoard.ID] = struct{}{}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return touched, nil
}

func (r *Repository) DailyCacheScopeForGame(ctx context.Context, gameID uuid.UUID) (*string, error) {
	var row struct {
		ChallengeDate *time.Time `gorm:"column:challenge_date"`
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT c.challenge_date
		FROM challenge_attempts ca
		JOIN challenges c ON c.id = ca.challenge_id
		WHERE ca.game_id = ?
		  AND c.type = ?
		  AND c.challenge_date IS NOT NULL
		LIMIT 1
	`, gameID, challenges.TypeDaily).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("get daily leaderboard cache scope: %w", err)
	}
	if row.ChallengeDate == nil {
		return nil, nil
	}
	scope := "daily:" + row.ChallengeDate.UTC().Format("2006-01-02")
	return &scope, nil
}

func (r *Repository) ensureLeaderboard(ctx context.Context, board *Leaderboard) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "kind"}, {Name: "scope_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":       StatusActive,
			"ranking_rule": board.RankingRule,
			"map_id":       board.MapID,
			"challenge_id": board.ChallengeID,
		}),
	}).Create(board).Error
}

func (r *Repository) getLeaderboard(ctx context.Context, kind string, scopeKey string) (*Leaderboard, error) {
	var board Leaderboard
	if err := r.db.WithContext(ctx).Where("kind = ? AND scope_key = ?", kind, scopeKey).First(&board).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeaderboardNotFound
		}
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	return &board, nil
}

func (r *Repository) ensureLeaderboardTx(tx *gorm.DB, board Leaderboard) (*Leaderboard, error) {
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "kind"}, {Name: "scope_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":       StatusActive,
			"ranking_rule": board.RankingRule,
			"map_id":       board.MapID,
			"challenge_id": board.ChallengeID,
		}),
	}).Create(&board).Error; err != nil {
		return nil, fmt.Errorf("ensure leaderboard: %w", err)
	}
	var loaded Leaderboard
	if err := tx.Where("kind = ? AND scope_key = ?", board.Kind, board.ScopeKey).First(&loaded).Error; err != nil {
		return nil, fmt.Errorf("load ensured leaderboard: %w", err)
	}
	return &loaded, nil
}

func (r *Repository) completedCandidatesTx(tx *gorm.DB, gameID uuid.UUID) ([]completedGameCandidate, error) {
	var candidates []completedGameCandidate
	if err := tx.Raw(`
		SELECT
			g.id AS game_id,
			g.map_id,
			gp.user_id,
			COALESCE(NULLIF(up.display_name, ''), gp.display_name) AS display_name,
			gp.total_score AS score,
			CASE
				WHEN g.started_at IS NOT NULL AND g.completed_at IS NOT NULL
				THEN (EXTRACT(EPOCH FROM (g.completed_at - g.started_at)) * 1000)::BIGINT
				ELSE NULL
			END AS completion_duration_ms,
			g.completed_at
		FROM games g
		JOIN game_players gp ON gp.game_id = g.id
		JOIN users u ON u.id = gp.user_id
		JOIN user_profiles up ON up.user_id = u.id
		WHERE g.id = ?
		  AND g.status = 'completed'
		  AND g.completed_at IS NOT NULL
		  AND gp.user_id IS NOT NULL
		  AND gp.status = 'active'
		  AND u.status = 'active'
		ORDER BY gp.user_id ASC
	`, gameID).Scan(&candidates).Error; err != nil {
		return nil, fmt.Errorf("load completed leaderboard candidate: %w", err)
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	return candidates, nil
}

func (r *Repository) upsertBestEntryTx(tx *gorm.DB, board Leaderboard, candidate completedGameCandidate) error {
	best, err := r.bestEntryForScopeTx(tx, board, candidate.UserID)
	if err != nil {
		return err
	}
	if best == nil {
		return nil
	}
	gamesPlayed, err := r.gamesPlayedForScopeTx(tx, board, candidate.UserID)
	if err != nil {
		return err
	}
	entry := Entry{
		LeaderboardID:        board.ID,
		GameID:               best.GameID,
		UserID:               best.UserID,
		DisplayNameSnapshot:  best.DisplayName,
		Score:                best.Score,
		GamesPlayed:          gamesPlayed,
		CompletionDurationMS: best.CompletionDurationMS,
		CompletedAt:          best.CompletedAt,
		Rank:                 1,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "leaderboard_id"}, {Name: "user_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "leaderboard_id IS NOT NULL"},
		}},
		DoUpdates: clause.Assignments(map[string]any{
			"game_id":                gorm.Expr("CASE WHEN leaderboard_entry_is_better(EXCLUDED.score, EXCLUDED.completion_duration_ms, EXCLUDED.completed_at, EXCLUDED.game_id, leaderboard_entries.score, leaderboard_entries.completion_duration_ms, leaderboard_entries.completed_at, leaderboard_entries.game_id) THEN EXCLUDED.game_id ELSE leaderboard_entries.game_id END"),
			"display_name_snapshot":  gorm.Expr("EXCLUDED.display_name_snapshot"),
			"score":                  gorm.Expr("CASE WHEN leaderboard_entry_is_better(EXCLUDED.score, EXCLUDED.completion_duration_ms, EXCLUDED.completed_at, EXCLUDED.game_id, leaderboard_entries.score, leaderboard_entries.completion_duration_ms, leaderboard_entries.completed_at, leaderboard_entries.game_id) THEN EXCLUDED.score ELSE leaderboard_entries.score END"),
			"games_played":           gorm.Expr("EXCLUDED.games_played"),
			"completion_duration_ms": gorm.Expr("CASE WHEN leaderboard_entry_is_better(EXCLUDED.score, EXCLUDED.completion_duration_ms, EXCLUDED.completed_at, EXCLUDED.game_id, leaderboard_entries.score, leaderboard_entries.completion_duration_ms, leaderboard_entries.completed_at, leaderboard_entries.game_id) THEN EXCLUDED.completion_duration_ms ELSE leaderboard_entries.completion_duration_ms END"),
			"completed_at":           gorm.Expr("CASE WHEN leaderboard_entry_is_better(EXCLUDED.score, EXCLUDED.completion_duration_ms, EXCLUDED.completed_at, EXCLUDED.game_id, leaderboard_entries.score, leaderboard_entries.completion_duration_ms, leaderboard_entries.completed_at, leaderboard_entries.game_id) THEN EXCLUDED.completed_at ELSE leaderboard_entries.completed_at END"),
			"updated_at":             time.Now().UTC(),
		}),
	}).Create(&entry).Error
}

func (r *Repository) bestEntryForScopeTx(tx *gorm.DB, board Leaderboard, userID uuid.UUID) (*completedGameCandidate, error) {
	query := `
		SELECT
			g.id AS game_id,
			g.map_id,
			gp.user_id,
			COALESCE(NULLIF(up.display_name, ''), gp.display_name) AS display_name,
			gp.total_score AS score,
			CASE
				WHEN g.started_at IS NOT NULL AND g.completed_at IS NOT NULL
				THEN (EXTRACT(EPOCH FROM (g.completed_at - g.started_at)) * 1000)::BIGINT
				ELSE NULL
			END AS completion_duration_ms,
			g.completed_at
		FROM games g
		JOIN game_players gp ON gp.game_id = g.id
		JOIN users u ON u.id = gp.user_id
		JOIN user_profiles up ON up.user_id = u.id
		WHERE gp.user_id = ?
		  AND gp.status = 'active'
		  AND u.status = 'active'
		  AND g.status = 'completed'
		  AND g.completed_at IS NOT NULL
	`
	args := []any{userID}
	if board.Kind == KindMap && board.MapID != nil {
		query += ` AND g.map_id = ?`
		args = append(args, *board.MapID)
	}
	query += ` ORDER BY gp.total_score DESC, completion_duration_ms ASC NULLS LAST, g.completed_at ASC, g.id ASC LIMIT 1`

	var candidate completedGameCandidate
	if err := tx.Raw(query, args...).Scan(&candidate).Error; err != nil {
		return nil, fmt.Errorf("load best leaderboard entry: %w", err)
	}
	if candidate.GameID == uuid.Nil {
		return nil, nil
	}
	return &candidate, nil
}

func (r *Repository) gamesPlayedForScopeTx(tx *gorm.DB, board Leaderboard, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) AS games_played
		FROM games g
		JOIN game_players gp ON gp.game_id = g.id
		WHERE gp.user_id = ?
		  AND gp.status = 'active'
		  AND g.status = 'completed'
	`
	args := []any{userID}
	if board.Kind == KindMap && board.MapID != nil {
		query += ` AND g.map_id = ?`
		args = append(args, *board.MapID)
	}
	var count int
	if err := tx.Raw(query, args...).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("count leaderboard games played: %w", err)
	}
	return count, nil
}

func seekAfterPredicate(stableColumn string) string {
	return seekPredicate(stableColumn, ">")
}

func seekThroughPredicate(stableColumn string) string {
	return seekPredicate(stableColumn, "<=")
}

func seekPredicate(stableColumn string, operator string) string {
	return fmt.Sprintf(
		"((-score), (completion_duration_ms IS NULL), COALESCE(completion_duration_ms, %d), completed_at, %s) %s (?, ?, ?, ?, ?)",
		nullDurationSortValue,
		stableColumn,
		operator,
	)
}
