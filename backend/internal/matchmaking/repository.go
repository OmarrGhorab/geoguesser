package matchmaking

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

// Repository provides durable matchmaking persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a matchmaking repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ActiveUser holds the minimal eligibility fields for a registered player.
type ActiveUser struct {
	ID     uuid.UUID
	Status string
}

// FindActiveUser returns the user when present. Callers revalidate status.
func (r *Repository) FindActiveUser(ctx context.Context, userID uuid.UUID) (*ActiveUser, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var row struct {
		ID     uuid.UUID `gorm:"column:id"`
		Status string    `gorm:"column:status"`
	}
	err := r.db.WithContext(ctx).
		Table("users").
		Select("id, status").
		Where("id = ?", userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ActiveUser{ID: row.ID, Status: row.Status}, nil
}

// FindActiveAssignment returns a non-terminal ranked assignment for the user, if any.
func (r *Repository) FindActiveAssignment(ctx context.Context, userID uuid.UUID) (*ActiveAssignment, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var row ActiveAssignment
	err := r.db.WithContext(ctx).
		Table("match_players AS mp").
		Select("m.id AS match_id, m.game_id, m.mode, m.status, m.matched_at, mp.user_id").
		Joins("JOIN matches m ON m.id = mp.match_id").
		Where("mp.user_id = ? AND mp.status IN ?", userID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// HasConflictingActiveGame reports whether the user has an active non-ranked multiplayer/solo game
// that should block matchmaking entry.
func (r *Repository) HasConflictingActiveGame(ctx context.Context, userID uuid.UUID) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrUnavailable
	}
	var count int64
	err := r.db.WithContext(ctx).
		Table("game_players AS gp").
		Joins("JOIN games g ON g.id = gp.game_id").
		Where("gp.user_id = ?", userID).
		Where("gp.status IN ?", []string{"active", "disconnected"}).
		Where("g.status IN ?", []string{"pending", "active"}).
		Where("g.mode <> ?", "ranked").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindMatchByFormationKey returns a durable match for formation-key recovery.
func (r *Repository) FindMatchByFormationKey(ctx context.Context, formationKey string) (*Match, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var match Match
	err := r.db.WithContext(ctx).Where("formation_key = ?", formationKey).Take(&match).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

// FormationBundle is the complete durable payload created during pair formation.
type FormationBundle struct {
	Match        *Match
	Participants []MatchPlayer
}

// FormationInput is the complete ranked formation request for one PostgreSQL transaction.
type FormationInput struct {
	FormationKey string
	Mode         string
	MapID        uuid.UUID
	RoundCount   int
	TimerSeconds int
	StartDelay   time.Duration
	UserIDs      [2]uuid.UUID
	LocationIDs  []uuid.UUID
	MatchedAt    time.Time
}

// FormationResult is the durable assignment produced by CreateFormationBundle.
type FormationResult struct {
	Match Match
}

// CreateFormationBundle creates the ranked game, players, rounds, match, and participants atomically.
// On unique formation_key conflict it returns the existing match (idempotent replay).
func (r *Repository) CreateFormationBundle(ctx context.Context, input FormationInput) (*FormationResult, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if input.FormationKey == "" || input.MapID == uuid.Nil || len(input.LocationIDs) < input.RoundCount || input.RoundCount < 1 {
		return nil, ErrContentUnavailable
	}
	if input.UserIDs[0] == uuid.Nil || input.UserIDs[1] == uuid.Nil || input.UserIDs[0] == input.UserIDs[1] {
		return nil, ErrInvalidRequest
	}

	// Stable lock order prevents deadlocks between concurrent formations.
	users := []uuid.UUID{input.UserIDs[0], input.UserIDs[1]}
	sort.Slice(users, func(i, j int) bool { return users[i].String() < users[j].String() })

	var result FormationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Replay path: unique formation key already committed.
		var existing Match
		if err := tx.Where("formation_key = ?", input.FormationKey).Take(&existing).Error; err == nil {
			result.Match = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Lock both users in stable UUID order and revalidate active status.
		// Lock users alone (no outer join) so FOR UPDATE is valid on PostgreSQL.
		displayNames := make(map[uuid.UUID]string, 2)
		for _, userID := range users {
			var row struct {
				ID     uuid.UUID `gorm:"column:id"`
				Status string    `gorm:"column:status"`
			}
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Table("users").
				Select("id, status").
				Where("id = ?", userID).
				Take(&row).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrAccountIneligible
				}
				return err
			}
			if row.Status != "active" {
				return ErrAccountIneligible
			}

			var displayName string
			if err := tx.Table("user_profiles").
				Select("display_name").
				Where("user_id = ?", userID).
				Limit(1).
				Scan(&displayName).Error; err != nil {
				return err
			}
			if displayName == "" {
				displayName = "Player"
			}
			displayNames[userID] = displayName

			var activeCount int64
			if err := tx.Table("match_players").
				Where("user_id = ? AND status IN ?", userID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
				Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount > 0 {
				return ErrAlreadyAssigned
			}

			var conflictCount int64
			if err := tx.Table("game_players AS gp").
				Joins("JOIN games g ON g.id = gp.game_id").
				Where("gp.user_id = ?", userID).
				Where("gp.status IN ?", []string{"active", "disconnected"}).
				Where("g.status IN ?", []string{"pending", "active"}).
				Where("g.mode <> ?", "ranked").
				Count(&conflictCount).Error; err != nil {
				return err
			}
			if conflictCount > 0 {
				return ErrActiveGameConflict
			}
		}

		matchedAt := input.MatchedAt.UTC()
		timerSeconds := input.TimerSeconds
		startsAt := matchedAt.Add(input.StartDelay)
		endsAt := startsAt.Add(time.Duration(timerSeconds) * time.Second)

		// Ranked game is immediately playable with a scheduled first-round countdown.
		gameID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO games (id, mode, status, map_id, round_count, timer_seconds, scoring_version, total_score, started_at, created_at, updated_at)
			VALUES (?, 'ranked', 'active', ?, ?, ?, 1, 0, ?, ?, ?)
		`, gameID, input.MapID, input.RoundCount, timerSeconds, matchedAt, matchedAt, matchedAt).Error; err != nil {
			return fmt.Errorf("create ranked game: %w", err)
		}

		playerIDs := make(map[uuid.UUID]uuid.UUID, 2)
		for _, userID := range []uuid.UUID{input.UserIDs[0], input.UserIDs[1]} {
			playerID := uuid.New()
			name := displayNames[userID]
			if name == "" {
				name = "Player"
			}
			if err := tx.Exec(`
				INSERT INTO game_players (id, game_id, user_id, display_name, role, status, total_score, joined_at)
				VALUES (?, ?, ?, ?, 'player', 'active', 0, ?)
			`, playerID, gameID, userID, name, matchedAt).Error; err != nil {
				return fmt.Errorf("create game player: %w", err)
			}
			playerIDs[userID] = playerID
		}

		for i := 0; i < input.RoundCount; i++ {
			roundID := uuid.New()
			status := "pending"
			var roundStarts, roundEnds *time.Time
			if i == 0 {
				status = "active"
				roundStarts = &startsAt
				roundEnds = &endsAt
			}
			if err := tx.Exec(`
				INSERT INTO rounds (id, game_id, location_id, round_number, status, starts_at, ends_at, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, roundID, gameID, input.LocationIDs[i], i+1, status, roundStarts, roundEnds, matchedAt).Error; err != nil {
				return fmt.Errorf("create round: %w", err)
			}
		}

		// Ranked game is created already-active with a scheduled first round, so the
		// durable match starts as active in the same transaction (lifecycle agreement).
		matchID := uuid.New()
		match := Match{
			ID:           matchID,
			FormationKey: input.FormationKey,
			GameID:       gameID,
			Mode:         input.Mode,
			Status:       MatchStatusActive,
			MatchedAt:    matchedAt,
			StartedAt:    &matchedAt,
			CreatedAt:    matchedAt,
			UpdatedAt:    matchedAt,
		}
		if err := tx.Create(&match).Error; err != nil {
			return fmt.Errorf("create match: %w", err)
		}

		for _, userID := range []uuid.UUID{input.UserIDs[0], input.UserIDs[1]} {
			mp := MatchPlayer{
				MatchID:      matchID,
				UserID:       userID,
				GamePlayerID: playerIDs[userID],
				Status:       ParticipantStatusActive,
				AssignedAt:   matchedAt,
			}
			if err := tx.Create(&mp).Error; err != nil {
				return fmt.Errorf("create match player: %w", err)
			}
		}

		result.Match = match
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TransitionMatch applies a guarded match lifecycle transition and mirrors participant status.
func (r *Repository) TransitionMatch(ctx context.Context, matchID uuid.UUID, fromStatus, toStatus string, at time.Time) error {
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return transitionMatchTx(tx, matchID, fromStatus, toStatus, at)
	})
}

// transitionMatchTx applies a match lifecycle transition inside an existing transaction.
func transitionMatchTx(tx *gorm.DB, matchID uuid.UUID, fromStatus, toStatus string, at time.Time) error {
	return transitionMatchWithFailureTx(tx, matchID, fromStatus, toStatus, at, "")
}

// transitionMatchWithFailureTx persists a terminal failure code with a guarded transition.
func transitionMatchWithFailureTx(tx *gorm.DB, matchID uuid.UUID, fromStatus, toStatus string, at time.Time, failureCode string) error {
	at = at.UTC()
	var match Match
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUnavailable
		}
		return err
	}
	if match.Status != fromStatus {
		// Idempotent replay of already-applied transition.
		if match.Status == toStatus {
			return nil
		}
		return ErrInvalidRequest
	}
	if !validMatchTransition(fromStatus, toStatus) {
		return ErrInvalidRequest
	}

	updates := map[string]any{
		"status":     toStatus,
		"updated_at": at,
	}
	switch toStatus {
	case MatchStatusActive:
		updates["started_at"] = at
	case MatchStatusCompleted:
		updates["completed_at"] = at
	case MatchStatusCancelled, MatchStatusFailedToStart:
		updates["closed_at"] = at
		if toStatus == MatchStatusFailedToStart {
			updates["failure_code"] = failureCode
		}
	}
	if err := tx.Model(&Match{}).Where("id = ? AND status = ?", matchID, fromStatus).Updates(updates).Error; err != nil {
		return err
	}

	participantTo := participantStatusForMatch(toStatus)
	participantUpdates := map[string]any{"status": participantTo}
	switch participantTo {
	case ParticipantStatusCompleted:
		participantUpdates["completed_at"] = at
	case ParticipantStatusCancelled, ParticipantStatusFailed:
		participantUpdates["closed_at"] = at
	}
	return tx.Model(&MatchPlayer{}).
		Where("match_id = ? AND status IN ?", matchID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
		Updates(participantUpdates).Error
}

