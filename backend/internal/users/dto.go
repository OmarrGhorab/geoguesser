package users

import (
	"time"

	"github.com/google/uuid"
)

// UserStatsResponse is the public stats response for a user.
type UserStatsResponse struct {
	Stats UserStats `json:"stats"`
}

// UserStats contains aggregate gameplay statistics.
type UserStats struct {
	GamesPlayed  int     `json:"games_played"`
	TotalScore   int     `json:"total_score"`
	AverageScore float64 `json:"average_score"`
	BestScore    int     `json:"best_score"`
}

// UserGameHistoryResponse is a cursor-paginated list of a user's games.
type UserGameHistoryResponse struct {
	Games []UserGameHistoryItem `json:"games"`
	Page  PageInfo              `json:"page"`
}

// UserGameHistoryItem is a lightweight summary of one game for history lists.
type UserGameHistoryItem struct {
	ID          uuid.UUID  `json:"id"`
	MapID       uuid.UUID  `json:"map_id"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	RoundCount  int        `json:"round_count"`
	TotalScore  int        `json:"total_score"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PageInfo describes pagination state for cursor-based lists.
type PageInfo struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
}
