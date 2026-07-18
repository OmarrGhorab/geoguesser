package matchmaking

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProgressionFinalizeInput is the matchmaking-local payload for competitive finalization.
// Mirrors games.TerminalMatchResult without importing games (cycle-safe for adapters).
type ProgressionFinalizeInput struct {
	GameID          uuid.UUID
	MatchID         uuid.UUID
	Result          string
	WinnerTeamSlot  *int
	TeamOneScore    int
	TeamTwoScore    int
	CompletedAt     time.Time
	IsRanked        bool
	AbandonedUserID *uuid.UUID
	AbandonReason   string
}

// ProgressionFinalizer applies exact-once competitive rating changes inside a transaction.
// Implementations live in the competitive package; nil means leave progression pending.
type ProgressionFinalizer interface {
	FinalizeInTx(ctx context.Context, tx *gorm.DB, input ProgressionFinalizeInput) error
}

// ErrProgressionDeferred signals a retryable progression failure (match stays terminal).
var ErrProgressionDeferred = errors.New("competitive progression deferred")

// MatchLifecycleAdapter finalizes durable match results and optional competitive progression.
// It implements the games RankedLifecycleHook and MatchLifecycleHook surfaces via methods
// that match those interfaces (wired from main without matchmaking importing games).
type MatchLifecycleAdapter struct {
	repo        *Repository
	progression ProgressionFinalizer
	logger      *slog.Logger
	metrics     MetricsRecorder
}

// NewMatchLifecycleAdapter constructs a unified match/competitive lifecycle hook.
func NewMatchLifecycleAdapter(repo *Repository) *MatchLifecycleAdapter {
	return &MatchLifecycleAdapter{
		repo:    repo,
		logger:  slog.Default(),
		metrics: NoopMetrics{},
	}
}

// WithProgression attaches a competitive finalizer (optional).
func (a *MatchLifecycleAdapter) WithProgression(p ProgressionFinalizer) *MatchLifecycleAdapter {
	a.progression = p
	return a
}

// WithLogger overrides the logger.
func (a *MatchLifecycleAdapter) WithLogger(logger *slog.Logger) *MatchLifecycleAdapter {
	if logger != nil {
		a.logger = logger
	}
	return a
}

// WithMetrics attaches metrics observers.
func (a *MatchLifecycleAdapter) WithMetrics(m MetricsRecorder) *MatchLifecycleAdapter {
	if m != nil {
		a.metrics = m
	}
	return a
}

// RankedLifecycleAdapter returns a thin adapter implementing the legacy ranked-only hooks
// so existing WithRankedLifecycle wiring keeps working.
func (a *MatchLifecycleAdapter) RankedLifecycleAdapter() *RankedLifecycleAdapter {
	if a == nil || a.repo == nil {
		return NewRankedLifecycleAdapter(nil)
	}
	return NewRankedLifecycleAdapter(a.repo)
}

// --- games.RankedLifecycleHook surface ---

// OnRankedGameStarted promotes matched → active.
func (a *MatchLifecycleAdapter) OnRankedGameStarted(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	return a.ApplyMatchActiveInTxStandalone(ctx, gameID, at)
}

// OnRankedGameCompleted completes the match and attempts progression.
func (a *MatchLifecycleAdapter) OnRankedGameCompleted(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return a.ApplyGameCompletedInTx(ctx, tx, gameID, at)
	})
}

// OnRankedGameCancelled cancels/fails the match.
func (a *MatchLifecycleAdapter) OnRankedGameCancelled(ctx context.Context, gameID uuid.UUID, at time.Time, failureCode string) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return a.ApplyGameCancelledInTx(ctx, tx, gameID, at, failureCode)
	})
}

// ApplyStartedInTx is the RankedLifecycleHook Tx method.
func (a *MatchLifecycleAdapter) ApplyStartedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	return a.ApplyMatchActiveInTx(ctx, tx, gameID, at)
}

// ApplyCompletedInTx is the RankedLifecycleHook Tx method.
func (a *MatchLifecycleAdapter) ApplyCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	return a.ApplyGameCompletedInTx(ctx, tx, gameID, at)
}

