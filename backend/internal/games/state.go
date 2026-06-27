package games

const (
	GameModeSolo = "solo"

	GameStatusPending   = "pending"
	GameStatusActive    = "active"
	GameStatusCompleted = "completed"
	GameStatusAbandoned = "abandoned"
	GameStatusCancelled = "cancelled"

	RoundStatusPending   = "pending"
	RoundStatusActive    = "active"
	RoundStatusCompleted = "completed"
	RoundStatusCancelled = "cancelled"

	PlayerRolePlayer = "player"

	PlayerStatusActive = "active"
)

// CanStart reports whether a game status can transition to active.
func CanStart(status string) bool {
	return status == GameStatusPending
}

// CanCompleteRound reports whether a round can transition to completed.
func CanCompleteRound(status string) bool {
	return status == RoundStatusActive
}
