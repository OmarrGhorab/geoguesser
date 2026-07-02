package leaderboards

import (
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Response is the public leaderboard payload used by global, daily, and map reads.
type Response struct {
	Data []EntryDTO       `json:"data"`
	Page apphttp.PageInfo `json:"page"`
}

// EntryDTO is a public-safe registered-user leaderboard row.
type EntryDTO struct {
	Rank        int       `json:"rank"`
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Score       int       `json:"score"`
	GamesPlayed int       `json:"games_played"`
}

type cachedResponse struct {
	Data       []EntryDTO `json:"data"`
	Limit      int        `json:"limit"`
	NextCursor *string    `json:"next_cursor,omitempty"`
	CachedAt   time.Time  `json:"cached_at"`
}
