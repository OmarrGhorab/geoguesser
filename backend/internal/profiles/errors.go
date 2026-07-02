package profiles

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors used by the profiles package. Handlers map these to stable
// HTTP error responses; callers should use errors.Is to check them.
var (
	ErrUnauthorized     = errors.New("registered session required")
	ErrForbidden        = errors.New("csrf validation failed")
	ErrProfileNotFound  = errors.New("profile not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidUserID    = errors.New("invalid user id")
	ErrInvalidCursor    = errors.New("invalid pagination cursor")
	ErrInvalidLimit     = errors.New("invalid pagination limit")
	ErrValidationFailed = errors.New("profile validation failed")
	ErrRateLimited      = errors.New("too many profile update attempts")
)

// ValidationError carries field-level validation failures for a profile
// update request.
type ValidationError struct {
	Fields []apphttp.FieldError
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return "profile validation failed"
}

// Unwrap allows errors.Is(err, ErrValidationFailed) to succeed for
// ValidationError values.
func (e *ValidationError) Unwrap() error {
	return ErrValidationFailed
}

// ToAPIError maps a profiles domain error to a stable API error response.
// Unrecognized errors are treated as internal errors by the shared
// apphttp.Error helper, so this only needs to cover known domain errors.
func ToAPIError(err error) error {
	var apiErr *apphttp.APIError

	switch {
	case errors.As(err, new(*ValidationError)):
		var verr *ValidationError
		errors.As(err, &verr)
		apiErr = apphttp.ErrValidationFailed.WithFields(verr.Fields...)
	case errors.Is(err, ErrUnauthorized):
		apiErr = apphttp.ErrUnauthorized
	case errors.Is(err, ErrForbidden):
		apiErr = apphttp.ErrForbidden
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrUserNotFound), errors.Is(err, ErrInvalidUserID):
		apiErr = apphttp.ErrNotFound
	case errors.Is(err, ErrInvalidCursor), errors.Is(err, ErrInvalidLimit):
		apiErr = apphttp.ErrValidationFailed
	case errors.Is(err, ErrRateLimited):
		apiErr = apphttp.NewAPIError(http.StatusTooManyRequests, apphttp.ErrCodeRateLimited, apphttp.MsgRateLimited)
	default:
		return err
	}

	return apiErr.WithCause(err)
}
