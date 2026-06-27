package games

import (
	"time"

	"github.com/google/uuid"
)

// CreateGameRequest is the request body for creating a solo game.
type CreateGameRequest struct {
	Mode         string    `json:"mode"`
	MapID        uuid.UUID `json:"map_id"`
	RoundCount   int       `json:"round_count"`
	TimerSeconds *int      `json:"timer_seconds"`
}

// SubmitGuessRequest is the request body for submitting a round guess.
type SubmitGuessRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GameResponse wraps a game DTO.
type GameResponse struct {
	Game GameDTO `json:"game"`
}

// CurrentRoundResponse wraps a current round DTO.
type CurrentRoundResponse struct {
	Round RoundDTO `json:"round"`
}

// GuessResultResponse is returned after an accepted or replayed guess.
type GuessResultResponse struct {
	Guess          GuessResult      `json:"guess"`
	ActualLocation RevealedLocation `json:"actual_location"`
}

// GameResultsResponse returns final durable game results.
type GameResultsResponse struct {
	Game    GameDTO         `json:"game"`
	Players []GamePlayerDTO `json:"players"`
	Rounds  []RoundResult   `json:"rounds"`
}

// GameDTO is the public game shape.
type GameDTO struct {
	ID                 uuid.UUID  `json:"id"`
	Mode               string     `json:"mode"`
	Status             string     `json:"status"`
	MapID              uuid.UUID  `json:"map_id"`
	RoundCount         int        `json:"round_count"`
	TimerSeconds       *int       `json:"timer_seconds"`
	ScoringVersion     int        `json:"scoring_version"`
	CurrentRoundNumber *int       `json:"current_round_number"`
	TotalScore         int        `json:"total_score"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
}

// RoundDTO is safe for current-round responses before reveal.
type RoundDTO struct {
	ID          uuid.UUID   `json:"id"`
	RoundNumber int         `json:"round_number"`
	Status      string      `json:"status"`
	StartsAt    *time.Time  `json:"starts_at"`
	EndsAt      *time.Time  `json:"ends_at"`
	Media       *RoundMedia `json:"media"`
}

// RoundMedia is media metadata safe for current-round display.
type RoundMedia struct {
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	Attribution *string `json:"attribution"`
}

// LocationMediaProvider resolves stored location media references into public URLs.
type LocationMediaProvider interface {
	MediaURL(provider, providerRef string) (string, error)
}

// GuessResult is a public scored guess.
type GuessResult struct {
	ID             uuid.UUID `json:"id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	DistanceMeters int       `json:"distance_meters"`
	Score          int       `json:"score"`
	SubmittedAt    time.Time `json:"submitted_at"`
}

// GamePlayerDTO is a public participant snapshot.
type GamePlayerDTO struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	TotalScore  int        `json:"total_score"`
}
