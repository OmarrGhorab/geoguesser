package competitive

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors for competitive progression.
var (
	ErrNoActiveSeason     = errors.New("no active competitive season")
	ErrSeasonNotFound     = errors.New("competitive season not found")
	ErrMatchNotFound      = errors.New("competitive match not found")
	ErrMatchNotRanked     = errors.New("match is not ranked")
	ErrMatchNotTerminal   = errors.New("match is not terminal")
	ErrMatchNotReady      = errors.New("match is not ready for progression")
	ErrSeasonMismatch     = errors.New("match season does not match active season")
	ErrInvalidResult      = errors.New("invalid match result for progression")
	ErrProgressionPending = errors.New("progression pending")
	ErrDependencyFailure  = errors.New("competitive dependency failure")
	ErrInvalidInput       = errors.New("invalid competitive input")
	ErrInvalidLimit       = errors.New("invalid competitive limit")
	ErrInvalidCursor      = errors.New("invalid competitive cursor")
	ErrUnauthorized       = errors.New("competitive unauthorized")
	ErrFeatureDisabled    = errors.New("competitive feature disabled")
)

// Stable API codes used by handlers that surface progression state.
const (
	CodeProgressionPending = "progression_pending"
	CodeNoActiveSeason     = "no_active_season"
	CodeUnavailable        = "competitive_unavailable"
	CodeFeatureDisabled    = "feature_disabled"
	CodeInvalidCursor      = "invalid_cursor"
	CodeInvalidLimit       = "invalid_limit"
)

// Safe user-facing messages (frontend localizes).
const (
	MsgProgressionPending = "Competitive progression is still being finalized."
	MsgNoActiveSeason     = "Ranked play is unavailable because no competitive season is active."
	MsgUnavailable        = "Competitive features are temporarily unavailable."
	MsgMatchNotReady      = "The match is not ready for competitive progression."
	MsgFeatureDisabled    = "Competitive features are not enabled."
	MsgInvalidCursor      = "The pagination cursor is invalid."
	MsgInvalidLimit       = "The limit parameter is invalid."
	MsgUnauthorized       = "Authentication is required."
)

// ToAPIError maps domain errors to stable HTTP API errors.
func ToAPIError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrProgressionPending):
		return apphttp.NewAPIError(http.StatusAccepted, CodeProgressionPending, MsgProgressionPending).WithCause(err)
	case errors.Is(err, ErrNoActiveSeason):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeNoActiveSeason, MsgNoActiveSeason).WithCause(err)
	case errors.Is(err, ErrMatchNotFound), errors.Is(err, ErrSeasonNotFound):
		return apphttp.ErrNotFound
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized
	case errors.Is(err, ErrFeatureDisabled):
		return apphttp.NewAPIError(http.StatusNotFound, CodeFeatureDisabled, MsgFeatureDisabled).WithCause(err)
	case errors.Is(err, ErrInvalidCursor):
		return apphttp.NewAPIError(http.StatusBadRequest, CodeInvalidCursor, MsgInvalidCursor).WithCause(err)
	case errors.Is(err, ErrInvalidLimit):
		return apphttp.NewAPIError(http.StatusBadRequest, CodeInvalidLimit, MsgInvalidLimit).WithCause(err)
	case errors.Is(err, ErrMatchNotRanked), errors.Is(err, ErrMatchNotTerminal),
		errors.Is(err, ErrMatchNotReady), errors.Is(err, ErrInvalidResult),
		errors.Is(err, ErrSeasonMismatch), errors.Is(err, ErrInvalidInput):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, apphttp.ErrCodeUnprocessable, MsgMatchNotReady).WithCause(err)
	case errors.Is(err, ErrDependencyFailure):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	default:
		return err
	}
}

// MapError is an alias of ToAPIError.
func MapError(err error) error { return ToAPIError(err) }

// IsRetryable reports whether the error should be retried by the progression worker.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrProgressionPending) ||
		errors.Is(err, ErrMatchNotReady) ||
		errors.Is(err, ErrDependencyFailure) ||
		errors.Is(err, ErrNoActiveSeason)
}
