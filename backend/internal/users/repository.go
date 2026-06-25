package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository owns user read queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new users repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetUserByID returns a user with profile display name by id.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var result struct {
		ID          uuid.UUID `gorm:"column:id"`
		Email       string    `gorm:"column:email"`
		Role        string    `gorm:"column:role"`
		Status      string    `gorm:"column:status"`
		DisplayName string    `gorm:"column:display_name"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}

	if err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.email, u.role, u.status, p.display_name, u.created_at
		FROM users u
		JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = ?
	`, id).Scan(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &User{
		ID:          result.ID,
		Email:       result.Email,
		Role:        result.Role,
		Status:      result.Status,
		DisplayName: result.DisplayName,
		CreatedAt:   result.CreatedAt,
	}, nil
}

// GetUserStats returns aggregate stats for a user.
func (r *Repository) GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
	var result struct {
		GamesPlayed  int     `gorm:"column:games_played"`
		TotalScore   int     `gorm:"column:total_score"`
		AverageScore float64 `gorm:"column:average_score"`
		BestScore    int     `gorm:"column:best_score"`
	}

	if err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) AS games_played,
			COALESCE(SUM(total_score), 0) AS total_score,
			COALESCE(AVG(total_score), 0) AS average_score,
			COALESCE(MAX(total_score), 0) AS best_score
		FROM game_players
		WHERE user_id = ? AND status = 'active'
	`, userID).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &UserStats{
		GamesPlayed:  result.GamesPlayed,
		TotalScore:   result.TotalScore,
		AverageScore: result.AverageScore,
		BestScore:    result.BestScore,
	}, nil
}
