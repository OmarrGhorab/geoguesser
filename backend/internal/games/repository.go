package games

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository owns solo game persistence.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a repository backed by PostgreSQL.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateGameBundle inserts a game, its solo player, and all rounds in one transaction.
func (r *Repository) CreateGameBundle(ctx context.Context, game *Game, player *GamePlayer, rounds []Round) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(game).Error; err != nil {
			return fmt.Errorf("create game: %w", err)
		}
		player.GameID = game.ID
		if err := tx.Create(player).Error; err != nil {
			return fmt.Errorf("create game player: %w", err)
		}
		for i := range rounds {
			rounds[i].GameID = game.ID
		}
		if len(rounds) > 0 {
			if err := tx.Create(&rounds).Error; err != nil {
				return fmt.Errorf("create rounds: %w", err)
			}
		}
		return nil
	})
}

// GetGameByID loads a game by id.
func (r *Repository) GetGameByID(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	var game Game
	if err := r.db.WithContext(ctx).First(&game, "id = ?", gameID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get game: %w", err)
	}
	return &game, nil
}

// GetSoloPlayer returns the active solo player for a game.
func (r *Repository) GetSoloPlayer(ctx context.Context, gameID uuid.UUID) (*GamePlayer, error) {
	var player GamePlayer
	if err := r.db.WithContext(ctx).
		Where("game_id = ? AND role = ? AND status = ?", gameID, PlayerRolePlayer, PlayerStatusActive).
		First(&player).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get solo player: %w", err)
	}
	return &player, nil
}

func (r *Repository) GetPlayerByOwner(ctx context.Context, gameID uuid.UUID, owner ownerIdentity) (*GamePlayer, error) {
	query := r.db.WithContext(ctx).Where("game_id = ? AND status = ?", gameID, PlayerStatusActive)
	if owner.userID != nil {
		query = query.Where("user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, nil
	}
	var player GamePlayer
	if err := query.First(&player).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get player by owner: %w", err)
	}
	return &player, nil
}

// StartGame activates a pending game and round 1.
func (r *Repository) StartGame(ctx context.Context, gameID uuid.UUID, now time.Time, timerSeconds *int) (*Game, error) {
	var game Game
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Status != GameStatusPending {
			return ErrInvalidTransition
		}
		var endsAt *time.Time
		if timerSeconds != nil {
			v := now.Add(time.Duration(*timerSeconds) * time.Second)
			endsAt = &v
		}
		if err := tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
			"status":     GameStatusActive,
			"started_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&Round{}).Where("game_id = ? AND round_number = ?", gameID, 1).Updates(map[string]any{
			"status":    RoundStatusActive,
			"starts_at": now,
			"ends_at":   endsAt,
		}).Error; err != nil {
			return err
		}
		return tx.First(&game, "id = ?", gameID).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("start game: %w", err)
	}
	current := 1
	game.CurrentRoundNumber = &current
	return &game, nil
}

func (r *Repository) StartPrivateRoomGame(ctx context.Context, gameID uuid.UUID, rounds []Round, now time.Time, timerSeconds *int, hooks MultiplayerTxHooks) (*MultiplayerStart, error) {
	out := &MultiplayerStart{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if !IsMultiplayerMode(game.Mode) || game.Status != GameStatusPending {
			return ErrInvalidTransition
		}
		var existingRounds int64
		if err := tx.Model(&Round{}).Where("game_id = ?", gameID).Count(&existingRounds).Error; err != nil {
			return err
		}
		if existingRounds == 0 && len(rounds) > 0 {
			for i := range rounds {
				rounds[i].GameID = gameID
			}
			if err := tx.Create(&rounds).Error; err != nil {
				return err
			}
		}
		var endsAt *time.Time
		if timerSeconds != nil {
			v := now.Add(time.Duration(*timerSeconds) * time.Second)
			endsAt = &v
		}
		if err := tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
			"status":     GameStatusActive,
			"started_at": now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&Round{}).Where("game_id = ? AND round_number = ?", gameID, 1).Updates(map[string]any{
			"status":    RoundStatusActive,
			"starts_at": now,
			"ends_at":   endsAt,
		}).Error; err != nil {
			return err
		}
		if hooks.OnMatchActive != nil {
			if err := hooks.OnMatchActive(ctx, tx, gameID, now); err != nil {
				return err
			}
		}
		if err := tx.First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		var round Round
		if err := tx.First(&round, "game_id = ? AND round_number = ?", gameID, 1).Error; err != nil {
			return err
		}
		out.Game = game
		out.CurrentRound = round
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("start private room game: %w", err)
	}
	return out, nil
}

