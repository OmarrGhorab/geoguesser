package matchmaking

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors for matchmaking operations.
var (
	ErrUnauthorized       = errors.New("matchmaking unauthorized")
	ErrAccountIneligible  = errors.New("matchmaking account ineligible")
	ErrUnsupportedMode    = errors.New("unsupported matchmaking mode")
	ErrAlreadyAssigned    = errors.New("matchmaking already assigned")
	ErrActiveGameConflict = errors.New("matchmaking active game conflict")
	ErrClaimInProgress    = errors.New("matchmaking claim in progress")
	ErrContentUnavailable = errors.New("matchmaking content unavailable")
	ErrUnavailable        = errors.New("matchmaking unavailable")
	ErrInvalidRequest     = errors.New("invalid matchmaking request")
	ErrInvalidJSON        = errors.New("invalid matchmaking json")
)

// Stable API error codes for matchmaking.
const (
	CodeUnsupportedMode    = "unsupported_matchmaking_mode"
	CodeAccountIneligible  = "matchmaking_account_ineligible"
	CodeAlreadyAssigned    = "matchmaking_already_assigned"
	CodeActiveGameConflict = "matchmaking_active_game_conflict"
	CodeClaimInProgress    = "matchmaking_claim_in_progress"
	CodeContentUnavailable = "matchmaking_content_unavailable"
	CodeUnavailable        = "matchmaking_unavailable"
)

// Safe user-facing messages (frontend localizes).
const (
	MsgUnsupportedMode    = "That matchmaking mode is not supported."
	MsgAccountIneligible  = "Your account is not eligible for ranked matchmaking."
	MsgAlreadyAssigned    = "You already have an active ranked assignment."
	MsgActiveGameConflict = "You already have an active game that conflicts with matchmaking."
	MsgClaimInProgress    = "A match assignment is currently being finalized."
	MsgContentUnavailable = "Ranked matchmaking content is temporarily unavailable."
	MsgUnavailable        = "Matchmaking is temporarily unavailable."
	MsgInvalidRequest     = "The matchmaking request is invalid."
)

// MapError converts domain errors into stable shared-envelope API errors.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized
	case errors.Is(err, ErrAccountIneligible):
		return apphttp.NewAPIError(http.StatusForbidden, CodeAccountIneligible, MsgAccountIneligible).WithCause(err)
	case errors.Is(err, ErrUnsupportedMode):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeUnsupportedMode, MsgUnsupportedMode).WithCause(err)
	case errors.Is(err, ErrAlreadyAssigned):
		return apphttp.NewAPIError(http.StatusConflict, CodeAlreadyAssigned, MsgAlreadyAssigned).WithCause(err)
	case errors.Is(err, ErrActiveGameConflict):
		return apphttp.NewAPIError(http.StatusConflict, CodeActiveGameConflict, MsgActiveGameConflict).WithCause(err)
	case errors.Is(err, ErrClaimInProgress):
		return apphttp.NewAPIError(http.StatusConflict, CodeClaimInProgress, MsgClaimInProgress).WithCause(err)
	case errors.Is(err, ErrContentUnavailable):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeContentUnavailable, MsgContentUnavailable).WithCause(err)
	case errors.Is(err, ErrUnavailable):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	case errors.Is(err, ErrInvalidJSON):
		return apphttp.ErrInvalidJSON
	case errors.Is(err, ErrInvalidRequest):
		return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgInvalidRequest).WithCause(err)
	default:
		return err
	}
}
