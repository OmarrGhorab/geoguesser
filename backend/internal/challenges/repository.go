package challenges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/maps"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDailyByDate(ctx context.Context, date time.Time) (*Challenge, error) {
	var challenge Challenge
	if err := r.db.WithContext(ctx).Where("type = ? AND challenge_date = ?", TypeDaily, date.Format("2006-01-02")).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get daily challenge: %w", err)
	}
	return &challenge, nil
}

func (r *Repository) GetChallengeByID(ctx context.Context, id uuid.UUID) (*Challenge, error) {
	var challenge Challenge
	if err := r.db.WithContext(ctx).First(&challenge, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get challenge: %w", err)
	}
	return &challenge, nil
}

func (r *Repository) GetSharedByCode(ctx context.Context, code string) (*Challenge, error) {
	var challenge Challenge
	if err := r.db.WithContext(ctx).Where("type = ? AND slug_or_code = ?", TypeShared, code).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get shared challenge: %w", err)
	}
	return &challenge, nil
}

func (r *Repository) GetDefaultActiveMapID(ctx context.Context) (uuid.UUID, error) {
	var row struct {
		ID uuid.UUID
	}
	if err := r.db.WithContext(ctx).
		Table("maps").
		Select("id").
		Where("status = ? AND visibility = ?", "active", "public").
		Order("created_at DESC, id DESC").
		Limit(1).
		Scan(&row).Error; err != nil {
		return uuid.Nil, fmt.Errorf("get default challenge map: %w", err)
	}
	if row.ID == uuid.Nil {
		return uuid.Nil, ErrChallengeUnavailable
	}
	return row.ID, nil
}

