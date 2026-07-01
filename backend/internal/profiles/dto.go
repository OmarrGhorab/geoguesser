package profiles

import (
	"time"

	"github.com/google/uuid"
)

// ProfileResponse is the response for the current registered profile,
// including safe stats and saved-progress summaries.
type ProfileResponse struct {
	Profile  ProfileDTO  `json:"profile"`
	Stats    StatsDTO    `json:"stats"`
	Progress ProgressDTO `json:"progress"`
}

// ProfileDTO is the owner-facing profile shape. It never includes auth
// tokens or session identifiers; email is included because this response is
// only ever returned to the profile's owner.
type ProfileDTO struct {
	UserID      uuid.UUID      `json:"user_id"`
	Email       string         `json:"email"`
	DisplayName string         `json:"display_name"`
	AvatarURL   *string        `json:"avatar_url,omitempty"`
	CountryCode *string        `json:"country_code,omitempty"`
	Locale      string         `json:"locale"`
	Timezone    *string        `json:"timezone,omitempty"`
	Preferences map[string]any `json:"preferences,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// UpdateProfileRequest is the request body for PATCH /profile. Pointer
// fields are omitted-vs-present aware; fields that support explicit
// clearing use a double pointer so `null` differs from "not provided".
type UpdateProfileRequest struct {
	DisplayName *string        `json:"display_name,omitempty"`
	AvatarURL   OptionalString `json:"avatar_url,omitempty"`
	CountryCode OptionalString `json:"country_code,omitempty"`
	Locale      *string        `json:"locale,omitempty"`
	Timezone    OptionalString `json:"timezone,omitempty"`
	Preferences OptionalPrefs  `json:"preferences,omitempty"`
}

// StatsDTO is the privacy-safe aggregate stats shape shared by the current
// profile response and the public stats response.
type StatsDTO struct {
	GamesPlayed  int        `json:"games_played"`
	TotalScore   int        `json:"total_score"`
	AverageScore float64    `json:"average_score"`
	BestScore    int        `json:"best_score"`
	LastPlayedAt *time.Time `json:"last_played_at,omitempty"`
}

// ProgressDTO carries the current player's recent saved-progress summary.
type ProgressDTO struct {
	RecentGames []GameHistoryItemDTO `json:"recent_games"`
	Page        PageDTO              `json:"page"`
}

// PublicProfileResponse is the response for GET /users/{userId}/stats.
type PublicProfileResponse struct {
	Profile PublicProfileDTO `json:"profile"`
	Stats   StatsDTO         `json:"stats"`
}

// PublicProfileDTO is the privacy-safe profile shape shown to any viewer.
// It never includes email, auth/session data, or private preferences.
type PublicProfileDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CountryCode *string   `json:"country_code,omitempty"`
}

// GameHistoryResponse is the response for GET /users/{userId}/games.
type GameHistoryResponse struct {
	Games []GameHistoryItemDTO `json:"games"`
	Page  PageDTO              `json:"page"`
}

// GameHistoryItemDTO is a public-safe summary of one game. It excludes
// hidden round answers, location IDs, answer coordinates, raw guess
// coordinates, and provider metadata by construction: those fields simply
// have no place to live here.
type GameHistoryItemDTO struct {
	ID                 uuid.UUID  `json:"id"`
	MapID              uuid.UUID  `json:"map_id"`
	Mode               string     `json:"mode"`
	Status             string     `json:"status"`
	RoundCount         int        `json:"round_count"`
	CurrentRoundNumber *int       `json:"current_round_number,omitempty"`
	TotalScore         int        `json:"total_score"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// PageDTO describes cursor pagination state for list responses.
type PageDTO struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
}