// GetCurrentRound returns the active round or next pending round for a game.
func (r *Repository) GetCurrentRound(ctx context.Context, gameID uuid.UUID) (*currentRoundRow, error) {
	var row currentRoundRow
	query := `
		SELECT
			r.id AS round_id,
			r.round_number,
			r.status AS round_status,
			r.starts_at,
			r.ends_at,
			l.id AS location_id,
			l.provider,
			l.provider_ref,
			l.attribution
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		WHERE r.game_id = ?
		  AND r.status IN ('active', 'pending')
		ORDER BY CASE WHEN r.status = 'active' THEN 0 ELSE 1 END, r.round_number ASC
		LIMIT 1
	`
	if err := r.db.WithContext(ctx).Raw(query, gameID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("get current round: %w", err)
	}
	if row.RoundID == uuid.Nil {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) GetPracticeRoundByCreationKey(ctx context.Context, gameID uuid.UUID, key string) (*currentRoundRow, error) {
	var row currentRoundRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id AS round_id, r.round_number, r.status AS round_status,
		       r.starts_at, r.ends_at, l.id AS location_id, l.provider,
		       l.provider_ref, l.attribution
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		WHERE r.game_id = ? AND r.creation_idempotency_key = ?
		LIMIT 1
	`, gameID, key).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("get practice round replay: %w", err)
	}
	if row.RoundID == uuid.Nil {
		return nil, nil
	}
	return &row, nil
}

// AppendPracticeRound creates exactly one sequential active round after the
// prior round completes. The game row lock and per-game idempotency key make
// concurrent retries deterministic without Redis.
func (r *Repository) AppendPracticeRound(ctx context.Context, gameID, locationID uuid.UUID, idempotencyKey string, now time.Time) (*currentRoundRow, error) {
	if idempotencyKey == "" {
		return nil, ErrInvalidGameRequest
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Mode != GameModePractice {
			return ErrWrongGameMode
		}
		if game.Status != GameStatusActive {
			return ErrGameNotActive
		}
		var replay Round
		if err := tx.Where("game_id = ? AND creation_idempotency_key = ?", gameID, idempotencyKey).First(&replay).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var latest Round
		if err := tx.Clauses(lockingClause()).Where("game_id = ?", gameID).Order("round_number DESC").First(&latest).Error; err != nil {
			return err
		}
		if latest.Status != RoundStatusCompleted {
			return ErrCurrentRoundIncomplete
		}
		nextNumber := latest.RoundNumber + 1
		round := Round{
			GameID: gameID, LocationID: locationID, RoundNumber: nextNumber,
			Status: RoundStatusActive, StartsAt: &now, CreatedAt: now,
			CreationIdempotencyKey: &idempotencyKey,
		}
		if err := tx.Create(&round).Error; err != nil {
			return err
		}
		return tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
			"round_count": nextNumber,
			"updated_at":  now,
		}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("append practice round: %w", err)
	}
	return r.GetCurrentRound(ctx, gameID)
}

// EndPractice marks an open-ended session complete and preserves all history.
func (r *Repository) EndPractice(ctx context.Context, gameID uuid.UUID, now time.Time) (*Game, error) {
	var game Game
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Mode != GameModePractice {
			return ErrWrongGameMode
		}
		if game.Status == GameStatusCompleted {
			return nil
		}
		if game.Status != GameStatusActive && game.Status != GameStatusPending {
			return ErrGameNotActive
		}
		if err := tx.Model(&Round{}).Where("game_id = ? AND status IN ?", gameID, []string{RoundStatusActive, RoundStatusPending}).Updates(map[string]any{
			"status": RoundStatusCancelled,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
			"status": GameStatusCompleted, "completed_at": now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.First(&game, "id = ?", gameID).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("end practice: %w", err)
	}
	return &game, nil
}

type practiceHistoryRow struct {
	RoundID         uuid.UUID
	RoundNumber     int
	RoundStatus     string
	StartsAt        *time.Time
	EndsAt          *time.Time
	Provider        string
	ProviderRef     string
	Attribution     *string
	AnswerLatitude  float64
	AnswerLongitude float64
	CountryCode     string
	Region          *string
	Locality        *string
	GuessID         *uuid.UUID
	GuessLatitude   *float64
	GuessLongitude  *float64
	DistanceMeters  *int
	AccuracyScore   *int
	SpeedBonus      *int
	Score           *int
	SubmittedAt     *time.Time
	TimedOut        *bool
}

func (r *Repository) ListPracticeHistory(ctx context.Context, gameID, playerID uuid.UUID, after, limit int) ([]practiceHistoryRow, bool, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows []practiceHistoryRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id AS round_id, r.round_number, r.status AS round_status, r.starts_at, r.ends_at,
		       l.provider, l.provider_ref, l.attribution,
		       l.latitude AS answer_latitude, l.longitude AS answer_longitude,
		       l.country_code, l.region, l.locality,
		       g.id AS guess_id, g.latitude AS guess_latitude, g.longitude AS guess_longitude,
		       g.distance_meters, g.accuracy_score, g.speed_bonus, g.score, g.submitted_at, g.timed_out
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		LEFT JOIN guesses g ON g.round_id = r.id AND g.game_player_id = ?
		WHERE r.game_id = ? AND r.round_number > ?
		ORDER BY r.round_number ASC
		LIMIT ?
	`, playerID, gameID, after, limit+1).Scan(&rows).Error
	if err != nil {
		return nil, false, fmt.Errorf("list practice history: %w", err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

// SubmitGuessTx persists a guess and advances round/game state atomically.
func (r *Repository) SubmitGuessTx(ctx context.Context, gameID, roundID, playerID uuid.UUID, guess Guess, now time.Time) (*Guess, *answerLocation, bool, error) {
	var saved Guess
	var answer answerLocation
	completedGame := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		var round Round
		if err := tx.Clauses(lockingClause()).First(&round, "id = ? AND game_id = ?", roundID, gameID).Error; err != nil {
			return err
		}
		if round.Status != RoundStatusActive {
			return ErrRoundClosed
		}
		if round.EndsAt != nil && now.After(*round.EndsAt) {
			return ErrRoundClosed
		}
		if err := tx.Raw(`
			SELECT id, latitude, longitude, country_code, region, locality
			FROM locations
			WHERE id = ?
		`, round.LocationID).Scan(&answer).Error; err != nil {
			return err
		}
		guess.DistanceMeters = DistanceMeters(guess.Latitude, guess.Longitude, answer.Latitude, answer.Longitude)
		accuracy, bonus, total := ComposeGuessScores(GameModeSolo, ScoreV1(guess.DistanceMeters), 0, 0)
		guess.AccuracyScore = accuracy
		guess.SpeedBonus = bonus
		guess.Score = total
		guess.RoundID = roundID
		guess.GamePlayerID = playerID
		guess.SubmittedAt = now
		if err := tx.Create(&guess).Error; err != nil {
			return err
		}
		saved = guess
		if err := tx.Model(&Round{}).Where("id = ?", roundID).Updates(map[string]any{
			"status":      RoundStatusCompleted,
			"revealed_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&GamePlayer{}).Where("id = ?", playerID).UpdateColumn("total_score", gorm.Expr("total_score + ?", guess.Score)).Error; err != nil {
			return err
		}
		if err := tx.Model(&Game{}).Where("id = ?", gameID).UpdateColumn("total_score", gorm.Expr("total_score + ?", guess.Score)).Error; err != nil {
			return err
		}
		var next Round
		if err := tx.Where("game_id = ? AND status = ?", gameID, RoundStatusPending).Order("round_number ASC").First(&next).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if game.Mode == GameModePractice {
					return tx.Model(&Game{}).Where("id = ?", gameID).Update("updated_at", now).Error
				}
				completedGame = true
				if err := tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
					"status":       GameStatusCompleted,
					"completed_at": now,
				}).Error; err != nil {
					return err
				}
				return nil
			}
			return err
		}
		var endsAt *time.Time
		if game.TimerSeconds != nil {
			v := now.Add(time.Duration(*game.TimerSeconds) * time.Second)
			endsAt = &v
		}
		return tx.Model(&Round{}).Where("id = ?", next.ID).Updates(map[string]any{
			"status":    RoundStatusActive,
			"starts_at": now,
			"ends_at":   endsAt,
		}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("submit guess: %w", err)
	}
	return &saved, &answer, completedGame, nil
}

