package matchplay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository is the PostgreSQL-backed matchplay store.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a matchplay repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// LoadSnapshotBundle implements Store.
func (r *Repository) LoadSnapshotBundle(ctx context.Context, matchID uuid.UUID) (*SnapshotBundle, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var match Match
	if err := r.db.WithContext(ctx).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match: %w", err)
	}

	var participants []MatchParticipant
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).
		Order("team_slot ASC, assigned_at ASC, user_id ASC").
		Find(&participants).Error; err != nil {
		return nil, fmt.Errorf("load participants: %w", err)
	}

	var players []GamePlayerRow
	if err := r.db.WithContext(ctx).Where("game_id = ?", match.GameID).Find(&players).Error; err != nil {
		return nil, fmt.Errorf("load players: %w", err)
	}

	bundle := &SnapshotBundle{
		Match:        match,
		Participants: participants,
		Players:      players,
		SubmittedIDs: map[uuid.UUID]bool{},
	}

	var current RoundRow
	err := r.db.WithContext(ctx).
		Where("game_id = ? AND status = ?", match.GameID, "active").
		Order("round_number ASC").
		Take(&current).Error
	if err == nil {
		bundle.CurrentRound = &current
		var submittedIDs []uuid.UUID
		if err := r.db.WithContext(ctx).Table("guesses").
			Where("round_id = ?", current.ID).
			Pluck("game_player_id", &submittedIDs).Error; err != nil {
			return nil, fmt.Errorf("load submitted: %w", err)
		}
		for _, id := range submittedIDs {
			bundle.SubmittedIDs[id] = true
		}
		var eligible int64
		if err := r.db.WithContext(ctx).Table("game_players").
			Where("game_id = ? AND status = ?", match.GameID, PlayerStatusActive).
			Count(&eligible).Error; err != nil {
			return nil, fmt.Errorf("count eligible: %w", err)
		}
		if eligible == 0 {
			// Fall back to non-abandoned participants.
			eligible = int64(countActiveParticipants(participants))
		}
		bundle.EligibleCount = int(eligible)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load current round: %w", err)
	}

	// Last completed round for optional last_round_result after reveal.
	var lastCompleted RoundRow
	err = r.db.WithContext(ctx).
		Where("game_id = ? AND status = ?", match.GameID, "completed").
		Order("round_number DESC").
		Take(&lastCompleted).Error
	if err == nil {
		bundle.LastCompletedRound = &lastCompleted
		var guesses []GuessRow
		if err := r.db.WithContext(ctx).Where("round_id = ?", lastCompleted.ID).Find(&guesses).Error; err != nil {
			return nil, fmt.Errorf("load last round guesses: %w", err)
		}
		bundle.LastRoundGuesses = guesses
		answer, err := r.loadAnswer(ctx, lastCompleted.LocationID)
		if err != nil {
			return nil, err
		}
		bundle.LastRoundAnswer = answer
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load last completed round: %w", err)
	}

	return bundle, nil
}

// LoadRoundResultBundle implements Store.
func (r *Repository) LoadRoundResultBundle(ctx context.Context, matchID, roundID uuid.UUID) (*RoundResultBundle, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var match Match
	if err := r.db.WithContext(ctx).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match: %w", err)
	}

	var round RoundRow
	if err := r.db.WithContext(ctx).Where("id = ? AND game_id = ?", roundID, match.GameID).Take(&round).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load round: %w", err)
	}
	if round.Status != "completed" {
		return nil, ErrRoundNotRevealed
	}

	var participants []MatchParticipant
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).Find(&participants).Error; err != nil {
		return nil, err
	}
	var players []GamePlayerRow
	if err := r.db.WithContext(ctx).Where("game_id = ?", match.GameID).Find(&players).Error; err != nil {
		return nil, err
	}
	var guesses []GuessRow
	if err := r.db.WithContext(ctx).Where("round_id = ?", round.ID).Find(&guesses).Error; err != nil {
		return nil, err
	}
	answer, err := r.loadAnswer(ctx, round.LocationID)
	if err != nil {
		return nil, err
	}
	if answer == nil {
		return nil, ErrUnavailable
	}

	return &RoundResultBundle{
		Match:        match,
		Participants: participants,
		Players:      players,
		Round:        round,
		Guesses:      guesses,
		Answer:       *answer,
		TeamOneScore: match.TeamOneScore,
		TeamTwoScore: match.TeamTwoScore,
	}, nil
}

