package friends

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors for the friends package.
var (
	ErrUnauthorized      = errors.New("friends unauthorized")
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidRequestID  = errors.New("invalid request id")
	ErrSelfPair          = errors.New("cannot socialize with self")
	ErrInvalidState      = errors.New("invalid friendship state")
	ErrInvalidCursor     = errors.New("invalid pagination cursor")
	ErrInvalidLimit      = errors.New("invalid pagination limit")
	ErrNotFound          = errors.New("friendship not found")
	ErrTargetNotFound    = errors.New("target user not found")
	ErrConflict          = errors.New("friendship conflict")
	ErrAlreadyFriends    = errors.New("already friends")
	ErrAlreadyPending    = errors.New("friend request already pending")
	ErrDependencyFailure = errors.New("friends dependency failure")
	ErrRateLimited       = errors.New("friends rate limited")
	ErrInvalidJSON       = errors.New("invalid friends json")
	ErrInvalidRequest    = errors.New("invalid friends request")
)

// Stable domain-specific API codes where shared codes are insufficient.
const (
	CodeAlreadyFriends = "already_friends"
	CodeAlreadyPending = "friend_request_already_pending"
	CodeUnavailable    = "friends_unavailable"
)

const (
	MsgSelfPair       = "You cannot perform this action on yourself."
	MsgAlreadyFriends = "You are already friends with this user."
	MsgAlreadyPending = "A friend request is already pending."
	MsgUnavailable    = "Friends features are temporarily unavailable."
)

// ToAPIError maps domain errors to stable HTTP API errors.
// Missing targets, inactive targets, and blocked-pair concealment use not_found.
func ToAPIError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized
	case errors.Is(err, ErrInvalidJSON):
		return apphttp.ErrInvalidJSON
	case errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrInvalidRequestID),
		errors.Is(err, ErrInvalidCursor), errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrSelfPair):
		if errors.Is(err, ErrSelfPair) {
			return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgSelfPair).WithCause(err)
		}
		return apphttp.ErrValidationFailed.WithCause(err)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrTargetNotFound):
		return apphttp.ErrNotFound
	case errors.Is(err, ErrAlreadyFriends):
		return apphttp.NewAPIError(http.StatusConflict, CodeAlreadyFriends, MsgAlreadyFriends).WithCause(err)
	case errors.Is(err, ErrAlreadyPending):
		return apphttp.NewAPIError(http.StatusConflict, CodeAlreadyPending, MsgAlreadyPending).WithCause(err)
	case errors.Is(err, ErrConflict):
		return apphttp.ErrConflict.WithCause(err)
	case errors.Is(err, ErrInvalidState):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, apphttp.ErrCodeUnprocessable, apphttp.MsgUnprocessable).WithCause(err)
	case errors.Is(err, ErrRateLimited):
		return apphttp.NewAPIError(http.StatusTooManyRequests, apphttp.ErrCodeRateLimited, apphttp.MsgRateLimited).WithCause(err)
	case errors.Is(err, ErrDependencyFailure):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	default:
		return err
	}
}
