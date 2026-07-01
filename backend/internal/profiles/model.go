package profiles

import (
	"time"

	"github.com/google/uuid"
)

// RegisteredProfile is the editable profile owned by a registered account.
// It is the private, owner-facing view and is never returned to other users.
type RegisteredProfile struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	AvatarURL   *string
	CountryCode *string
	Locale      string
	Timezone    *string
	Preferences map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProfileUpdate carries the requested changes to a registered profile. Nil
// pointer fields mean "leave unchanged"; a non-nil pointer to an empty/zero
// value means "clear this optional field" where the field supports clearing.
type ProfileUpdate struct {
	DisplayName    *string
	AvatarURL      **string
	CountryCode    **string
	Locale         *string
	Timezone       **string
	Preferences    **map[string]any
	HasDisplayName bool
	HasAvatarURL   bool
	HasCountryCode bool
	HasLocale      bool
	HasTimezone    bool
	HasPreferences bool
}

// PublicProfileSummary is the privacy-safe profile view shown to any viewer.
type PublicProfileSummary struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CountryCode *string
}

// StatsSummary is the privacy-safe aggregate gameplay stats for a user.
type StatsSummary struct {
	GamesPlayed  int
	TotalScore   int
	AverageScore float64
	BestScore    int
	LastPlayedAt *time.Time
}

// GameHistoryItem is a bounded, privacy-safe summary of one game a registered
// user participated in. It intentionally excludes hidden round answers,
// location IDs, answer coordinates, raw guess coordinates, and provider
// metadata.
type GameHistoryItem struct {
	GameID             uuid.UUID
	MapID              uuid.UUID
	Mode               string
	Status             string
	RoundCount         int
	CurrentRoundNumber *int
	TotalScore         int
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
}

// GameHistoryPage is a cursor-paginated page of game history items.
type GameHistoryPage struct {
	Items      []GameHistoryItem
	Limit      int
	NextCursor *string
}