func (r *Repository) CreateChallengeWithLocations(ctx context.Context, challenge *Challenge, locations []ChallengeLocation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(challenge).Error; err != nil {
			return fmt.Errorf("create challenge: %w", err)
		}
		for i := range locations {
			locations[i].ChallengeID = challenge.ID
		}
		if len(locations) > 0 {
			if err := tx.Create(&locations).Error; err != nil {
				return fmt.Errorf("create challenge locations: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) ListChallengeLocations(ctx context.Context, challengeID uuid.UUID) ([]ChallengeLocation, error) {
	var locations []ChallengeLocation
	if err := r.db.WithContext(ctx).Where("challenge_id = ?", challengeID).Order("round_number ASC").Find(&locations).Error; err != nil {
		return nil, fmt.Errorf("list challenge locations: %w", err)
	}
	return locations, nil
}

func (r *Repository) GetAttemptForOwner(ctx context.Context, challengeID uuid.UUID, owner ownerIdentity) (*ChallengeAttempt, error) {
	query := r.db.WithContext(ctx).Where("challenge_id = ?", challengeID)
	if owner.userID != nil {
		query = query.Where("user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var attempt ChallengeAttempt
	if err := query.First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get challenge attempt: %w", err)
	}
	return &attempt, nil
}

func (r *Repository) GetAttemptByGameID(ctx context.Context, gameID uuid.UUID) (*ChallengeAttempt, error) {
	var attempt ChallengeAttempt
	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get attempt by game: %w", err)
	}
	return &attempt, nil
}

func (r *Repository) UpdateAttemptCompletion(ctx context.Context, attemptID uuid.UUID, totalScore int, totalDistanceMeters int, completionDurationMS *int64, completedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&ChallengeAttempt{}).Where("id = ?", attemptID).Updates(map[string]any{
		"status":                 AttemptStatusCompleted,
		"total_score":            totalScore,
		"total_distance_meters":  totalDistanceMeters,
		"completion_duration_ms": completionDurationMS,
		"completed_at":           completedAt,
		"updated_at":             completedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update attempt completion: %w", result.Error)
	}
	return nil
}

func (r *Repository) LoadGameRoundResults(ctx context.Context, gameID uuid.UUID) ([]gameRoundResult, error) {
	var rows []gameRoundResult
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			r.round_number,
			COALESCE(g.score, 0) AS score,
			COALESCE(g.distance_meters, 0) AS distance_meters
		FROM rounds r
		LEFT JOIN guesses g ON g.round_id = r.id
		WHERE r.game_id = ?
		ORDER BY r.round_number ASC
	`, gameID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load game round results: %w", err)
	}
	return rows, nil
}

func (r *Repository) GetGameCompletionData(ctx context.Context, gameID uuid.UUID) (*gameCompletionData, error) {
	var data gameCompletionData
	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			g.total_score,
			g.started_at,
			g.completed_at
		FROM games g
		WHERE g.id = ?
	`, gameID).Scan(&data).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get game completion data: %w", err)
	}
	if data.CompletedAt == nil {
		return nil, fmt.Errorf("game %s not completed", gameID)
	}
	return &data, nil
}

func (r *Repository) CreateAttemptWithGame(ctx context.Context, challenge Challenge, owner ownerIdentity, selected []maps.SelectedLocation, settings SettingsSnapshot, now time.Time) (*ChallengeAttempt, *games.Game, error) {
	var attempt ChallengeAttempt
	var game games.Game
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := getAttemptForOwnerTx(tx, challenge.ID, owner)
		if err != nil {
			return err
		}
		if existing != nil {
			attempt = *existing
			if existing.GameID != nil {
				if err := tx.First(&game, "id = ?", *existing.GameID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			return nil
		}
		game = games.Game{
			Mode:            "challenge",
			Status:          games.GameStatusPending,
			MapID:           challenge.MapID,
			CreatedByUserID: owner.userID,
			RoundCount:      settings.RoundCount,
			TimerSeconds:    settings.TimerSeconds,
			ScoringVersion:  settings.ScoringVersion,
		}
		if err := tx.Create(&game).Error; err != nil {
			return fmt.Errorf("create challenge game: %w", err)
		}
		player := games.GamePlayer{
			GameID:            game.ID,
			UserID:            owner.userID,
			GuestIdentityHash: owner.guestHash,
			DisplayName:       owner.displayName,
			Role:              games.PlayerRolePlayer,
			Status:            games.PlayerStatusActive,
		}
		if err := tx.Create(&player).Error; err != nil {
			return fmt.Errorf("create challenge player: %w", err)
		}
		rounds := make([]games.Round, len(selected))
		for i, location := range selected {
			rounds[i] = games.Round{GameID: game.ID, LocationID: location.ID, RoundNumber: i + 1, Status: games.RoundStatusPending}
		}
		if len(rounds) > 0 {
			if err := tx.Create(&rounds).Error; err != nil {
				return fmt.Errorf("create challenge rounds: %w", err)
			}
		}
		attempt = ChallengeAttempt{
			ChallengeID:         challenge.ID,
			GameID:              &game.ID,
			UserID:              owner.userID,
			GuestIdentityHash:   owner.guestHash,
			Status:              AttemptStatusPending,
			LeaderboardEligible: challenge.Type == TypeDaily && owner.userID != nil,
			StartedAt:           &now,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("create challenge attempt: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &attempt, &game, nil
}

func (r *Repository) StartAttemptGame(ctx context.Context, attemptID uuid.UUID, now time.Time) (*ChallengeAttempt, *games.Game, error) {
	var attempt ChallengeAttempt
	var game games.Game
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&attempt, "id = ?", attemptID).Error; err != nil {
			return err
		}
		if attempt.GameID == nil {
			return ErrChallengeUnavailable
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&game, "id = ?", *attempt.GameID).Error; err != nil {
			return err
		}
		if game.Status == games.GameStatusPending {
			var endsAt *time.Time
			if game.TimerSeconds != nil {
				v := now.Add(time.Duration(*game.TimerSeconds) * time.Second)
				endsAt = &v
			}
			if err := tx.Model(&games.Game{}).Where("id = ?", game.ID).Updates(map[string]any{"status": games.GameStatusActive, "started_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&games.Round{}).Where("game_id = ? AND round_number = ?", game.ID, 1).Updates(map[string]any{"status": games.RoundStatusActive, "starts_at": now, "ends_at": endsAt}).Error; err != nil {
				return err
			}
			if err := tx.Model(&ChallengeAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": AttemptStatusActive, "started_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.First(&game, "id = ?", game.ID).Error
		}
		if game.Status == games.GameStatusActive {
			return tx.Model(&ChallengeAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": AttemptStatusActive, "updated_at": now}).Error
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrChallengeNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("start challenge attempt: %w", err)
	}
	if game.Status == games.GameStatusActive {
		current := 1
		game.CurrentRoundNumber = &current
	}
	attempt.Status = AttemptStatusActive
	return &attempt, &game, nil
}

func (r *Repository) CountLeaderboardEntries(ctx context.Context, challengeID uuid.UUID) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&LeaderboardEntry{}).Where("challenge_id = ?", challengeID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count leaderboard entries: %w", err)
	}
	return int(count), nil
}

func (r *Repository) ListLeaderboardEntries(ctx context.Context, challengeID uuid.UUID, limit int, cursor string) ([]LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var entries []LeaderboardEntry
	query := r.db.WithContext(ctx).Where("challenge_id = ?", challengeID)
	if cursor != "" {
		var cursorAttemptID uuid.UUID
		if err := cursorAttemptID.UnmarshalText([]byte(cursor)); err == nil {
			query = query.Where("attempt_id > ?", cursorAttemptID)
		}
	}
	if err := query.Order("rank ASC, attempt_id ASC").Limit(limit).Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list leaderboard entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) ListLeaderboardEntriesForUser(ctx context.Context, challengeID uuid.UUID, userID uuid.UUID) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry
	if err := r.db.WithContext(ctx).Where("challenge_id = ? AND user_id = ?", challengeID, userID).Order("rank ASC").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list leaderboard entries for user: %w", err)
	}
	return entries, nil
}

func (r *Repository) CreateResultAndLeaderboardEntry(ctx context.Context, attempt ChallengeAttempt, result ChallengeResult, displayName string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&result).Error; err != nil {
			return fmt.Errorf("create challenge result: %w", err)
		}
		if attempt.UserID == nil || !attempt.LeaderboardEligible {
			return nil
		}
		entry := LeaderboardEntry{
			ChallengeID:          attempt.ChallengeID,
			AttemptID:            attempt.ID,
			UserID:               *attempt.UserID,
			DisplayNameSnapshot:  displayName,
			Score:                result.TotalScore,
			CompletionDurationMS: attempt.CompletionDurationMS,
			CompletedAt:          result.CompletedAt,
			Rank:                 1,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&entry).Error; err != nil {
			return fmt.Errorf("create leaderboard entry: %w", err)
		}
		return r.recomputeRanksTx(tx, attempt.ChallengeID)
	})
}