// ListCompletedRankedResults returns matches eligible for future rating calculation:
// match status completed AND linked game status completed.
func (r *Repository) ListCompletedRankedResults(ctx context.Context, limit int) ([]Match, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 {
		limit = 50
	}
	var matches []Match
	err := r.db.WithContext(ctx).
		Table("matches AS m").
		Select("m.*").
		Joins("JOIN games g ON g.id = m.game_id").
		Where("m.status = ? AND g.status = ?", MatchStatusCompleted, "completed").
		Order("m.completed_at DESC, m.id DESC").
		Limit(limit).
		Find(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func validMatchTransition(from, to string) bool {
	switch from {
	case MatchStatusMatched:
		return to == MatchStatusActive || to == MatchStatusCancelled || to == MatchStatusFailedToStart
	case MatchStatusActive:
		return to == MatchStatusCompleted || to == MatchStatusCancelled || to == MatchStatusFailedToStart
	default:
		return false
	}
}

func participantStatusForMatch(matchStatus string) string {
	switch matchStatus {
	case MatchStatusActive:
		return ParticipantStatusActive
	case MatchStatusCompleted:
		return ParticipantStatusCompleted
	case MatchStatusCancelled:
		return ParticipantStatusCancelled
	case MatchStatusFailedToStart:
		return ParticipantStatusFailed
	default:
		return ParticipantStatusAssigned
	}
}

// RankedLifecycleAdapter implements games.RankedLifecycleHook against durable match transitions.
type RankedLifecycleAdapter struct {
	repo *Repository
}

// NewRankedLifecycleAdapter constructs a games-safe lifecycle callback.
func NewRankedLifecycleAdapter(repo *Repository) *RankedLifecycleAdapter {
	return &RankedLifecycleAdapter{repo: repo}
}

func (a *RankedLifecycleAdapter) OnRankedGameStarted(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	return a.transitionByGameID(ctx, gameID, MatchStatusMatched, MatchStatusActive, at)
}

func (a *RankedLifecycleAdapter) OnRankedGameCompleted(ctx context.Context, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return a.completeInTx(ctx, tx, gameID, at)
	})
}