// ApplyCancelledInTx is the RankedLifecycleHook Tx method.
func (a *MatchLifecycleAdapter) ApplyCancelledInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error {
	return a.ApplyGameCancelledInTx(ctx, tx, gameID, at, failureCode)
}

// --- games.MatchLifecycleHook surface ---

// OnMatchActive best-effort post-commit activation.
func (a *MatchLifecycleAdapter) OnMatchActive(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	return a.ApplyMatchActiveInTxStandalone(ctx, gameID, at)
}

// OnGameCompleted best-effort post-commit completion.
func (a *MatchLifecycleAdapter) OnGameCompleted(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	return a.OnRankedGameCompleted(ctx, gameID, at)
}

// OnGameCancelled best-effort post-commit cancel.
func (a *MatchLifecycleAdapter) OnGameCancelled(ctx context.Context, gameID uuid.UUID, at time.Time, failureCode string) error {
	return a.OnRankedGameCancelled(ctx, gameID, at, failureCode)
}

// ApplyMatchActiveInTx activates the match inside an existing game transaction.
func (a *MatchLifecycleAdapter) ApplyMatchActiveInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	match, err := matchByGameID(ctx, tx, gameID)
	if err != nil || match == nil {
		return err
	}
	if match.Status == MatchStatusActive || IsTerminalMatch(match.Status) {
		return nil
	}
	return transitionMatchTx(tx, match.ID, MatchStatusMatched, MatchStatusActive, at)
}

// ApplyGameCompletedInTx completes the match, writes team result when missing, and
// attempts competitive finalization. Progression failures leave pending (retryable)
// without rolling back match completion.
func (a *MatchLifecycleAdapter) ApplyGameCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	at = at.UTC()
	match, err := matchByGameIDForUpdate(ctx, tx, gameID)
	if err != nil || match == nil {
		return err
	}

	// Promote matched → active when needed, then active → completed.
	if match.Status == MatchStatusMatched {
		if err := transitionMatchTx(tx, match.ID, MatchStatusMatched, MatchStatusActive, at); err != nil {
			return err
		}
		match.Status = MatchStatusActive
	}
	if match.Status == MatchStatusCompleted {
		// Idempotent replay: still try deferred progression if pending.
		return a.tryFinalizeProgression(ctx, tx, match, at, nil)
	}
	if match.Status != MatchStatusActive {
		return nil
	}

	// Fill team scores/result from game_players when not already set (forfeit paths set them earlier).
	if match.Result == nil {
		one, two, result, winner, scoreErr := loadTeamOutcome(tx, match.GameID)
		if scoreErr != nil {
			return scoreErr
		}
		chatUntil := at.Add(15 * time.Minute)
		updates := map[string]any{
			"team_one_score":    one,
			"team_two_score":    two,
			"result":            result,
			"winner_team_slot":  winner,
			"status":            MatchStatusCompleted,
			"completed_at":      at,
			"chat_access_until": chatUntil,
			"last_activity_at":  at,
			"updated_at":        at,
		}
		if err := tx.Model(&Match{}).Where("id = ? AND status = ?", match.ID, MatchStatusActive).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&MatchPlayer{}).
			Where("match_id = ? AND status IN ?", match.ID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
			Updates(map[string]any{
				"status":       ParticipantStatusCompleted,
				"completed_at": at,
				"closed_at":    at,
			}).Error; err != nil {
			return err
		}
		match.TeamOneScore = one
		match.TeamTwoScore = two
		match.Result = &result
		match.WinnerTeamSlot = winner
		match.Status = MatchStatusCompleted
		match.CompletedAt = &at
		match.ChatAccessUntil = &chatUntil
	} else {
		if err := transitionMatchTx(tx, match.ID, MatchStatusActive, MatchStatusCompleted, at); err != nil {
			return err
		}
		match.Status = MatchStatusCompleted
		match.CompletedAt = &at
	}

	a.metrics.ObserveFormation("completed", 0)
	return a.tryFinalizeProgression(ctx, tx, match, at, nil)
}

