package parties

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors for the parties package.
var (
	ErrUnauthorized        = errors.New("parties unauthorized")
	ErrNotFound            = errors.New("party not found")
	ErrInviteNotFound      = errors.New("party invite not found")
	ErrActivePartyConflict = errors.New("active party conflict")
	ErrActiveQueueOrMatch  = errors.New("active queue or match")
	ErrUnsupportedFormat   = errors.New("unsupported party format")
	ErrLeaderRequired      = errors.New("party leader required")
	ErrFriendUnavailable   = errors.New("friend unavailable")
	ErrPartyFull           = errors.New("party full")
	ErrAlreadyInvited      = errors.New("already invited")
	ErrTargetBusy          = errors.New("target busy")
	ErrPartyLocked         = errors.New("party locked")
	ErrPartyIncomplete     = errors.New("party incomplete")
	ErrPartyNotReady       = errors.New("party not ready")
	ErrInvalidRequest      = errors.New("invalid party request")
	ErrInvalidJSON         = errors.New("invalid party json")
	ErrInvalidPartyID      = errors.New("invalid party id")
	ErrInvalidInviteID     = errors.New("invalid invite id")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrSelfInvite          = errors.New("cannot invite self")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrIdempotencyRequired = errors.New("idempotency key required")
	ErrDependencyFailure   = errors.New("parties dependency failure")
	ErrRateLimited         = errors.New("parties rate limited")
	ErrVersionMismatch     = errors.New("party version mismatch")
	ErrFeatureDisabled     = errors.New("casual matchmaking disabled")
)

// Stable domain-specific API codes.
const (
	CodeActivePartyConflict = "active_party_conflict"
	CodeActiveQueueOrMatch  = "active_queue_or_match"
	CodeUnsupportedFormat   = "unsupported_format"
	CodeLeaderRequired      = "leader_required"
	CodeFriendUnavailable   = "friend_unavailable"
	CodePartyFull           = "party_full"
	CodeAlreadyInvited      = "already_invited"
	CodeTargetBusy          = "target_busy"
	CodePartyLocked         = "party_locked"
	CodePartyIncomplete     = "party_incomplete"
	CodePartyNotReady       = "party_not_ready"
	CodeIdempotencyConflict = "idempotency_conflict"
	CodeUnavailable         = "parties_unavailable"
)

// Safe user-facing messages (frontend localizes).
const (
	MsgActivePartyConflict = "You already have an active party."
	MsgActiveQueueOrMatch  = "You cannot change parties while queued or in a match."
	MsgUnsupportedFormat   = "That party format is not supported."
	MsgLeaderRequired      = "Only the party leader can perform this action."
	MsgFriendUnavailable   = "That player is not available for party invites."
	MsgPartyFull           = "The party is full."
	MsgAlreadyInvited      = "That player already has a pending invite to this party."
	MsgTargetBusy          = "That player is already in a party or match."
	MsgPartyLocked         = "Leave the queue before changing the party."
	MsgPartyIncomplete     = "The party is not complete."
	MsgPartyNotReady       = "Every party member must be ready."
	MsgIdempotencyConflict = "The idempotency key was reused with a different request."
	MsgIdempotencyRequired = "An Idempotency-Key header is required."
	MsgUnavailable         = "Party features are temporarily unavailable."
	MsgNotFound            = "The requested party was not found."
	MsgInviteNotFound      = "The party invitation was not found."
	MsgSelfInvite          = "You cannot invite yourself."
	MsgInvalidRequest      = "The party request is invalid."
)

// ToAPIError maps domain errors to stable HTTP API errors.
func ToAPIError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized
	case errors.Is(err, ErrInvalidJSON):
		return apphttp.ErrInvalidJSON
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidPartyID),
		errors.Is(err, ErrInvalidInviteID), errors.Is(err, ErrInvalidUserID),
		errors.Is(err, ErrSelfInvite), errors.Is(err, ErrIdempotencyRequired):
		if errors.Is(err, ErrSelfInvite) {
			return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgSelfInvite).WithCause(err)
		}
		if errors.Is(err, ErrIdempotencyRequired) {
			return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgIdempotencyRequired).WithCause(err)
		}
		return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgInvalidRequest).WithCause(err)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrInviteNotFound):
		return apphttp.ErrNotFound
	case errors.Is(err, ErrFriendUnavailable):
		return apphttp.NewAPIError(http.StatusNotFound, CodeFriendUnavailable, MsgFriendUnavailable).WithCause(err)
	case errors.Is(err, ErrLeaderRequired):
		return apphttp.NewAPIError(http.StatusForbidden, CodeLeaderRequired, MsgLeaderRequired).WithCause(err)
	case errors.Is(err, ErrActivePartyConflict):
		return apphttp.NewAPIError(http.StatusConflict, CodeActivePartyConflict, MsgActivePartyConflict).WithCause(err)
	case errors.Is(err, ErrActiveQueueOrMatch):
		return apphttp.NewAPIError(http.StatusConflict, CodeActiveQueueOrMatch, MsgActiveQueueOrMatch).WithCause(err)
	case errors.Is(err, ErrPartyFull):
		return apphttp.NewAPIError(http.StatusConflict, CodePartyFull, MsgPartyFull).WithCause(err)
	case errors.Is(err, ErrAlreadyInvited):
		return apphttp.NewAPIError(http.StatusConflict, CodeAlreadyInvited, MsgAlreadyInvited).WithCause(err)
	case errors.Is(err, ErrTargetBusy):
		return apphttp.NewAPIError(http.StatusConflict, CodeTargetBusy, MsgTargetBusy).WithCause(err)
	case errors.Is(err, ErrPartyLocked):
		return apphttp.NewAPIError(http.StatusConflict, CodePartyLocked, MsgPartyLocked).WithCause(err)
	case errors.Is(err, ErrIdempotencyConflict):
		return apphttp.NewAPIError(http.StatusConflict, CodeIdempotencyConflict, MsgIdempotencyConflict).WithCause(err)
	case errors.Is(err, ErrUnsupportedFormat):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeUnsupportedFormat, MsgUnsupportedFormat).WithCause(err)
	case errors.Is(err, ErrPartyIncomplete):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodePartyIncomplete, MsgPartyIncomplete).WithCause(err)
	case errors.Is(err, ErrPartyNotReady), errors.Is(err, ErrVersionMismatch):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodePartyNotReady, MsgPartyNotReady).WithCause(err)
	case errors.Is(err, ErrFeatureDisabled):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	case errors.Is(err, ErrRateLimited):
		return apphttp.NewAPIError(http.StatusTooManyRequests, apphttp.ErrCodeRateLimited, apphttp.MsgRateLimited).WithCause(err)
	case errors.Is(err, ErrDependencyFailure):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	default:
		return err
	}
}

// MapError is an alias of ToAPIError for packages that expect MapError naming.
func MapError(err error) error { return ToAPIError(err) }
