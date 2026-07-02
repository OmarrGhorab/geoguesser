package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository owns profile, public stats, and game history queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new profiles repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type profileRow struct {
	UserID      uuid.UUID `gorm:"column:user_id"`
	Email       string    `gorm:"column:email"`
	DisplayName string    `gorm:"column:display_name"`
	AvatarURL   *string   `gorm:"column:avatar_url"`
	CountryCode *string   `gorm:"column:country_code"`
	Locale      string    `gorm:"column:locale"`
	Timezone    *string   `gorm:"column:timezone"`
	Preferences []byte    `gorm:"column:preferences"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (row profileRow) toDomain() (*RegisteredProfile, error) {
	prefs, err := decodePreferences(row.Preferences)
	if err != nil {
		return nil, err
	}
	return &RegisteredProfile{
		UserID:      row.UserID,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		CountryCode: row.CountryCode,
		Locale:      row.Locale,
		Timezone:    row.Timezone,
		Preferences: prefs,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func decodePreferences(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var prefs map[string]any
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return nil, fmt.Errorf("failed to decode preferences: %w", err)
	}
	if prefs == nil {
		prefs = map[string]any{}
	}
	return prefs, nil
}

// GetCurrentProfile returns the full owner-facing profile for a registered
// user, or nil if the account or its profile row does not exist.
func (r *Repository) GetCurrentProfile(ctx context.Context, userID uuid.UUID) (*RegisteredProfile, error) {
	var row profileRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.email, p.display_name, p.avatar_url, p.country_code,
		       p.locale, p.timezone, p.preferences, p.created_at, p.updated_at
		FROM users u
		JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = ?
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}
	if row.UserID == uuid.Nil {
		return nil, nil
	}
	return row.toDomain()
}

// UpdateProfile applies a partial update to a registered user's profile
// within a row-locking transaction so concurrent updates from different
// sessions cannot silently clobber each other's untouched fields.
func (r *Repository) UpdateProfile(ctx context.Context, userID uuid.UUID, update ProfileUpdate) (*RegisteredProfile, error) {
	var result *RegisteredProfile

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row profileRow
		err := tx.Raw(`
			SELECT u.id AS user_id, u.email, p.display_name, p.avatar_url, p.country_code,
			       p.locale, p.timezone, p.preferences, p.created_at, p.updated_at
			FROM users u
			JOIN user_profiles p ON p.user_id = u.id
			WHERE u.id = ?
			FOR UPDATE OF p
		`, userID).Scan(&row).Error
		if err != nil {
			return fmt.Errorf("failed to lock profile: %w", err)
		}
		if row.UserID == uuid.Nil {
			return ErrProfileNotFound
		}

		if update.HasDisplayName && update.DisplayName != nil {
			row.DisplayName = *update.DisplayName
		}
		if update.HasAvatarURL {
			row.AvatarURL = *update.AvatarURL
		}
		if update.HasCountryCode {
			row.CountryCode = *update.CountryCode
		}
		if update.HasLocale && update.Locale != nil {
			row.Locale = *update.Locale
		}
		if update.HasTimezone {
			row.Timezone = *update.Timezone
		}

		prefsRaw := row.Preferences
		if update.HasPreferences {
			prefs := *update.Preferences
			if prefs == nil {
				prefsRaw = []byte("{}")
			} else {
				encoded, encErr := json.Marshal(*prefs)
				if encErr != nil {
					return fmt.Errorf("failed to encode preferences: %w", encErr)
				}
				prefsRaw = encoded
			}
		}

		if err := tx.Exec(`
			UPDATE user_profiles
			SET display_name = ?, avatar_url = ?, country_code = ?, locale = ?, timezone = ?, preferences = ?
			WHERE user_id = ?
		`, row.DisplayName, row.AvatarURL, row.CountryCode, row.Locale, row.Timezone, prefsRaw, userID).Error; err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}

		var updated profileRow
		if err := tx.Raw(`
			SELECT u.id AS user_id, u.email, p.display_name, p.avatar_url, p.country_code,
			       p.locale, p.timezone, p.preferences, p.created_at, p.updated_at
			FROM users u
			JOIN user_profiles p ON p.user_id = u.id
			WHERE u.id = ?
		`, userID).Scan(&updated).Error; err != nil {
			return fmt.Errorf("failed to reload profile: %w", err)
		}

		result, err = updated.toDomain()
		return err
	})
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return nil, err
		}
		return nil, err
	}

	return result, nil
}

type publicUserRow struct {
	UserID      uuid.UUID `gorm:"column:user_id"`
	Status      string    `gorm:"column:status"`
	DisplayName string    `gorm:"column:display_name"`
	AvatarURL   *string   `gorm:"column:avatar_url"`
	CountryCode *string   `gorm:"column:country_code"`
}

// GetPublicProfile returns the public-safe profile summary for a user. It
// returns nil for missing users and for users whose account status is not
// publicly visible (disabled, deleted, pending verification), so callers do
// not need to branch on account state themselves.
func (r *Repository) GetPublicProfile(ctx context.Context, userID uuid.UUID) (*PublicProfileSummary, error) {
	var row publicUserRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.status, p.display_name, p.avatar_url, p.country_code
		FROM users u
		JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = ?
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get public profile: %w", err)
	}
	if row.UserID == uuid.Nil || row.Status != "active" {
		return nil, nil
	}
	return &PublicProfileSummary{
		UserID:      row.UserID,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		CountryCode: row.CountryCode,
	}, nil
}

// GetStats returns aggregate public-safe stats for a user, based only on
// eligible completed games. Missing games return a zero-value summary, not
// an error.
func (r *Repository) GetStats(ctx context.Context, userID uuid.UUID) (*StatsSummary, error) {
	var result struct {
		GamesPlayed  int        `gorm:"column:games_played"`
		TotalScore   int        `gorm:"column:total_score"`
		AverageScore float64    `gorm:"column:average_score"`
		BestScore    int        `gorm:"column:best_score"`
		LastPlayedAt *time.Time `gorm:"column:last_played_at"`
	}

	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS games_played,
			COALESCE(SUM(gp.total_score), 0) AS total_score,
			COALESCE(AVG(gp.total_score), 0) AS average_score,
			COALESCE(MAX(gp.total_score), 0) AS best_score,
			MAX(g.completed_at) AS last_played_at
		FROM game_players gp
		JOIN games g ON g.id = gp.game_id
		WHERE gp.user_id = ? AND gp.status = 'active' AND g.status = 'completed'
	`, userID).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &StatsSummary{
		GamesPlayed:  result.GamesPlayed,
		TotalScore:   result.TotalScore,
		AverageScore: result.AverageScore,
		BestScore:    result.BestScore,
		LastPlayedAt: result.LastPlayedAt,
	}, nil
}