func (a *RankedLifecycleAdapter) OnRankedGameCancelled(ctx context.Context, gameID uuid.UUID, at time.Time, failureCode string) error {
	if a == nil || a.repo == nil {
		return nil
	}
	return a.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return a.cancelInTx(ctx, tx, gameID, at, failureCode)
	})
}

// ApplyStartedInTx activates the ranked match inside an existing game transaction.
func (a *RankedLifecycleAdapter) ApplyStartedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	return a.startInTx(ctx, tx.WithContext(ctx), gameID, at)
}

// ApplyCompletedInTx completes the ranked match inside an existing game transaction.
func (a *RankedLifecycleAdapter) ApplyCompletedInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	return a.completeInTx(ctx, tx.WithContext(ctx), gameID, at)
}

// ApplyCancelledInTx cancels/fails the ranked match inside an existing game transaction.
func (a *RankedLifecycleAdapter) ApplyCancelledInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error {
	if a == nil || a.repo == nil || tx == nil {
		return nil
	}
	return a.cancelInTx(ctx, tx.WithContext(ctx), gameID, at, failureCode)
}

func (a *RankedLifecycleAdapter) startInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	match, err := a.matchByGameID(ctx, tx, gameID)
	if err != nil || match == nil {
		return err
	}
	// Formation already activates matches when the ranked game is created active.
	// Idempotent no-op when already active/terminal; promotes legacy matched rows.
	if match.Status == MatchStatusActive || IsTerminalMatch(match.Status) {
		return nil
	}
	return transitionMatchTx(tx, match.ID, MatchStatusMatched, MatchStatusActive, at)
}