func (r *Repository) GetResultByAttempt(ctx context.Context, attemptID uuid.UUID) (*ChallengeResult, error) {
	var result ChallengeResult
	if err := r.db.WithContext(ctx).First(&result, "attempt_id = ?", attemptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get challenge result: %w", err)
	}
	return &result, nil
}

func (r *Repository) ListAttemptsForOwner(ctx context.Context, owner ownerIdentity, limit int) ([]ChallengeAttempt, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := r.db.WithContext(ctx)
	if owner.userID != nil {
		query = query.Where("user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var attempts []ChallengeAttempt
	if err := query.Order("updated_at DESC, id DESC").Limit(limit).Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("list owner challenge attempts: %w", err)
	}
	return attempts, nil
}

func (r *Repository) UpsertStreak(ctx context.Context, owner ownerIdentity, streak Streak) error {
	if owner.userID != nil {
		streak.OwnerUserID = owner.userID
	} else if owner.guestHash != nil {
		streak.GuestIdentityHash = owner.guestHash
	} else {
		return ErrForbidden
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: ownerColumn(owner)}},
		DoUpdates: clause.AssignmentColumns([]string{
			"current_count",
			"best_count",
			"last_completed_challenge_date",
			"status",
			"protection_state",
			"updated_at",
		}),
	}).Create(&streak).Error
}