// ExpireSoloRoundTx records a zero-score timeout and advances the daily game.
func (r *Repository) ExpireSoloRoundTx(ctx context.Context, gameID, roundID, playerID uuid.UUID, now time.Time) (*Guess, *answerLocation, bool, error) {
	var saved Guess
	var answer answerLocation
	completedGame := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var round Round
		if err := tx.Clauses(lockingClause()).First(&round, "id = ? AND game_id = ?", roundID, gameID).Error; err != nil {
			return err
		}
		if round.Status != RoundStatusActive || round.EndsAt == nil || now.Before(*round.EndsAt) {
			return ErrRoundClosed
		}
		if err := tx.Raw(`SELECT id, latitude, longitude, country_code, region, locality FROM locations WHERE id = ?`, round.LocationID).Scan(&answer).Error; err != nil {
			return err
		}
		saved = Guess{
			RoundID: roundID, GamePlayerID: playerID,
			Latitude: 0, Longitude: 0, DistanceMeters: 0,
			AccuracyScore: 0, SpeedBonus: 0, Score: 0,
			SubmittedAt: now, TimedOut: true,
		}
		if err := tx.Create(&saved).Error; err != nil {
			return err
		}
		if err := tx.Model(&Round{}).Where("id = ?", roundID).Updates(map[string]any{"status": RoundStatusCompleted, "revealed_at": now}).Error; err != nil {
			return err
		}
		var next Round
		if err := tx.Where("game_id = ? AND status = ?", gameID, RoundStatusPending).Order("round_number ASC").First(&next).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			completedGame = true
			return tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{"status": GameStatusCompleted, "completed_at": now}).Error
		}
		var game Game
		if err := tx.First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		var endsAt *time.Time
		if game.TimerSeconds != nil {
			deadline := now.Add(time.Duration(*game.TimerSeconds) * time.Second)
			endsAt = &deadline
		}
		return tx.Model(&Round{}).Where("id = ?", next.ID).Updates(map[string]any{"status": RoundStatusActive, "starts_at": now, "ends_at": endsAt}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("expire solo round: %w", err)
	}
	return &saved, &answer, completedGame, nil
}