// LoadTerminalResultBundle implements Store.
func (r *Repository) LoadTerminalResultBundle(ctx context.Context, matchID uuid.UUID) (*TerminalResultBundle, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var match Match
	if err := r.db.WithContext(ctx).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match: %w", err)
	}
	if !IsTerminalMatch(match.Status) {
		return nil, ErrMatchNotActive
	}

	var participants []MatchParticipant
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).Find(&participants).Error; err != nil {
		return nil, err
	}
	var players []GamePlayerRow
	if err := r.db.WithContext(ctx).Where("game_id = ?", match.GameID).Find(&players).Error; err != nil {
		return nil, err
	}

	var rounds []RoundRow
	if err := r.db.WithContext(ctx).Where("game_id = ? AND status = ?", match.GameID, "completed").
		Order("round_number ASC").Find(&rounds).Error; err != nil {
		return nil, err
	}

	out := &TerminalResultBundle{
		Match:        match,
		Participants: participants,
		Players:      players,
		Rounds:       make([]RoundResultBundle, 0, len(rounds)),
	}
	for _, round := range rounds {
		var guesses []GuessRow
		if err := r.db.WithContext(ctx).Where("round_id = ?", round.ID).Find(&guesses).Error; err != nil {
			return nil, err
		}
		answer, err := r.loadAnswer(ctx, round.LocationID)
		if err != nil {
			return nil, err
		}
		if answer == nil {
			continue
		}
		out.Rounds = append(out.Rounds, RoundResultBundle{
			Match:        match,
			Participants: participants,
			Players:      players,
			Round:        round,
			Guesses:      guesses,
			Answer:       *answer,
			TeamOneScore: match.TeamOneScore,
			TeamTwoScore: match.TeamTwoScore,
		})
	}
	return out, nil
}

// FindParticipant implements Store.
func (r *Repository) FindParticipant(ctx context.Context, matchID, userID uuid.UUID) (*MatchParticipant, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var p MatchParticipant
	if err := r.db.WithContext(ctx).Where("match_id = ? AND user_id = ?", matchID, userID).Take(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find participant: %w", err)
	}
	return &p, nil
}

// ExplicitLeaveTx implements Store.
func (r *Repository) ExplicitLeaveTx(ctx context.Context, matchID, userID uuid.UUID, now time.Time) (*LeaveOutcome, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var outcome LeaveOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applyForfeitTx(tx, matchID, userID, now, AbandonReasonExplicitLeave, &outcome)
	})
	if err != nil {
		return nil, err
	}
	return &outcome, nil
}

// ForfeitDisconnectTx implements Store.
func (r *Repository) ForfeitDisconnectTx(ctx context.Context, matchID, userID uuid.UUID, now time.Time) (*LeaveOutcome, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var outcome LeaveOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applyForfeitTx(tx, matchID, userID, now, AbandonReasonDisconnectTimeout, &outcome)
	})
	if err != nil {
		return nil, err
	}
	return &outcome, nil
}