func (r *Repository) CreateStreakEvent(ctx context.Context, event StreakEvent) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: ownerColumn(ownerIdentity{userID: event.OwnerUserID, guestHash: event.GuestIdentityHash})}, {Name: "challenge_date"}, {Name: "event_type"}},
		DoNothing: true,
	}).Create(&event).Error
}

func (r *Repository) ApplyMissionProgressEvent(ctx context.Context, mission Mission, owner ownerIdentity, event MissionProgressEvent, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&mission).Error; err != nil {
			return fmt.Errorf("ensure mission: %w", err)
		}
		event.MissionID = mission.ID
		event.OwnerUserID = owner.userID
		event.GuestIdentityHash = owner.guestHash
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error; err != nil {
			return fmt.Errorf("create mission progress event: %w", err)
		}
		progress := MissionProgress{
			MissionID:         mission.ID,
			OwnerUserID:       owner.userID,
			GuestIdentityHash: owner.guestHash,
			CurrentValue:      minInt(event.Delta, mission.TargetValue),
			TargetValue:       mission.TargetValue,
			Status:            "in_progress",
			UpdatedAt:         now,
		}
		if progress.CurrentValue >= progress.TargetValue {
			progress.Status = "completed"
			progress.CompletedAt = &now
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "mission_id"}, {Name: ownerColumn(owner)}},
			DoUpdates: clause.Assignments(map[string]any{
				"current_value": gorm.Expr("LEAST(mission_progress.current_value + EXCLUDED.current_value, mission_progress.target_value)"),
				"status":        gorm.Expr("CASE WHEN LEAST(mission_progress.current_value + EXCLUDED.current_value, mission_progress.target_value) >= mission_progress.target_value THEN 'completed' ELSE 'in_progress' END"),
				"completed_at":  gorm.Expr("CASE WHEN LEAST(mission_progress.current_value + EXCLUDED.current_value, mission_progress.target_value) >= mission_progress.target_value THEN ? ELSE mission_progress.completed_at END", now),
				"updated_at":    now,
			}),
		}).Create(&progress).Error
	})
}

