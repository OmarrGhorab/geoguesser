package games

import "time"

const (
	GameModeSolo        = "solo"
	GameModePrivateRoom = "private_room"
	GameModeRanked      = "ranked"

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
func IsMultiplayerMode(mode string) bool {
	return mode == GameModePrivateRoom || mode == GameModeRanked
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
// Ranked rounds enforce a scheduled countdown; private rooms start immediately.
func CanGuessBeforeStart(mode string, startsAt *time.Time, now time.Time) bool {
	if startsAt == nil {
		return true
	}
	if mode == GameModeRanked && now.Before(*startsAt) {
		return false
	}
	return true
}