// MultiplayerTxHooks run inside the multiplayer game transaction so ranked/match
// lifecycle stays atomic with game/round state changes.
type MultiplayerTxHooks struct {
	OnMatchActive   func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error
	OnGameCompleted func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time) error
	OnGameCancelled func(ctx context.Context, tx *gorm.DB, gameID uuid.UUID, now time.Time, failureCode string) error
	// OnTerminalResult receives computed team totals/draw result when a matchmade game ends.
	OnTerminalResult func(ctx context.Context, tx *gorm.DB, result TerminalMatchResult) error
}

func (r *Repository) SubmitMultiplayerGuessTx(ctx context.Context, gameID, roundID, playerID uuid.UUID, guess Guess, now time.Time, hooks MultiplayerTxHooks) (*MultiplayerGuessOutcome, *answerLocation, error) {
	out := &MultiplayerGuessOutcome{}
	var answer answerLocation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if !IsMultiplayerMode(game.Mode) || game.Status != GameStatusActive {
			return ErrGameNotActive
		}
		var round Round
		if err := tx.Clauses(lockingClause()).First(&round, "id = ? AND game_id = ?", roundID, gameID).Error; err != nil {
			return err
		}
		if round.Status != RoundStatusActive {
			return ErrRoundClosed
		}
		if !CanGuessBeforeStart(game.Mode, round.StartsAt, now) {
			return ErrRoundClosed
		}
		if round.EndsAt != nil && now.After(*round.EndsAt) {
			return ErrRoundClosed
		}
		if err := tx.Raw(`
			SELECT id, latitude, longitude, country_code, region, locality
			FROM locations
			WHERE id = ?
		`, round.LocationID).Scan(&answer).Error; err != nil {
			return err
		}
		guess.DistanceMeters = DistanceMeters(guess.Latitude, guess.Longitude, answer.Latitude, answer.Longitude)
		// Atomic accuracy/speed/total: computed and written with the player total in one TX.
		accuracy, bonus, total := ScoreWithSpeedBonus(game.Mode, ScoreV1(guess.DistanceMeters), round.StartsAt, round.EndsAt, now)
		guess.AccuracyScore = accuracy
		guess.SpeedBonus = bonus
		guess.Score = total
		guess.RoundID = roundID
		guess.GamePlayerID = playerID
		guess.SubmittedAt = now
		if err := tx.Create(&guess).Error; err != nil {
			return err
		}
		// total_score accumulates accuracy + speed bonus (score column) for team standings.
		if err := tx.Model(&GamePlayer{}).Where("id = ?", playerID).UpdateColumn("total_score", gorm.Expr("total_score + ?", guess.Score)).Error; err != nil {
			return err
		}
		out.Guess = guess
		// First accepted multiplayer guess activates the ranked match record (legacy + canonical).
		if IsRankedMode(game.Mode) && hooks.OnMatchActive != nil {
			if err := hooks.OnMatchActive(ctx, tx, gameID, now); err != nil {
				return err
			}
		}
		submitted, eligible, err := multiplayerProgress(tx, gameID, roundID)
		if err != nil {
			return err
		}
		out.SubmittedCount = submitted
		out.EligibleCount = eligible
		if eligible > 0 && submitted >= eligible {
			return completeMultiplayerRound(ctx, tx, game, round.ID, now, out, hooks)
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("submit multiplayer guess: %w", err)
	}
	return out, &answer, nil
}

