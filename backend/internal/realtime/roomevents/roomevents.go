// Package roomevents defines the room-channel realtime event names shared by
// the rooms feature (publisher) and the realtime hub (validator, metrics,
// fanout). Both packages alias these constants, so a room event name cannot
// drift between the publisher and the hub allowlist: adding an event here is
// the single source of truth, and a name that only exists in one side is a
// compile error instead of a silent "other"-labelled metric or a rejected
// publish at runtime.
//
// This is a dependency-free leaf package because realtime imports rooms
// (handler wiring), so rooms cannot import realtime without a cycle.
package roomevents

const (
	EventRoomSnapshot           = "room.snapshot"
	EventRoomPlayerJoined       = "room.player_joined"
	EventRoomPlayerLeft         = "room.player_left"
	EventRoomPlayerDisconnected = "room.player_disconnected"
	EventRoomPlayerReconnected  = "room.player_reconnected"
	EventRoomPlayerRemoved      = "room.player_removed"
	EventRoomSettingsUpdated    = "room.settings_updated"
	EventRoomReadyUpdated       = "room.ready_updated"
	EventRoomReadyReset         = "room.ready_reset"
	EventRoomStarted            = "room.started"
	EventRoundStarted           = "round.started"
	EventRoundGuessCountChanged = "round.guess_count_changed"
	EventRoundEnded             = "round.ended"
	EventRoundResultsRevealed   = "round.results_revealed"
	EventGameCompleted          = "game.completed"
	EventRoomError              = "room.error"
)
