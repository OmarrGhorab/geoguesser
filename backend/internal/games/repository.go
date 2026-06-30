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

func (r *Repository) StartPrivateRoomGame(ctx context.Context, gameID uuid.UUID, rounds []Round, now time.Time, timerSeconds *int) (*MultiplayerStart, error) {
	out := &MultiplayerStart{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Mode != GameModePrivateRoom || game.Status != GameStatusPending {
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

// SubmitGuessTx persists a guess and advances round/game state atomically.
func (r *Repository) SubmitGuessTx(ctx context.Context, gameID, roundID, playerID uuid.UUID, guess Guess, now time.Time) (*Guess, *answerLocation, bool, error) {
	var saved Guess
	var answer answerLocation
	completedGame := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		guess.Score = ScoreV1(guess.DistanceMeters)
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
		var game Game
		if err := tx.First(&game, "id = ?", gameID).Error; err != nil {
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

func (r *Repository) SubmitMultiplayerGuessTx(ctx context.Context, gameID, roundID, playerID uuid.UUID, guess Guess, now time.Time) (*MultiplayerGuessOutcome, *answerLocation, error) {
	out := &MultiplayerGuessOutcome{}
	var answer answerLocation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Mode != GameModePrivateRoom || game.Status != GameStatusActive {
			return ErrGameNotActive
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
		guess.Score = ScoreV1(guess.DistanceMeters)
		guess.RoundID = roundID
		guess.GamePlayerID = playerID
		guess.SubmittedAt = now
		if err := tx.Create(&guess).Error; err != nil {
			return err
		}
		if err := tx.Model(&GamePlayer{}).Where("id = ?", playerID).UpdateColumn("total_score", gorm.Expr("total_score + ?", guess.Score)).Error; err != nil {
			return err
		}
		out.Guess = guess
		submitted, eligible, err := multiplayerProgress(tx, gameID, roundID)
		if err != nil {
			return err
		}
		out.SubmittedCount = submitted
		out.EligibleCount = eligible
		if eligible > 0 && submitted >= eligible {
			return completeMultiplayerRound(tx, gameID, round.ID, now, game.TimerSeconds, out)
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

func (r *Repository) CloseExpiredMultiplayerRound(ctx context.Context, gameID uuid.UUID, now time.Time) (*MultiplayerGuessOutcome, error) {
	out := &MultiplayerGuessOutcome{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var game Game
		if err := tx.Clauses(lockingClause()).First(&game, "id = ?", gameID).Error; err != nil {
			return err
		}
		if game.Mode != GameModePrivateRoom || game.Status != GameStatusActive {
			return ErrGameNotActive
		}
		var round Round
		if err := tx.Clauses(lockingClause()).First(&round, "game_id = ? AND status = ?", gameID, RoundStatusActive).Error; err != nil {
			return err
		}
		if round.EndsAt == nil || now.Before(*round.EndsAt) {
			return ErrRoundClosed
		}
		submitted, eligible, err := multiplayerProgress(tx, gameID, round.ID)
		if err != nil {
			return err
		}
		out.SubmittedCount = submitted
		out.EligibleCount = eligible
		return completeMultiplayerRound(tx, gameID, round.ID, now, game.TimerSeconds, out)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("close expired multiplayer round: %w", err)
	}
	return out, nil
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

// LoadResults loads final results in bounded batches.
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
		Score          *int
		SubmittedAt    *time.Time
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
			g.score,
			g.submitted_at
		FROM rounds r
		JOIN locations l ON l.id = r.location_id
		LEFT JOIN guesses g ON g.round_id = r.id
		WHERE r.game_id = ?
		ORDER BY r.round_number ASC
	`, gameID).Scan(&rows).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("load round results: %w", err)
	}
	results := make([]RoundResult, 0, len(rows))
	for _, row := range rows {
		result := RoundResult{
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
		}
		if row.GuessID != nil && row.GuessLatitude != nil && row.GuessLongitude != nil && row.DistanceMeters != nil && row.Score != nil && row.SubmittedAt != nil {
			result.Guesses = append(result.Guesses, GuessResult{
				ID:             *row.GuessID,
				Latitude:       *row.GuessLatitude,
				Longitude:      *row.GuessLongitude,
				DistanceMeters: *row.DistanceMeters,
				Score:          *row.Score,
				SubmittedAt:    *row.SubmittedAt,
			})
		}
		results = append(results, result)
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

func completeMultiplayerRound(tx *gorm.DB, gameID, roundID uuid.UUID, now time.Time, timerSeconds *int, out *MultiplayerGuessOutcome) error {
	if err := tx.Model(&Round{}).Where("id = ?", roundID).Updates(map[string]any{
		"status":      RoundStatusCompleted,
		"revealed_at": now,
	}).Error; err != nil {
		return err
	}
	out.RoundCompleted = true
	var next Round
	if err := tx.Where("game_id = ? AND status = ?", gameID, RoundStatusPending).Order("round_number ASC").First(&next).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			out.GameCompleted = true
			return tx.Model(&Game{}).Where("id = ?", gameID).Updates(map[string]any{
				"status":       GameStatusCompleted,
				"completed_at": now,
				"updated_at":   now,
			}).Error
		}
		return err
	}
	var endsAt *time.Time
	if timerSeconds != nil {
		v := now.Add(time.Duration(*timerSeconds) * time.Second)
		endsAt = &v
	}
	if err := tx.Model(&Round{}).Where("id = ?", next.ID).Updates(map[string]any{
		"status":    RoundStatusActive,
		"starts_at": now,
		"ends_at":   endsAt,
	}).Error; err != nil {
		return err
	}
	out.NextRoundNumber = &next.RoundNumber
	return nil
}

func lockingClause() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}