func (r *Repository) ListMissionProgressForOwner(ctx context.Context, owner ownerIdentity) ([]MissionProgress, error) {
	query := r.db.WithContext(ctx)
	if owner.userID != nil {
		query = query.Where("owner_user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var progress []MissionProgress
	if err := query.Order("mission_id ASC").Find(&progress).Error; err != nil {
		return nil, fmt.Errorf("list mission progress: %w", err)
	}
	return progress, nil
}

func (r *Repository) GetMissionProgress(ctx context.Context, missionID uuid.UUID, owner ownerIdentity) (*MissionProgress, error) {
	query := r.db.WithContext(ctx).Where("mission_id = ?", missionID)
	if owner.userID != nil {
		query = query.Where("owner_user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var progress MissionProgress
	if err := query.First(&progress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get mission progress: %w", err)
	}
	return &progress, nil
}

func (r *Repository) ClaimMissionProgress(ctx context.Context, missionID uuid.UUID, owner ownerIdentity, now time.Time) error {
	query := r.db.WithContext(ctx).Model(&MissionProgress{}).Where("mission_id = ? AND status = ?", missionID, "completed")
	if owner.userID != nil {
		query = query.Where("owner_user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return ErrForbidden
	}
	result := query.Where("claimed_at IS NULL").Update("claimed_at", now)
	if result.Error != nil {
		return fmt.Errorf("claim mission progress: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChallengeNotFound
	}
	return nil
}

func (r *Repository) ListActiveMissions(ctx context.Context, now time.Time) ([]Mission, error) {
	var missions []Mission
	if err := r.db.WithContext(ctx).Where("status = ? AND active_starts_at <= ? AND (active_ends_at IS NULL OR active_ends_at > ?)", "active", now, now).Order("active_starts_at ASC").Find(&missions).Error; err != nil {
		return nil, fmt.Errorf("list active missions: %w", err)
	}
	return missions, nil
}

func (r *Repository) GetMissionByCode(ctx context.Context, code string) (*Mission, error) {
	var mission Mission
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&mission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get mission by code: %w", err)
	}
	return &mission, nil
}

func (r *Repository) UpsertMission(ctx context.Context, mission *Mission) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(mission).Error
}

func (r *Repository) recomputeRanksTx(tx *gorm.DB, challengeID uuid.UUID) error {
	var entries []LeaderboardEntry
	if err := tx.Where("challenge_id = ?", challengeID).
		Order("score DESC, completion_duration_ms ASC NULLS LAST, completed_at ASC, attempt_id ASC").
		Find(&entries).Error; err != nil {
		return err
	}
	for i := range entries {
		rank := i + 1
		if err := tx.Model(&LeaderboardEntry{}).
			Where("challenge_id = ? AND attempt_id = ?", challengeID, entries[i].AttemptID).
			Update("rank", rank).Error; err != nil {
			return err
		}
	}
	return nil
}

func leaderboardLess(a, b LeaderboardEntry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	if a.CompletionDurationMS != nil && b.CompletionDurationMS != nil && *a.CompletionDurationMS != *b.CompletionDurationMS {
		return *a.CompletionDurationMS < *b.CompletionDurationMS
	}
	if a.CompletionDurationMS != nil && b.CompletionDurationMS == nil {
		return true
	}
	if a.CompletionDurationMS == nil && b.CompletionDurationMS != nil {
		return false
	}
	if !a.CompletedAt.Equal(b.CompletedAt) {
		return a.CompletedAt.Before(b.CompletedAt)
	}
	return a.AttemptID.String() < b.AttemptID.String()
}

func (r *Repository) GetStreakForOwner(ctx context.Context, owner ownerIdentity) (*Streak, error) {
	query := r.db.WithContext(ctx)
	if owner.userID != nil {
		query = query.Where("owner_user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var streak Streak
	if err := query.First(&streak).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get streak: %w", err)
	}
	return &streak, nil
}

func getAttemptForOwnerTx(tx *gorm.DB, challengeID uuid.UUID, owner ownerIdentity) (*ChallengeAttempt, error) {
	query := tx.Where("challenge_id = ?", challengeID)
	if owner.userID != nil {
		query = query.Where("user_id = ?", *owner.userID)
	} else if owner.guestHash != nil {
		query = query.Where("guest_identity_hash = ?", *owner.guestHash)
	} else {
		return nil, ErrForbidden
	}
	var attempt ChallengeAttempt
	if err := query.First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

func encodeSettings(settings SettingsSnapshot) (json.RawMessage, error) {
	b, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("encode settings snapshot: %w", err)
	}
	return b, nil
}

func decodeSettings(raw json.RawMessage) (SettingsSnapshot, error) {
	var settings SettingsSnapshot
	if len(raw) == 0 {
		return settings, ErrInvalidChallengeInput
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings, fmt.Errorf("decode settings snapshot: %w", err)
	}
	if settings.RoundCount == 0 {
		settings.RoundCount = DefaultRoundCount
	}
	if settings.ScoringVersion == 0 {
		settings.ScoringVersion = DefaultScoringVersion
	}
	return settings, nil
}

func ownerColumn(owner ownerIdentity) string {
	if owner.userID != nil {
		return "owner_user_id"
	}
	return "guest_identity_hash"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type gameRoundResult struct {
	RoundNumber    int `gorm:"column:round_number"`
	Score          int `gorm:"column:score"`
	DistanceMeters int `gorm:"column:distance_meters"`
}

type gameCompletionData struct {
	TotalScore int        `gorm:"column:total_score"`
	StartedAt  *time.Time `gorm:"column:started_at"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}