// ApplyGameCancelledInTx cancels or fails the match.
func (a *MatchLifecycleAdapter) ApplyGameCancelledInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	if len(failureCode) > 64 {
		return ErrInvalidRequest
	}
	match, err := matchByGameID(ctx, tx, gameID)
	if err != nil || match == nil {
		return err
	}
	if IsTerminalMatch(match.Status) {
		return nil
	}
	to := MatchStatusCancelled
	if failureCode != "" {
		to = MatchStatusFailedToStart
	}
	return transitionMatchWithFailureTx(tx, match.ID, match.Status, to, at, failureCode)
}

// ApplyTerminalResultInTx implements games.TerminalResultApplier.
// Writes the games-computed terminal outcome then attempts progression.
func (a *MatchLifecycleAdapter) ApplyTerminalResultInTx(ctx context.Context, tx *gorm.DB, result games.TerminalMatchResult) error {
	return a.applyTerminalResult(ctx, tx, ProgressionFinalizeInput{
		GameID:          result.GameID,
		Result:          result.Result,
		WinnerTeamSlot:  result.WinnerTeamSlot,
		TeamOneScore:    result.TeamOneScore,
		TeamTwoScore:    result.TeamTwoScore,
		CompletedAt:     result.CompletedAt,
		IsRanked:        result.IsRanked,
		AbandonedUserID: result.AbandonedUserID,
		AbandonReason:   result.AbandonReason,
	})
}

// applyTerminalResult writes match scores/result and attempts progression finalization.
func (a *MatchLifecycleAdapter) applyTerminalResult(ctx context.Context, tx *gorm.DB, input ProgressionFinalizeInput) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	at := input.CompletedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	match, err := matchByGameIDForUpdate(ctx, tx, input.GameID)
	if err != nil || match == nil {
		return err
	}
	if match.Status == MatchStatusMatched {
		if err := transitionMatchTx(tx, match.ID, MatchStatusMatched, MatchStatusActive, at); err != nil {
			return err
		}
		match.Status = MatchStatusActive
	}
	if IsTerminalMatch(match.Status) && match.ProgressionFinalizedAt != nil {
		return nil
	}

	chatUntil := at.Add(15 * time.Minute)
	resultCode := input.Result
	updates := map[string]any{
		"team_one_score":    input.TeamOneScore,
		"team_two_score":    input.TeamTwoScore,
		"result":            resultCode,
		"winner_team_slot":  input.WinnerTeamSlot,
		"status":            MatchStatusCompleted,
		"completed_at":      at,
		"chat_access_until": chatUntil,
		"last_activity_at":  at,
		"updated_at":        at,
	}
	if err := tx.Model(&Match{}).Where("id = ?", match.ID).Updates(updates).Error; err != nil {
		return err
	}
	if err := tx.Model(&MatchPlayer{}).
		Where("match_id = ? AND status IN ?", match.ID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
		Updates(map[string]any{
			"status":       ParticipantStatusCompleted,
			"completed_at": at,
			"closed_at":    at,
		}).Error; err != nil {
		return err
	}
	match.TeamOneScore = input.TeamOneScore
	match.TeamTwoScore = input.TeamTwoScore
	match.Result = &resultCode
	match.WinnerTeamSlot = input.WinnerTeamSlot
	match.Status = MatchStatusCompleted
	match.CompletedAt = &at
	match.ChatAccessUntil = &chatUntil

	return a.tryFinalizeProgression(ctx, tx, match, at, &progressionContext{
		abandonedUserID: input.AbandonedUserID,
		abandonReason:   input.AbandonReason,
	})
}

type progressionContext struct {
	abandonedUserID *uuid.UUID
	abandonReason   string
}