func (r *Repository) GetMultiplayerRoundState(ctx context.Context, gameID uuid.UUID) (*MultiplayerRoundState, error) {
	var row MultiplayerRoundState
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			r.id AS round_id,
			r.round_number,
			r.status,
			r.starts_at,
			r.ends_at,
			l.provider,
			l.provider_ref,
			l.attribution
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		WHERE r.game_id = ?
		  AND r.status IN ('active', 'completed')
		ORDER BY CASE WHEN r.status = 'active' THEN 0 ELSE 1 END, r.round_number DESC
		LIMIT 1
	`, gameID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("get multiplayer round state: %w", err)
	}
	if row.RoundID == uuid.Nil {
		return nil, nil
	}
	var submittedIDs []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&Guess{}).Where("round_id = ?", row.RoundID).Pluck("game_player_id", &submittedIDs).Error; err != nil {
		return nil, fmt.Errorf("get submitted player ids: %w", err)
	}
	var eligible int64
	if err := r.db.WithContext(ctx).Model(&GamePlayer{}).Where("game_id = ? AND status = ?", gameID, PlayerStatusActive).Count(&eligible).Error; err != nil {
		return nil, fmt.Errorf("count eligible players: %w", err)
	}
	row.SubmittedPlayerIDs = submittedIDs
	row.SubmittedCount = len(submittedIDs)
	row.EligibleCount = int(eligible)
	return &row, nil
}

// CancelMultiplayerGameTx marks a multiplayer game abandoned/cancelled and runs
// ranked lifecycle cancellation inside the same transaction.
// gameStatus must be GameStatusAbandoned or GameStatusCancelled.
func (r *Repository) CancelMultiplayerGameTx(ctx context.Context, gameID uuid.UUID, now time.Time, gameStatus, failureCode string, hooks MultiplayerTxHooks) error {
	if gameStatus != GameStatusAbandoned && gameStatus != GameStatusCancelled {
		return fmt.Errorf("invalid multiplayer terminal status %q", gameStatus)
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if !IsMultiplayerMode(game.Mode) {
			return ErrGameNotActive
		}
		if game.Status == GameStatusCompleted || game.Status == GameStatusAbandoned || game.Status == GameStatusCancelled {
			// Idempotent terminal replay.
			return nil
		}
		if game.Status != GameStatusActive && game.Status != GameStatusPending {
			return ErrGameNotActive
		}
		updates := map[string]any{
			"status":     gameStatus,
			"updated_at": now,
		}
		if err := tx.Model(&Game{}).Where("id = ? AND status = ?", gameID, game.Status).Updates(updates).Error; err != nil {
			return err
		}
		// Cancel non-terminal rounds so the game cannot advance after abandon/cancel.
		if err := tx.Model(&Round{}).
			Where("game_id = ? AND status IN ?", gameID, []string{RoundStatusPending, RoundStatusActive}).
			Updates(map[string]any{"status": RoundStatusCancelled}).Error; err != nil {
			return err
		}
		if hooks.OnGameCancelled != nil {
			return hooks.OnGameCancelled(ctx, tx, gameID, now, failureCode)
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrGameNotFound
	}
	if err != nil {
		return fmt.Errorf("cancel multiplayer game: %w", err)
	}
	return nil
}

func (r *Repository) CloseExpiredMultiplayerRound(ctx context.Context, gameID uuid.UUID, now time.Time, hooks MultiplayerTxHooks) (*MultiplayerGuessOutcome, error) {
	out := &MultiplayerGuessOutcome{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if !IsMultiplayerMode(game.Mode) || game.Status != GameStatusActive {
			return ErrGameNotActive
		}
		// Casual modes have no gameplay deadline; never auto-close on timer.
		if IsCasualMode(game.Mode) {
			return ErrRoundClosed
		}
		var round Round
		if err := tx.Clauses(lockingClause()).First(&round, "game_id = ? AND status = ?", gameID, RoundStatusActive).Error; err != nil {
			return err
		}
		if round.EndsAt == nil || now.Before(*round.EndsAt) {
			return ErrRoundClosed
		}
		// Insert zero scores for players who never submitted before closing.
		if err := insertMissingMultiplayerGuesses(tx, game.ID, round.ID, now); err != nil {
			return err
		}
		submitted, eligible, err := multiplayerProgress(tx, gameID, round.ID)
		if err != nil {
			return err
		}
		out.SubmittedCount = submitted
		out.EligibleCount = eligible
		return completeMultiplayerRound(ctx, tx, game, round.ID, now, out, hooks)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("close expired multiplayer round: %w", err)
	}
	return out, nil
}

// ListExpiredTimedMultiplayerGameIDs returns every timed multiplayer mode
// serviced by the shared deadline worker.
func (r *Repository) ListExpiredTimedMultiplayerGameIDs(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	if r == nil || r.db == nil {
		return nil, ErrGameNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT g.id
		FROM games g
		JOIN rounds r ON r.game_id = g.id
		WHERE g.status = 'active'
		  AND g.mode IN ('ranked_solo', 'ranked_duo', 'ranked_squad', 'party_lobby', 'private_room')
		  AND r.status = 'active'
		  AND r.ends_at IS NOT NULL
		  AND r.ends_at <= ?
		ORDER BY r.ends_at ASC, g.id ASC
		LIMIT ?
	`, now.UTC(), limit).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("list expired timed multiplayer games: %w", err)
	}
	return ids, nil
}

