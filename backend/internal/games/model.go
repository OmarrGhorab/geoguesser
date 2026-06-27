package games

import (
	"time"

	"github.com/google/uuid"
)

// Game is the durable solo game record.
type Game struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Mode               string     `gorm:"type:text;not null"`
	Status             string     `gorm:"type:text;not null;default:'pending'"`
	MapID              uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedByUserID    *uuid.UUID `gorm:"type:uuid"`
	RoundCount         int        `gorm:"type:int;not null;default:5"`
	TimerSeconds       *int       `gorm:"type:int"`
	ScoringVersion     int        `gorm:"type:int;not null;default:1"`
	TotalScore         int        `gorm:"type:int;not null;default:0"`
	CurrentRoundNumber *int       `gorm:"-"`
	StartedAt          *time.Time `gorm:"type:timestamptz"`
	CompletedAt        *time.Time `gorm:"type:timestamptz"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Game) TableName() string {
	return "games"
}

// Round is one playable challenge within a game.
type Round struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GameID      uuid.UUID  `gorm:"type:uuid;not null"`
	LocationID  uuid.UUID  `gorm:"type:uuid;not null"`
	RoundNumber int        `gorm:"type:int;not null"`
	Status      string     `gorm:"type:text;not null;default:'pending'"`
	StartsAt    *time.Time `gorm:"type:timestamptz"`
	EndsAt      *time.Time `gorm:"type:timestamptz"`
	RevealedAt  *time.Time `gorm:"type:timestamptz"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Round) TableName() string {
	return "rounds"
}

// GamePlayer is the solo participant snapshot.
type GamePlayer struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	GameID            uuid.UUID  `gorm:"type:uuid;not null"`
	UserID            *uuid.UUID `gorm:"type:uuid"`
	GuestIdentityHash *string    `gorm:"type:text"`
	DisplayName       string     `gorm:"type:text;not null"`
	Role              string     `gorm:"type:text;not null;default:'player'"`
	Status            string     `gorm:"type:text;not null;default:'active'"`
	TotalScore        int        `gorm:"type:int;not null;default:0"`
	JoinedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	LeftAt            *time.Time `gorm:"type:timestamptz"`
}

// TableName returns the database table name.
func (GamePlayer) TableName() string {
	return "game_players"
}

// Guess is a server-scored guess for one round.
type Guess struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoundID        uuid.UUID `gorm:"type:uuid;not null"`
	GamePlayerID   uuid.UUID `gorm:"type:uuid;not null"`
	Latitude       float64   `gorm:"type:numeric(9,6);not null"`
	Longitude      float64   `gorm:"type:numeric(9,6);not null"`
	DistanceMeters int       `gorm:"type:int;not null"`
	Score          int       `gorm:"type:int;not null"`
	IdempotencyKey *string   `gorm:"type:text"`
	SubmittedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
	CreatedAt      time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Guess) TableName() string {
	return "guesses"
}

// SelectedLocation is the answer data needed to create rounds and score guesses.
type SelectedLocation struct {
	ID          uuid.UUID
	Latitude    float64
	Longitude   float64
	CountryCode string
	Region      *string
	Locality    *string
	Provider    string
	Attribution *string
}

// currentRoundRow contains joined round/location data for current round reads.
type currentRoundRow struct {
	RoundID     uuid.UUID
	RoundNumber int
	RoundStatus string
	StartsAt    *time.Time
	EndsAt      *time.Time
	LocationID  uuid.UUID
	Provider    string
	ProviderRef string
	Attribution *string
}

// answerLocation is the hidden answer data used after reveal/scoring.
type answerLocation struct {
	ID          uuid.UUID
	Latitude    float64
	Longitude   float64
	CountryCode string
	Region      *string
	Locality    *string
}

// RevealedLocation is the answer location exposed after reveal.
type RevealedLocation struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryCode string  `json:"country_code"`
	Region      *string `json:"region,omitempty"`
	Locality    *string `json:"locality,omitempty"`
}

// RoundResult is a completed round result.
type RoundResult struct {
	RoundID        uuid.UUID        `json:"round_id"`
	RoundNumber    int              `json:"round_number"`
	ActualLocation RevealedLocation `json:"actual_location"`
	Guesses        []GuessResult    `json:"guesses"`
}
