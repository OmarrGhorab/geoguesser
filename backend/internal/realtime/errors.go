package realtime

import "errors"

var (
	ErrOriginForbidden   = errors.New("realtime origin forbidden")
	ErrAuthRequired      = errors.New("realtime auth required")
	ErrNotParticipant    = errors.New("realtime participant required")
	ErrTicketInvalid     = errors.New("realtime ticket invalid")
	ErrTicketUsed        = errors.New("realtime ticket already used")
	ErrTicketExpired     = errors.New("realtime ticket expired")
	ErrVersionGap        = errors.New("realtime version gap")
	ErrCommandInvalid    = errors.New("realtime command invalid")
	ErrCommandThrottled  = errors.New("realtime command throttled")
	ErrPlayerLocked      = errors.New("realtime player locked")
	ErrChannelMismatch   = errors.New("realtime channel mismatch")
	ErrFeatureDisabled   = errors.New("realtime feature disabled")
	ErrSpectateForbidden = errors.New("spectate forbidden")
	ErrViewUnchanged     = errors.New("view unchanged")
)

const (
	CodeOriginForbidden  = "realtime_origin_forbidden"
	CodeAuthRequired     = "realtime_auth_required"
	CodeTicketInvalid    = "realtime_ticket_invalid"
	CodeTicketUsed       = "realtime_ticket_used"
	CodeTicketExpired    = "realtime_ticket_expired"
	CodeVersionGap       = "version_gap"
	CodeNotParticipant   = "not_found"
	CodeCommandInvalid   = "validation_failed"
	CodeCommandThrottled = "rate_limited"
	CodePlayerLocked     = "guess_locked"
	CodeFeatureDisabled  = "temporarily_unavailable"
	CodeSlowConsumer     = "slow_consumer"
	// CodeSpectateForbidden is privacy-safe: does not reveal why a target is disallowed.
	CodeSpectateForbidden = "spectate_forbidden"
)

// Application WebSocket close codes (contract).
const (
	CloseTicketUnauthorized = 4003
	CloseSlowConsumer       = 4008
)