// CreateTeamGameBundle inserts a matchmade Casual/Ranked team game, slotted players,
// and all rounds in one transaction. Casual uses TimerSeconds=nil and null first-round ends_at.
// Exported for matchmaking formation (games ownership stays in games).
func (r *Repository) CreateTeamGameBundle(ctx context.Context, input TeamGameFormationInput) (*TeamGameFormationResult, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("create team game: repository unavailable")
	}
	if !IsCasualMode(input.Mode) && !IsRankedMode(input.Mode) {
		return nil, ErrInvalidGameRequest
	}
	if input.MapID == uuid.Nil || input.RoundCount < 1 || len(input.LocationIDs) < input.RoundCount {
		return nil, ErrInvalidGameRequest
	}
	if len(input.TeamOne) == 0 || len(input.TeamTwo) == 0 {
		return nil, ErrInvalidGameRequest
	}
	if len(input.TeamOne) != len(input.TeamTwo) {
		return nil, ErrInvalidGameRequest
	}
	scoringVersion := input.ScoringVersion
	if scoringVersion == 0 {
		scoringVersion = ScoringVersionV1
	}
	startedAt := input.StartedAt.UTC()
	roundStarts := input.RoundStartsAt.UTC()
	if roundStarts.IsZero() {
		roundStarts = startedAt
	}
	// Casual never persists a timer or first-round ends_at.
	// Ranked defaults to the authoritative 60-second timer when unset.
	var timerSeconds *int
	var firstEndsAt *time.Time
	if IsCasualMode(input.Mode) {
		timerSeconds = nil
		firstEndsAt = nil
	} else {
		timerSeconds = ApplyRankedTimerDefaults(input.Mode, input.TimerSeconds)
		// ends_at is relative to RoundStartsAt so every player shares the same deadline.
		if timerSeconds != nil {
			v := roundStarts.Add(time.Duration(*timerSeconds) * time.Second)
			firstEndsAt = &v
		}
	}

	out := &TeamGameFormationResult{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		game := Game{
			Mode:           input.Mode,
			Status:         GameStatusActive,
			MapID:          input.MapID,
			RoundCount:     input.RoundCount,
			TimerSeconds:   timerSeconds,
			ScoringVersion: scoringVersion,
			StartedAt:      &startedAt,
		}
		if err := tx.Create(&game).Error; err != nil {
			return fmt.Errorf("create team game: %w", err)
		}
		players := make([]GamePlayer, 0, len(input.TeamOne)+len(input.TeamTwo))
		appendTeam := func(members []TeamRosterMember, slot int) error {
			for _, m := range members {
				if m.UserID == uuid.Nil {
					return ErrInvalidGameRequest
				}
				name := m.DisplayName
				if name == "" {
					name = "Player"
				}
				uid := m.UserID
				slotCopy := slot
				p := GamePlayer{
					GameID:      game.ID,
					UserID:      &uid,
					DisplayName: name,
					Role:        PlayerRolePlayer,
					Status:      PlayerStatusActive,
					TeamSlot:    &slotCopy,
					JoinedAt:    startedAt,
				}
				if err := tx.Create(&p).Error; err != nil {
					return fmt.Errorf("create team player: %w", err)
				}
				players = append(players, p)
			}
			return nil
		}
		if err := appendTeam(input.TeamOne, TeamSlotOne); err != nil {
			return err
		}
		if err := appendTeam(input.TeamTwo, TeamSlotTwo); err != nil {
			return err
		}

		rounds := make([]Round, input.RoundCount)
		for i := 0; i < input.RoundCount; i++ {
			rounds[i] = Round{
				GameID:      game.ID,
				LocationID:  input.LocationIDs[i],
				RoundNumber: i + 1,
				Status:      RoundStatusPending,
				CreatedAt:   startedAt,
			}
			if i == 0 {
				rounds[i].Status = RoundStatusActive
				rounds[i].StartsAt = &roundStarts
				rounds[i].EndsAt = firstEndsAt
			}
		}
		if err := tx.Create(&rounds).Error; err != nil {
			return fmt.Errorf("create team rounds: %w", err)
		}
		out.Game = game
		out.Players = players
		out.Rounds = rounds
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListPlayers returns all game_players for a game ordered by join time.
func (r *Repository) ListPlayers(ctx context.Context, gameID uuid.UUID) ([]GamePlayer, error) {
	var players []GamePlayer
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Order("joined_at ASC, id ASC").Find(&players).Error; err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	return players, nil
}

// LoadSharedRoundResults returns answer + all guesses for a completed multiplayer round.
func (r *Repository) LoadSharedRoundResults(ctx context.Context, gameID, roundID uuid.UUID) (*SharedRoundResultsResponse, error) {
	var round Round
	if err := r.db.WithContext(ctx).First(&round, "id = ? AND game_id = ?", roundID, gameID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load round: %w", err)
	}
	if round.Status != RoundStatusCompleted {
		return nil, ErrResultsNotReady
	}
	answer, err := r.GetAnswerForRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if answer == nil {
		return nil, ErrRoundNotFound
	}
	players, err := r.ListPlayers(ctx, gameID)
	if err != nil {
		return nil, err
	}
	var guesses []Guess
	if err := r.db.WithContext(ctx).Where("round_id = ?", roundID).Find(&guesses).Error; err != nil {
		return nil, fmt.Errorf("load round guesses: %w", err)
	}
	guessByPlayer := make(map[uuid.UUID]Guess, len(guesses))
	for _, g := range guesses {
		guessByPlayer[g.GamePlayerID] = g
	}
	out := &SharedRoundResultsResponse{
		RoundID:        round.ID,
		RoundNumber:    round.RoundNumber,
		ActualLocation: toRevealedLocation(*answer),
		Guesses:        make([]PlayerGuessDTO, 0, len(players)),
		SubmittedCount: len(guesses),
		EligibleCount:  0,
	}
	for _, p := range players {
		if p.Status == PlayerStatusActive || p.Status == PlayerStatusDisconnected {
			out.EligibleCount++
		}
		g, ok := guessByPlayer[p.ID]
		if !ok {
			// Missing guess contributes zero (closure paths should already insert zeros).
			g = Guess{
				RoundID: roundID, GamePlayerID: p.ID,
				AccuracyScore: 0, SpeedBonus: 0, Score: 0, TimedOut: true,
			}
		}
		out.Guesses = append(out.Guesses, PlayerGuessDTO{
			GamePlayerID: p.ID,
			UserID:       p.UserID,
			DisplayName:  p.DisplayName,
			TeamSlot:     p.TeamSlot,
			Guess:        toGuessResult(g),
		})
	}
	totals := SumRoundTeamScores(players, guesses)
	out.TeamOneScore = totals.TeamOneScore
	out.TeamTwoScore = totals.TeamTwoScore
	return out, nil
}

// GetRoundByID returns a round by id.
func (r *Repository) GetRoundByID(ctx context.Context, roundID uuid.UUID) (*Round, error) {
	var round Round
	if err := r.db.WithContext(ctx).First(&round, "id = ?", roundID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get round: %w", err)
	}
	return &round, nil
}

// GetGuessByRoundPlayer returns the existing guess for one player in a round.
func (r *Repository) GetGuessByRoundPlayer(ctx context.Context, roundID, playerID uuid.UUID) (*Guess, error) {
	var guess Guess
	if err := r.db.WithContext(ctx).Where("round_id = ? AND game_player_id = ?", roundID, playerID).First(&guess).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get guess by round player: %w", err)
	}
	return &guess, nil
}

