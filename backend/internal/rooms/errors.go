package rooms

import "errors"

var (
	ErrRoomNotFound         = errors.New("room not found")
	ErrRoomFull             = errors.New("room full")
	ErrRoomNotJoinable      = errors.New("room not joinable")
	ErrRoomExpired          = errors.New("room expired")
	ErrRoomAlreadyStarted   = errors.New("room already started")
	ErrRoomSettingsLocked   = errors.New("room settings locked")
	ErrRoomHostRequired     = errors.New("room host required")
	ErrRoomPlayerNotFound   = errors.New("room player not found")
	ErrRoomPlayerRemoved    = errors.New("room player removed")
	ErrRoomReconnectExpired = errors.New("room reconnect expired")
	ErrRoomIdentityMismatch = errors.New("room identity mismatch")
	ErrRoomCodeRateLimited  = errors.New("room code rate limited")
	ErrInvalidRoomRequest   = errors.New("invalid room request")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
)

const (
	CodeRoomNotFound         = "room_not_found"
	CodeRoomFull             = "room_full"
	CodeRoomNotJoinable      = "room_not_joinable"
	CodeRoomExpired          = "room_expired"
	CodeRoomAlreadyStarted   = "room_already_started"
	CodeRoomSettingsLocked   = "room_settings_locked"
	CodeRoomHostRequired     = "room_host_required"
	CodeRoomPlayerNotFound   = "room_player_not_found"
	CodeRoomPlayerRemoved    = "room_player_removed"
	CodeRoomReconnectExpired = "room_reconnect_expired"
	CodeRoomIdentityMismatch = "room_identity_mismatch"
	CodeRoomCodeRateLimited  = "room_code_rate_limited"
	CodeIdempotencyConflict  = "idempotency_conflict"
)
