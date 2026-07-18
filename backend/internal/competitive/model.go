package competitive

import (
	"time"

	"github.com/google/uuid"
)

// Season and standing lifecycle constants.
const (
	SeasonStatusScheduled = "scheduled"
	SeasonStatusActive    = "active"
	SeasonStatusClosed    = "closed"

	OutcomeWin  = "win"
	OutcomeLoss = "loss"
	OutcomeDraw = "draw"

	// PlacementsRequired is the number of ranked matches before rank is visible.
	PlacementsRequired = 5

	// Default Elo / season constants (v1). Config may override service usage.
	DefaultInitialRating  = 800
	DefaultEloK           = 32
	DefaultAbandonPenalty = 15 // magnitude; stored as negative on rows
	DefaultResetFactorBPS = 5000
	DefaultTop500Min      = 25

	TeamSlotOne = 1
	TeamSlotTwo = 2
)

// Season is a competitive rating window (competitive_seasons).
type Season struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Sequence         int        `gorm:"type:int;not null"`
	Slug             string     `gorm:"type:text;not null"`
	Name             string     `gorm:"type:text;not null"`
	Status           string     `gorm:"type:text;not null"`
	StartsAt         time.Time  `gorm:"type:timestamptz;not null"`
	EndsAt           time.Time  `gorm:"type:timestamptz;not null"`
	ClosedAt         *time.Time `gorm:"type:timestamptz"`
	InitialRating    int        `gorm:"type:int;not null;default:800"`
	EloK             int        `gorm:"type:int;not null;default:32"`
	ResetFactorBPS   int        `gorm:"type:int;not null;default:5000"`
	Top500MinMatches int        `gorm:"type:int;not null;default:25"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Season) TableName() string { return "competitive_seasons" }

// Standing is a per-user seasonal rating row (competitive_standings).
// Primary key: (season_id, user_id).
type Standing struct {
	SeasonID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Rating              int        `gorm:"type:int;not null"`
	PlacementsCompleted int        `gorm:"type:smallint;not null;default:0"`
	MatchesPlayed       int        `gorm:"type:int;not null;default:0"`
	Wins                int        `gorm:"type:int;not null;default:0"`
	Losses              int        `gorm:"type:int;not null;default:0"`
	Draws               int        `gorm:"type:int;not null;default:0"`
	Abandons            int        `gorm:"type:int;not null;default:0"`
	PeakRating          int        `gorm:"type:int;not null"`
	RatingReachedAt     time.Time  `gorm:"type:timestamptz;not null"`
	LastMatchID         *uuid.UUID `gorm:"type:uuid"`
	FinalPosition       *int       `gorm:"type:int"`
	EndingRankCode      *string    `gorm:"type:text"`
	PeakRankCode        *string    `gorm:"type:text"`
	CreatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt           time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Standing) TableName() string { return "competitive_standings" }

// RankVisible reports whether placement matches are complete.
func (s Standing) RankVisible() bool {
	return s.PlacementsCompleted >= PlacementsRequired
}

// RatingChange is an exact-once per-match rating history row.
type RatingChange struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SeasonID            uuid.UUID `gorm:"type:uuid;not null"`
	MatchID             uuid.UUID `gorm:"type:uuid;not null"`
	UserID              uuid.UUID `gorm:"type:uuid;not null"`
	TeamSlot            int       `gorm:"type:smallint;not null"`
	Outcome             string    `gorm:"type:text;not null"`
	OwnTeamAverage      int       `gorm:"type:int;not null"`
	OpponentTeamAverage int       `gorm:"type:int;not null"`
	ExpectedBPS         int       `gorm:"type:int;not null"`
	BaseDelta           int       `gorm:"type:int;not null"`
	AbandonPenalty      int       `gorm:"type:int;not null;default:0"`
	OldRating           int       `gorm:"type:int;not null"`
	NewRating           int       `gorm:"type:int;not null"`
	TotalDelta          int       `gorm:"type:int;not null"`
	OldRankCode         *string   `gorm:"type:text"`
	NewRankCode         *string   `gorm:"type:text"`
	CreatedAt           time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (RatingChange) TableName() string { return "competitive_rating_changes" }

// MatchTerminal is the durable match facts needed for rating finalization.
type MatchTerminal struct {
	ID                     uuid.UUID
	GameID                 uuid.UUID
	Mode                   string
	Playlist               string
	Format                 string
	TeamSize               int
	SeasonID               *uuid.UUID
	Status                 string
	Result                 *string
	WinnerTeamSlot         *int
	ProgressionFinalizedAt *time.Time
	CompletedAt            *time.Time
}

// MatchParticipantTerminal is a roster row used during finalization.
type MatchParticipantTerminal struct {
	MatchID       uuid.UUID
	UserID        uuid.UUID
	TeamSlot      int
	AbandonedAt   *time.Time
	AbandonReason *string
	Status        string
}