// CloseInactiveCasualTx implements Store.
func (r *Repository) CloseInactiveCasualTx(ctx context.Context, matchID uuid.UUID, now time.Time) (*LeaveOutcome, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var outcome LeaveOutcome
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var match Match
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", matchID).Take(&match).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if IsTerminalMatch(match.Status) {
			outcome.Match = match
			outcome.AlreadyTerminal = true
			outcome.RestoredPartyIDs = partyIDsFromMatch(tx, match.ID)
			return nil
		}
		if !IsCasualMatch(match) {
			return ErrInvalidLeave
		}
		if match.Status != MatchStatusActive && match.Status != MatchStatusMatched {
			return ErrMatchNotActive
		}

		now = now.UTC()
		result := MatchResultAbandoned
		chatUntil := now.Add(DefaultChatAccessWindow)
		updates := map[string]any{
			"status":            MatchStatusCompleted,
			"result":            result,
			"completed_at":      now,
			"closed_at":         now,
			"chat_access_until": chatUntil,
			"last_activity_at":  now,
			"updated_at":        now,
			// No winner for mutual inactivity abandonment.
			"winner_team_slot": nil,
		}
		if err := tx.Model(&Match{}).Where("id = ?", match.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&MatchParticipant{}).
			Where("match_id = ? AND status IN ?", match.ID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
			Updates(map[string]any{
				"status":       ParticipantStatusCompleted,
				"completed_at": now,
				"closed_at":    now,
			}).Error; err != nil {
			return err
		}
		// Cancel open rounds and abandon the game.
		if err := tx.Exec(`
			UPDATE rounds SET status = 'cancelled'
			WHERE game_id = ? AND status IN ('pending', 'active')
		`, match.GameID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE games SET status = 'abandoned', updated_at = ?
			WHERE id = ? AND status IN ('pending', 'active')
		`, now, match.GameID).Error; err != nil {
			return err
		}

		match.Status = MatchStatusCompleted
		match.Result = &result
		match.CompletedAt = &now
		match.ClosedAt = &now
		match.ChatAccessUntil = &chatUntil
		match.LastActivityAt = now
		outcome.Match = match
		outcome.RestoredPartyIDs = collectPartyIDs(tx, match.ID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &outcome, nil
}

// ListDisconnectCandidates implements Store.
// Uses game_players.status = disconnected with left_at / updated heuristic via match last_activity.
// Durable disconnect markers: game_players.status disconnected and no abandoned_at on match_players.
func (r *Repository) ListDisconnectCandidates(ctx context.Context, now time.Time, grace time.Duration, limit int) ([]DisconnectCandidate, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 {
		limit = DefaultSweepBatchSize
	}
	cutoff := now.UTC().Add(-grace)

	type row struct {
		MatchID      uuid.UUID  `gorm:"column:match_id"`
		UserID       uuid.UUID  `gorm:"column:user_id"`
		GamePlayerID uuid.UUID  `gorm:"column:game_player_id"`
		TeamSlot     int        `gorm:"column:team_slot"`
		LeftAt       *time.Time `gorm:"column:left_at"`
		Playlist     string     `gorm:"column:playlist"`
		GameID       uuid.UUID  `gorm:"column:game_id"`
		PartyID      *uuid.UUID `gorm:"column:party_id"`
		UpdatedAt    time.Time  `gorm:"column:updated_at"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT mp.match_id, mp.user_id, mp.game_player_id, mp.team_slot,
		       gp.left_at, m.playlist, m.game_id, mp.party_id, m.updated_at
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		JOIN game_players gp ON gp.id = mp.game_player_id
		WHERE m.status IN ('matched', 'active')
		  AND mp.status IN ('assigned', 'active')
		  AND mp.abandoned_at IS NULL
		  AND gp.status = 'disconnected'
		  AND COALESCE(gp.left_at, m.updated_at) <= ?
		ORDER BY COALESCE(gp.left_at, m.updated_at) ASC
		LIMIT ?
	`, cutoff, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list disconnect candidates: %w", err)
	}

	out := make([]DisconnectCandidate, 0, len(rows))
	for _, row := range rows {
		at := row.UpdatedAt
		if row.LeftAt != nil {
			at = *row.LeftAt
		}
		out = append(out, DisconnectCandidate{
			MatchID:        row.MatchID,
			UserID:         row.UserID,
			GamePlayerID:   row.GamePlayerID,
			TeamSlot:       row.TeamSlot,
			DisconnectedAt: at,
			Playlist:       row.Playlist,
			GameID:         row.GameID,
			PartyID:        row.PartyID,
		})
	}
	return out, nil
}

// MarkConnectionState implements Store. It only updates live matches and uses
// the current state predicate to make repeated connect/disconnect callbacks idempotent.
func (r *Repository) MarkConnectionState(ctx context.Context, matchID, userID uuid.UUID, connected bool, at time.Time) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrUnavailable
	}
	status := "disconnected"
	leftAt := any(at.UTC())
	previous := []string{"active", "connected"}
	if connected {
		status = "active"
		leftAt = nil
		previous = []string{"disconnected"}
	}
	result := r.db.WithContext(ctx).Exec(`
		UPDATE game_players gp
		SET status = ?, left_at = ?
		FROM match_players mp
		JOIN matches m ON m.id = mp.match_id
		WHERE mp.game_player_id = gp.id
		  AND mp.match_id = ? AND mp.user_id = ?
		  AND m.status IN ('matched', 'active')
		  AND mp.status IN ('assigned', 'active')
		  AND mp.abandoned_at IS NULL
		  AND gp.status IN ?
	`, status, leftAt, matchID, userID, previous)
	if result.Error != nil {
		return false, fmt.Errorf("mark connection state: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// FindMatchIDByGameID resolves the durable realtime channel for game-owned
// post-commit transitions.
func (r *Repository) FindMatchIDByGameID(ctx context.Context, gameID uuid.UUID) (uuid.UUID, error) {
	if r == nil || r.db == nil {
		return uuid.Nil, ErrUnavailable
	}
	var matchID uuid.UUID
	err := r.db.WithContext(ctx).Model(&Match{}).Select("id").Where("game_id = ?", gameID).Scan(&matchID).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("find match by game: %w", err)
	}
	return matchID, nil
}

// ListInactiveCasualMatches implements Store.
func (r *Repository) ListInactiveCasualMatches(ctx context.Context, cutoff time.Time, limit int) ([]InactivityCandidate, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit <= 0 {
		limit = DefaultSweepBatchSize
	}
	type row struct {
		MatchID        uuid.UUID `gorm:"column:id"`
		GameID         uuid.UUID `gorm:"column:game_id"`
		LastActivityAt time.Time `gorm:"column:last_activity_at"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, game_id, last_activity_at
		FROM matches
		WHERE status IN ('matched', 'active')
		  AND playlist = 'casual'
		  AND last_activity_at <= ?
		ORDER BY last_activity_at ASC
		LIMIT ?
	`, cutoff.UTC(), limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list inactive casual: %w", err)
	}

	out := make([]InactivityCandidate, 0, len(rows))
	for _, row := range rows {
		parties := collectPartyIDs(r.db.WithContext(ctx), row.MatchID)
		out = append(out, InactivityCandidate{
			MatchID:        row.MatchID,
			GameID:         row.GameID,
			LastActivityAt: row.LastActivityAt,
			PartyIDs:       parties,
		})
	}
	return out, nil
}

// TouchActivity implements Store.
func (r *Repository) TouchActivity(ctx context.Context, matchID uuid.UUID, at time.Time) error {
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	return r.db.WithContext(ctx).Model(&Match{}).
		Where("id = ? AND status IN ?", matchID, []string{MatchStatusMatched, MatchStatusActive}).
		Updates(map[string]any{
			"last_activity_at": at.UTC(),
			"updated_at":       at.UTC(),
		}).Error
}

func (r *Repository) loadAnswer(ctx context.Context, locationID uuid.UUID) (*AnswerLocation, error) {
	var row struct {
		Latitude    float64 `gorm:"column:latitude"`
		Longitude   float64 `gorm:"column:longitude"`
		CountryCode string  `gorm:"column:country_code"`
		Region      *string `gorm:"column:region"`
		Locality    *string `gorm:"column:locality"`
		Provider    string  `gorm:"column:provider"`
		ProviderRef string  `gorm:"column:provider_ref"`
		Attribution *string `gorm:"column:attribution"`
	}
	err := r.db.WithContext(ctx).Table("locations").
		Select("latitude, longitude, country_code, region, locality, provider, provider_ref, attribution").
		Where("id = ?", locationID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load answer location: %w", err)
	}
	return &AnswerLocation{
		Latitude:    row.Latitude,
		Longitude:   row.Longitude,
		CountryCode: row.CountryCode,
		Region:      row.Region,
		Locality:    row.Locality,
		Provider:    row.Provider,
		ProviderRef: row.ProviderRef,
		Attribution: row.Attribution,
	}, nil
}

// applyForfeitTx marks user abandoned and ends the match as a forfeit for their team.
// Only the quitter receives abandon facts (teammates complete normally).
// Ranked leaves progression_finalized_at null so competitive finalization can apply
// a normal opponent win plus quitter-only -15 penalty. Never writes rating rows here.
func applyForfeitTx(tx *gorm.DB, matchID, userID uuid.UUID, now time.Time, reason string, outcome *LeaveOutcome) error {
	now = now.UTC()
	var match Match
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	var participant MatchParticipant
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("match_id = ? AND user_id = ?", matchID, userID).
		Take(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	// Idempotent terminal replay — no mutation of winner/abandon facts.
	if IsTerminalMatch(match.Status) {
		outcome.Match = match
		outcome.AlreadyTerminal = true
		outcome.RestoredPartyIDs = collectPartyIDs(tx, match.ID)
		return nil
	}
	if participant.AbandonedAt != nil {
		// Already marked abandoner on a still-active match (partial retry); do not double-apply.
		outcome.Match = match
		outcome.AlreadyTerminal = false
		return nil
	}
	if match.Status != MatchStatusActive && match.Status != MatchStatusMatched {
		return ErrInvalidLeave
	}

	// Mark abandoner only (quitter-only abandon facts).
	if err := tx.Model(&MatchParticipant{}).
		Where("match_id = ? AND user_id = ?", matchID, userID).
		Updates(map[string]any{
			"abandoned_at":   now,
			"abandon_reason": reason,
			"status":         ParticipantStatusCompleted,
			"completed_at":   now,
			"closed_at":      now,
		}).Error; err != nil {
		return err
	}
	if err := tx.Exec(`
		UPDATE game_players SET status = 'left', left_at = ?
		WHERE id = ?
	`, now, participant.GamePlayerID).Error; err != nil {
		return err
	}

	// Opposing team receives a normal win; result stays forfeit with winner_team_slot set
	// so progression applies base Elo as a normal win/loss plus quitter-only abandon penalty.
	winner := WinningTeamForForfeit(participant.TeamSlot)
	result := MatchResultForfeit
	chatUntil := now.Add(DefaultChatAccessWindow)
	// progression_finalized_at intentionally left null for Ranked (retryable pending).
	// Casual never sets it either (progression-neutral).
	if err := tx.Model(&Match{}).Where("id = ?", match.ID).Updates(map[string]any{
		"status":            MatchStatusCompleted,
		"result":            result,
		"winner_team_slot":  winner,
		"completed_at":      now,
		"closed_at":         now,
		"chat_access_until": chatUntil,
		"last_activity_at":  now,
		"updated_at":        now,
	}).Error; err != nil {
		return err
	}

	// Complete remaining non-abandoner participants without abandon facts.
	if err := tx.Model(&MatchParticipant{}).
		Where("match_id = ? AND status IN ? AND abandoned_at IS NULL", match.ID, []string{ParticipantStatusAssigned, ParticipantStatusActive}).
		Updates(map[string]any{
			"status":       ParticipantStatusCompleted,
			"completed_at": now,
			"closed_at":    now,
		}).Error; err != nil {
		return err
	}

	if err := tx.Exec(`
		UPDATE rounds SET status = 'cancelled'
		WHERE game_id = ? AND status IN ('pending', 'active')
	`, match.GameID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`
		UPDATE games SET status = 'abandoned', updated_at = ?
		WHERE id = ? AND status IN ('pending', 'active')
	`, now, match.GameID).Error; err != nil {
		return err
	}

	match.Status = MatchStatusCompleted
	match.Result = &result
	match.WinnerTeamSlot = &winner
	match.CompletedAt = &now
	match.ClosedAt = &now
	match.ChatAccessUntil = &chatUntil
	match.LastActivityAt = now
	// Explicitly ensure ranked abandon leaves progression pending.
	match.ProgressionFinalizedAt = nil
	outcome.Match = match
	outcome.AbandonedUserIDs = []uuid.UUID{userID}
	outcome.RestoredPartyIDs = collectPartyIDs(tx, match.ID)
	return nil
}

func collectPartyIDs(tx *gorm.DB, matchID uuid.UUID) []uuid.UUID {
	var ids []uuid.UUID
	_ = tx.Table("match_players").
		Where("match_id = ? AND party_id IS NOT NULL", matchID).
		Distinct("party_id").
		Pluck("party_id", &ids)
	return ids
}

func partyIDsFromMatch(tx *gorm.DB, matchID uuid.UUID) []uuid.UUID {
	return collectPartyIDs(tx, matchID)
}

func countActiveParticipants(participants []MatchParticipant) int {
	n := 0
	for _, p := range participants {
		if IsActiveParticipant(p.Status) && p.AbandonedAt == nil {
			n++
		}
	}
	return n
}
