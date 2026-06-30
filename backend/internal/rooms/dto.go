package rooms

import (
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
)

type CreateRoomRequest struct {
	MapID        uuid.UUID `json:"map_id"`
	Visibility   string    `json:"visibility"`
	RoundCount   int       `json:"round_count"`
	TimerSeconds *int      `json:"timer_seconds"`
	MaxPlayers   int       `json:"max_players"`
	DisplayName  *string   `json:"display_name,omitempty"`
}

type JoinRoomRequest struct {
	Code        string  `json:"code"`
	DisplayName *string `json:"display_name,omitempty"`
}

type UpdateRoomSettingsRequest struct {
	MapID        *uuid.UUID `json:"map_id,omitempty"`
	RoundCount   *int       `json:"round_count,omitempty"`
	TimerSeconds *int       `json:"timer_seconds"`
	MaxPlayers   *int       `json:"max_players,omitempty"`
}

type ReadyRoomRequest struct {
	Ready bool `json:"ready"`
}

type CreateRoomResponse struct {
	Room RoomDTO `json:"room"`
}

type RoomResponse struct {
	Room RoomDTO `json:"room"`
}

type RoomDTO struct {
	ID              uuid.UUID            `json:"id"`
	Code            string               `json:"code"`
	Visibility      string               `json:"visibility"`
	Status          string               `json:"status"`
	GameID          *uuid.UUID           `json:"game_id"`
	HostPlayerID    *uuid.UUID           `json:"host_player_id"`
	CurrentPlayerID *uuid.UUID           `json:"current_player_id"`
	Version         int64                `json:"version"`
	MaxPlayers      int                  `json:"max_players"`
	RoundCount      int                  `json:"round_count"`
	TimerSeconds    *int                 `json:"timer_seconds"`
	ExpiresAt       time.Time            `json:"expires_at"`
	Players         []RoomPlayerDTO      `json:"players"`
	ReadyPlayerIDs  []uuid.UUID          `json:"ready_player_ids"`
	CurrentRound    *RoomCurrentRoundDTO `json:"current_round,omitempty"`
	GuessProgress   *RoomGuessProgress   `json:"guess_progress,omitempty"`
}

type RoomPlayerDTO struct {
	ID               uuid.UUID  `json:"id"`
	UserID           *uuid.UUID `json:"user_id"`
	DisplayName      string     `json:"display_name"`
	Role             string     `json:"role"`
	MembershipStatus string     `json:"membership_status"`
	PresenceStatus   string     `json:"presence_status"`
	IsReady          bool       `json:"is_ready"`
	TotalScore       int        `json:"total_score"`
	JoinedAt         time.Time  `json:"joined_at"`
	LeftAt           *time.Time `json:"left_at"`
}

type RoomCurrentRoundDTO struct {
	ID          uuid.UUID         `json:"id"`
	RoundNumber int               `json:"round_number"`
	Status      string            `json:"status"`
	StartsAt    *time.Time        `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at"`
	Media       *games.RoundMedia `json:"media"`
	Revealed    bool              `json:"revealed"`
}

type RoomGuessProgress struct {
	SubmittedCount     int         `json:"submitted_count"`
	EligibleCount      int         `json:"eligible_count"`
	SubmittedPlayerIDs []uuid.UUID `json:"submitted_player_ids"`
}

type SafeSettings struct {
	MapID        uuid.UUID `json:"map_id"`
	RoundCount   int       `json:"round_count"`
	TimerSeconds *int      `json:"timer_seconds"`
	MaxPlayers   int       `json:"max_players"`
}