func (a *MatchLifecycleAdapter) tryFinalizeProgression(ctx context.Context, tx *gorm.DB, match *Match, at time.Time, pctx *progressionContext) error {
	if match == nil {
		return nil
	}
	if match.ProgressionFinalizedAt != nil {
		a.metrics.ObserveRecovery("progression_replay")
		return nil
	}
	// Casual never finalizes competitive progression.
	if match.Playlist == PlaylistCasual || IsCasual(match.Mode) {
		return nil
	}
	if a.progression == nil {
		// Pending until competitive is injected / worker retries.
		a.metrics.ObserveRecovery("progression_pending")
		return nil
	}
	if match.Result == nil {
		a.metrics.ObserveRecovery("progression_pending")
		return nil
	}

	input := ProgressionFinalizeInput{
		GameID:         match.GameID,
		MatchID:        match.ID,
		Result:         *match.Result,
		WinnerTeamSlot: match.WinnerTeamSlot,
		TeamOneScore:   match.TeamOneScore,
		TeamTwoScore:   match.TeamTwoScore,
		CompletedAt:    at,
		IsRanked:       true,
	}
	if pctx != nil {
		input.AbandonedUserID = pctx.abandonedUserID
		input.AbandonReason = pctx.abandonReason
	} else {
		// Load abandoner from match_players when present (single-quitter forfeit).
		var abandoner uuid.UUID
		var reason string
		row := tx.WithContext(ctx).Raw(`
			SELECT user_id, COALESCE(abandon_reason, '') FROM match_players
			WHERE match_id = ? AND abandoned_at IS NOT NULL
			ORDER BY abandoned_at ASC, user_id ASC
			LIMIT 1
		`, match.ID).Row()
		if row != nil {
			_ = row.Scan(&abandoner, &reason)
			if abandoner != uuid.Nil {
				input.AbandonedUserID = &abandoner
				input.AbandonReason = reason
			}
		}
	}

	if err := a.progression.FinalizeInTx(ctx, tx, input); err != nil {
		// Do not fail the match transaction — leave progression_finalized_at null for retry.
		if a.logger != nil {
			a.logger.WarnContext(ctx, "competitive progression deferred",
				slog.String("match_id", match.ID.String()),
				slog.String("game_id", match.GameID.String()),
				slog.Any("error", err),
			)
		}
		a.metrics.ObserveRecovery("progression_pending")
		a.metrics.ObserveDependencyFailure("competitive")
		return nil
	}

	if err := tx.Model(&Match{}).Where("id = ? AND progression_finalized_at IS NULL", match.ID).
		Updates(map[string]any{
			"progression_finalized_at": at,
			"updated_at":               at,
		}).Error; err != nil {
		return err
	}
	a.metrics.ObserveRecovery("progression_finalized")
	return nil
}

func (a *MatchLifecycleAdapter) ApplyMatchActiveInTxStandalone(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return a.ApplyMatchActiveInTx(ctx, tx, gameID, at)
	})
}

func matchByGameID(ctx context.Context, tx *gorm.DB, gameID uuid.UUID) (*Match, error) {
	var match Match
	if err := tx.WithContext(ctx).Where("game_id = ?", gameID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

func matchByGameIDForUpdate(ctx context.Context, tx *gorm.DB, gameID uuid.UUID) (*Match, error) {
	var match Match
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("game_id = ?", gameID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

func loadTeamOutcome(tx *gorm.DB, gameID uuid.UUID) (one, two int, result string, winner *int, err error) {
	type row struct {
		TeamSlot   *int
		TotalScore int
	}
	var rows []row
	if err = tx.Table("game_players").
		Select("team_slot, total_score").
		Where("game_id = ?", gameID).
		Find(&rows).Error; err != nil {
		return 0, 0, "", nil, err
	}
	for _, r := range rows {
		if r.TeamSlot == nil {
			continue
		}
		switch *r.TeamSlot {
		case 1:
			one += r.TotalScore
		case 2:
			two += r.TotalScore
		}
	}
	if one > two {
		slot := 1
		return one, two, MatchResultTeamOneWin, &slot, nil
	}
	if two > one {
		slot := 2
		return one, two, MatchResultTeamTwoWin, &slot, nil
	}
	return one, two, MatchResultDraw, nil, nil
}