// ListGameHistory returns a cursor-paginated, privacy-safe list of games a
// registered user participated in, ordered by most recent first.
func (r *Repository) ListGameHistory(ctx context.Context, userID uuid.UUID, limit int, cursor string) (*GameHistoryPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	createdAtCursor, idCursor, cursorErr := parseHistoryCursor(cursor)
	if cursor != "" && cursorErr != nil {
		return nil, ErrInvalidCursor
	}

	query := `
		SELECT
			g.id AS game_id, g.map_id, g.mode, g.status, g.round_count, g.total_score,
			g.started_at, g.completed_at, g.created_at,
			(SELECT r.round_number FROM rounds r WHERE r.game_id = g.id AND r.status = 'active' LIMIT 1) AS current_round_number
		FROM games g
		JOIN game_players gp ON gp.game_id = g.id
		WHERE gp.user_id = ?
		  AND gp.status = 'active'
		  AND g.status IN ('completed', 'active', 'abandoned')
	`
	args := []any{userID}
	if createdAtCursor != nil && idCursor != nil {
		query += ` AND (g.created_at, g.id) < (?, ?)`
		args = append(args, *createdAtCursor, *idCursor)
	}
	query += ` ORDER BY g.created_at DESC, g.id DESC LIMIT ?`
	args = append(args, limit)

	var rows []GameHistoryItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list game history: %w", err)
	}

	page := &GameHistoryPage{Items: rows, Limit: limit}
	if len(rows) == limit {
		last := rows[len(rows)-1]
		c := last.CreatedAt.Format(time.RFC3339Nano) + "|" + last.GameID.String()
		page.NextCursor = &c
	}
	return page, nil
}

func parseHistoryCursor(cursor string) (*time.Time, *uuid.UUID, error) {
	if cursor == "" {
		return nil, nil, nil
	}
	parts := strings.SplitN(cursor, "|", 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("malformed cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, nil, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, nil, err
	}
	return &createdAt, &id, nil
}