// GetGuessByIdempotencyKey returns the existing guess for a player's idempotency key.
func (r *Repository) GetGuessByIdempotencyKey(ctx context.Context, playerID uuid.UUID, key string) (*Guess, error) {
	var guess Guess
	if err := r.db.WithContext(ctx).Where("game_player_id = ? AND idempotency_key = ?", playerID, key).First(&guess).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get guess by idempotency key: %w", err)
	}
	return &guess, nil
}

// GetAnswerForRound returns revealed answer data for a round.
func (r *Repository) GetAnswerForRound(ctx context.Context, roundID uuid.UUID) (*answerLocation, error) {
	var answer answerLocation
	if err := r.db.WithContext(ctx).Raw(`
		SELECT l.id, l.latitude, l.longitude, l.country_code, l.region, l.locality
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		WHERE r.id = ?
	`, roundID).Scan(&answer).Error; err != nil {
		return nil, fmt.Errorf("get answer for round: %w", err)
	}
	if answer.ID == uuid.Nil {
		return nil, nil
	}
	return &answer, nil
}

// LoadResults loads final results in bounded batches, grouping multiplayer guesses by round.
func (r *Repository) LoadResults(ctx context.Context, gameID uuid.UUID) (*Game, []GamePlayer, []RoundResult, error) {
	game, err := r.GetGameByID(ctx, gameID)
	if err != nil || game == nil {
		return game, nil, nil, err
	}
	var players []GamePlayer
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Order("joined_at ASC").Find(&players).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("load players: %w", err)
	}
	var rows []struct {
		RoundID        uuid.UUID
		RoundNumber    int
		Latitude       float64
		Longitude      float64
		CountryCode    string
		Region         *string
		Locality       *string
		GuessID        *uuid.UUID
		GuessLatitude  *float64
		GuessLongitude *float64
		DistanceMeters *int
		AccuracyScore  *int
		SpeedBonus     *int
		Score          *int
		SubmittedAt    *time.Time
		TimedOut       *bool
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			r.id AS round_id,
			r.round_number,
			l.latitude,
			l.longitude,
			l.country_code,
			l.region,
			l.locality,
			g.id AS guess_id,
			g.latitude AS guess_latitude,
			g.longitude AS guess_longitude,
			g.distance_meters,
			g.accuracy_score,
			g.speed_bonus,
			g.score,
			g.submitted_at,
			g.timed_out
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		LEFT JOIN guesses g ON g.round_id = r.id
		WHERE r.game_id = ?
		ORDER BY r.round_number ASC, g.submitted_at ASC NULLS LAST
	`, gameID).Scan(&rows).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("load round results: %w", err)
	}
	results := make([]RoundResult, 0)
	indexByRound := make(map[uuid.UUID]int)
	for _, row := range rows {
		idx, ok := indexByRound[row.RoundID]
		if !ok {
			results = append(results, RoundResult{
				RoundID:     row.RoundID,
				RoundNumber: row.RoundNumber,
				ActualLocation: RevealedLocation{
					Latitude:    row.Latitude,
					Longitude:   row.Longitude,
					CountryCode: row.CountryCode,
					Region:      row.Region,
					Locality:    row.Locality,
				},
				Guesses: []GuessResult{},
			})
			idx = len(results) - 1
			indexByRound[row.RoundID] = idx
		}
		if row.GuessID != nil && row.GuessLatitude != nil && row.GuessLongitude != nil && row.DistanceMeters != nil && row.Score != nil && row.SubmittedAt != nil {
			acc := 0
			if row.AccuracyScore != nil {
				acc = *row.AccuracyScore
			} else {
				acc = *row.Score
			}
			bonus := 0
			if row.SpeedBonus != nil {
				bonus = *row.SpeedBonus
			}
			results[idx].Guesses = append(results[idx].Guesses, GuessResult{
				ID:             *row.GuessID,
				Latitude:       *row.GuessLatitude,
				Longitude:      *row.GuessLongitude,
				DistanceMeters: *row.DistanceMeters,
				AccuracyScore:  acc,
				SpeedBonus:     bonus,
				Score:          *row.Score,
				SubmittedAt:    *row.SubmittedAt,
				TimedOut:       row.TimedOut != nil && *row.TimedOut,
			})
		}
	}
	return game, players, results, nil
}

func multiplayerProgress(tx *gorm.DB, gameID, roundID uuid.UUID) (int, int, error) {
	var submitted int64
	if err := tx.Model(&Guess{}).Where("round_id = ?", roundID).Count(&submitted).Error; err != nil {
		return 0, 0, err
	}
	var eligible int64
	if err := tx.Model(&GamePlayer{}).Where("game_id = ? AND status = ?", gameID, PlayerStatusActive).Count(&eligible).Error; err != nil {
		return 0, 0, err
	}
	return int(submitted), int(eligible), nil
}

// insertMissingMultiplayerGuesses records zero-score timed-out guesses for active
// players who have not submitted when a timed round closes.
func insertMissingMultiplayerGuesses(tx *gorm.DB, gameID, roundID uuid.UUID, now time.Time) error {
	var players []GamePlayer
	if err := tx.Where("game_id = ? AND status = ?", gameID, PlayerStatusActive).Find(&players).Error; err != nil {
		return err
	}
	var submittedIDs []uuid.UUID
	if err := tx.Model(&Guess{}).Where("round_id = ?", roundID).Pluck("game_player_id", &submittedIDs).Error; err != nil {
		return err
	}
	submitted := make(map[uuid.UUID]struct{}, len(submittedIDs))
	for _, id := range submittedIDs {
		submitted[id] = struct{}{}
	}
	for _, p := range players {
		if _, ok := submitted[p.ID]; ok {
			continue
		}
		zero := Guess{
			RoundID:        roundID,
			GamePlayerID:   p.ID,
			Latitude:       0,
			Longitude:      0,
			DistanceMeters: 0,
			AccuracyScore:  0,
			SpeedBonus:     0,
			Score:          0,
			TimedOut:       true,
			SubmittedAt:    now,
		}
		if err := tx.Create(&zero).Error; err != nil {
			return err
		}
	}
	return nil
}

func completeMultiplayerRound(ctx context.Context, tx *gorm.DB, game Game, roundID uuid.UUID, now time.Time, out *MultiplayerGuessOutcome, hooks MultiplayerTxHooks) error {
	out.RoundID = roundID
	// When closing via all-submitted, missing guesses should not exist; when closing
	// via other paths that call insertMissing first, zeros are already present.
	// Defensive zero-fill keeps team totals correct for any residual missing rows.
	if err := insertMissingMultiplayerGuesses(tx, game.ID, roundID, now); err != nil {
		return err
	}
	if err := tx.Model(&Round{}).Where("id = ?", roundID).Updates(map[string]any{
		"status":      RoundStatusCompleted,
		"revealed_at": now,
	}).Error; err != nil {
		return err
	}
	out.RoundCompleted = true
	// Refresh counts after zero-fill.
	if submitted, eligible, err := multiplayerProgress(tx, game.ID, roundID); err == nil {
		out.SubmittedCount = submitted
		out.EligibleCount = eligible
	}
	var next Round
	if err := tx.Where("game_id = ? AND status = ?", game.ID, RoundStatusPending).Order("round_number ASC").First(&next).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			out.GameCompleted = true
			if err := tx.Model(&Game{}).Where("id = ?", game.ID).Updates(map[string]any{
				"status":       GameStatusCompleted,
				"completed_at": now,
				"updated_at":   now,
			}).Error; err != nil {
				return err
			}
			var players []GamePlayer
			if err := tx.Where("game_id = ?", game.ID).Find(&players).Error; err != nil {
				return err
			}
			terminal := BuildTerminalMatchResult(game.ID, game.Mode, players, now)
			out.Terminal = &terminal
			if hooks.OnTerminalResult != nil {
				if err := hooks.OnTerminalResult(ctx, tx, terminal); err != nil {
					return err
				}
			}
			if hooks.OnGameCompleted != nil {
				return hooks.OnGameCompleted(ctx, tx, game.ID, now)
			}
			return nil
		}
		return err
	}
	// Casual: null ends_at. Ranked/private_room: timer-based deadline when configured.
	endsAt := NextRoundEndsAt(game.Mode, game.TimerSeconds, now)
	if err := tx.Model(&Round{}).Where("id = ?", next.ID).Updates(map[string]any{
		"status":    RoundStatusActive,
		"starts_at": now,
		"ends_at":   endsAt,
	}).Error; err != nil {
		return err
	}
	out.NextRoundNumber = &next.RoundNumber
	out.NextRoundID = &next.ID
	return nil
}

func lockingClause() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

func toGuessResult(g Guess) GuessResult {
	return GuessResult{
		ID:             g.ID,
		Latitude:       g.Latitude,
		Longitude:      g.Longitude,
		DistanceMeters: g.DistanceMeters,
		AccuracyScore:  g.AccuracyScore,
		SpeedBonus:     g.SpeedBonus,
		Score:          g.Score,
		SubmittedAt:    g.SubmittedAt,
		TimedOut:       g.TimedOut,
	}
}