func (a *RankedLifecycleAdapter) completeInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time) error {
	match, err := a.matchByGameID(ctx, tx, gameID)
	if err != nil || match == nil {
		return err
	}
	if match.Status == MatchStatusCompleted {
		return nil
	}
	if match.Status == MatchStatusMatched {
		if err := transitionMatchTx(tx, match.ID, MatchStatusMatched, MatchStatusActive, at); err != nil {
			return err
		}
	}
	if match.Status != MatchStatusActive && match.Status != MatchStatusMatched {
		return nil
	}
	return transitionMatchTx(tx, match.ID, MatchStatusActive, MatchStatusCompleted, at)
}

func (a *RankedLifecycleAdapter) cancelInTx(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, at time.Time, failureCode string) error {
	if len(failureCode) > 64 {
		return ErrInvalidRequest
	}
	match, err := a.matchByGameID(ctx, tx, gameID)
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

func (a *RankedLifecycleAdapter) matchByGameID(ctx context.Context, tx *gorm.DB, gameID uuid.UUID) (*Match, error) {
	var match Match
	if err := tx.WithContext(ctx).Where("game_id = ?", gameID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}

func (a *RankedLifecycleAdapter) transitionByGameID(ctx context.Context, gameID uuid.UUID, from, to string, at time.Time) error {
	if a == nil || a.repo == nil {
		return nil
	}
	var match Match
	if err := a.repo.db.WithContext(ctx).Where("game_id = ?", gameID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return a.repo.TransitionMatch(ctx, match.ID, from, to, at)
}
