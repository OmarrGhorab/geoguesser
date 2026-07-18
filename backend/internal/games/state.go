package games

import "time"

const (
	GameModeSolo        = "solo"
	GameModePrivateRoom = "private_room"
	GameModeDaily       = "daily"
	// GameModeRanked is the legacy multiplayer ranked mode stored on games.mode.
	GameModeRanked = "ranked"

	// Canonical matchmade game modes (aligned with matchmaking mode strings).
	GameModeCasualSolo  = "casual_solo"
	GameModeCasualDuo   = "casual_duo"
	GameModeCasualSquad = "casual_squad"
	GameModeRankedSolo  = "ranked_solo"
	GameModeRankedDuo   = "ranked_duo"
	GameModeRankedSquad = "ranked_squad"

	GameStatusPending   = "pending"
	GameStatusActive    = "active"
	GameStatusCompleted = "completed"
	GameStatusAbandoned = "abandoned"
	GameStatusCancelled = "cancelled"

	RoundStatusPending   = "pending"
	RoundStatusActive    = "active"
	RoundStatusCompleted = "completed"
	RoundStatusCancelled = "cancelled"

	PlayerRoleHost   = "host"
	PlayerRolePlayer = "player"

	PlayerStatusActive       = "active"
	PlayerStatusDisconnected = "disconnected"
	PlayerStatusLeft         = "left"
	PlayerStatusKicked       = "kicked"
)

// IsMultiplayerMode reports whether the game mode uses multiplayer round/guess semantics.
// Includes private rooms, legacy ranked, and all casual_*/ranked_* matchmade modes.
func IsMultiplayerMode(mode string) bool {
	switch mode {
	case GameModePrivateRoom, GameModeRanked,
		GameModeCasualSolo, GameModeCasualDuo, GameModeCasualSquad,
		GameModeRankedSolo, GameModeRankedDuo, GameModeRankedSquad:
		return true
	default:
		return false
	}
}

// IsRankedMode reports whether the game enforces ranked timer/scoring semantics.
// Includes legacy GameModeRanked and the three ranked_* canonical modes.
func IsRankedMode(mode string) bool {
	switch mode {
	case GameModeRanked, GameModeRankedSolo, GameModeRankedDuo, GameModeRankedSquad:
		return true
	default:
		return false
	}
}

// IsCasualMode reports whether the game is a matchmade casual mode (no timer/rating).
func IsCasualMode(mode string) bool {
	switch mode {
	case GameModeCasualSolo, GameModeCasualDuo, GameModeCasualSquad:
		return true
	default:
		return false
	}
}

// CanStart reports whether a game status can transition to active.
func CanStart(status string) bool {
	return status == GameStatusPending
}

// CanCompleteRound reports whether a round can transition to completed.
func CanCompleteRound(status string) bool {
	return status == RoundStatusActive
}

// CanGuessBeforeStart reports whether guesses are allowed before the round starts_at.
// Ranked modes enforce a scheduled countdown; casual and private rooms start immediately.
func CanGuessBeforeStart(mode string, startsAt *time.Time, now time.Time) bool {
	if startsAt == nil {
		return true
	}
	if IsRankedMode(mode) && now.Before(*startsAt) {
		return false
	}
	return true
}
