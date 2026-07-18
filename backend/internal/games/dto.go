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
// Solo/daily always populate ActualLocation. Multiplayer withholds it until the
// shared round closes (ActualLocation is then non-null and RoundCompleted true).
// Ranked guesses include accuracy_score + speed_bonus (0–250); Casual keeps
// speed_bonus=0. SubmittedCount/EligibleCount are multiplayer progress fields.
type GuessResultResponse struct {
	Guess              GuessResult       `json:"guess"`
	ActualLocation     *RevealedLocation `json:"actual_location"` // nil until delayed shared reveal
	MaxScore           int               `json:"max_score"`
	ScorePercent       int               `json:"score_percent"`
	MaxAccuracyScore   int               `json:"max_accuracy_score"`
	MaxSpeedBonus      int               `json:"max_speed_bonus"` // 250 Ranked; 0 Casual/solo
	Outcome            string            `json:"outcome"`
	RoundCompleted     bool              `json:"round_completed"`
	GameCompleted      bool              `json:"game_completed"`
	SubmittedCount     *int              `json:"submitted_count,omitempty"`
	EligibleCount      *int              `json:"eligible_count,omitempty"`
	NextRoundNumber    *int              `json:"next_round_number,omitempty"`
	NextRoundAvailable bool              `json:"next_round_available,omitempty"`
}

// GameResultsResponse returns final durable game results.
type GameResultsResponse struct {
	Game         GameDTO         `json:"game"`
	Players      []GamePlayerDTO `json:"players"`
	Rounds       []RoundResult   `json:"rounds"`
	TeamOneScore *int            `json:"team_one_score,omitempty"`
	TeamTwoScore *int            `json:"team_two_score,omitempty"`
	Result       *string         `json:"result,omitempty"`
	WinnerTeam   *int            `json:"winner_team_slot,omitempty"`
}

// SharedRoundResultsResponse is the multiplayer revealed round payload (post-close).
type SharedRoundResultsResponse struct {
	RoundID        uuid.UUID        `json:"round_id"`
	RoundNumber    int              `json:"round_number"`
	ActualLocation RevealedLocation `json:"actual_location"`
	Guesses        []PlayerGuessDTO `json:"guesses"`
	TeamOneScore   int              `json:"team_one_score"`
	TeamTwoScore   int              `json:"team_two_score"`
	SubmittedCount int              `json:"submitted_count"`
	EligibleCount  int              `json:"eligible_count"`
}

// PlayerGuessDTO is one participant's revealed guess for a closed round.
type PlayerGuessDTO struct {
	GamePlayerID uuid.UUID   `json:"game_player_id"`
	UserID       *uuid.UUID  `json:"user_id,omitempty"`
	DisplayName  string      `json:"display_name"`
	TeamSlot     *int        `json:"team_slot"`
	Guess        GuessResult `json:"guess"`
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
	OpenEnded          bool       `json:"open_ended"`
}

// PracticeRoundHistoryItem is one bounded owner-only Practice history row.
type PracticeRoundHistoryItem struct {
	Round          RoundDTO          `json:"round"`
	Guess          *GuessResult      `json:"guess,omitempty"`
	ActualLocation *RevealedLocation `json:"actual_location,omitempty"`
}

// PracticeHistoryResponse is a bounded cursor page for an open-ended session.
type PracticeHistoryResponse struct {
	Items      []PracticeRoundHistoryItem `json:"items"`
	NextCursor *string                    `json:"next_cursor"`
	HasMore    bool                       `json:"has_more"`
}

// RoundDTO is safe for current-round responses before reveal.
// Ranked sets StartsAt/EndsAt (shared 60s deadline for all players).
// Casual keeps both timer fields null (no scoring deadline).
type RoundDTO struct {
	ID             uuid.UUID   `json:"id"`
	RoundNumber    int         `json:"round_number"`
	Status         string      `json:"status"`
	StartsAt       *time.Time  `json:"starts_at"`
	EndsAt         *time.Time  `json:"ends_at"` // nil for Casual; Ranked shared deadline
	Media          *RoundMedia `json:"media"`
	SubmittedCount *int        `json:"submitted_count,omitempty"`
	EligibleCount  *int        `json:"eligible_count,omitempty"`
}

// RoundMedia is media metadata safe for current-round display.
type RoundMedia struct {
	Type        string  `json:"type"`
	URL         string  `json:"url,omitempty"`
	PanoramaID  string  `json:"panorama_id,omitempty"`
	Attribution *string `json:"attribution,omitempty"`
}

// LocationMediaProvider resolves stored location media references into public URLs.
type LocationMediaProvider interface {
	MediaURL(provider, providerRef string) (string, error)
}

// GuessResult is a public scored guess.
// Score MUST equal AccuracyScore + SpeedBonus (0–5,250 combined for Ranked).
type GuessResult struct {
	ID             uuid.UUID `json:"id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	DistanceMeters int       `json:"distance_meters"`
	AccuracyScore  int       `json:"accuracy_score"`
	SpeedBonus     int       `json:"speed_bonus"` // 0–250 Ranked; always 0 outside Ranked
	Score          int       `json:"score"`
	SubmittedAt    time.Time `json:"submitted_at"`
	TimedOut       bool      `json:"timed_out"`
}

// GamePlayerDTO is a public participant snapshot.
type GamePlayerDTO struct {
	ID          uuid.UUID  `json:"id"`
	UserID      *uuid.UUID `json:"user_id"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	TeamSlot    *int       `json:"team_slot"`
	TotalScore  int        `json:"total_score"`
}
