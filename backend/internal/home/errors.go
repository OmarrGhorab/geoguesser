package home

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

var (
	// ErrUnauthorized conceals invalid, inactive, deleted, and missing viewers.
	ErrUnauthorized = errors.New("registered active session required")
	// ErrUnavailable reports a dependency failure without exposing its details.
	ErrUnavailable = errors.New("authenticated home unavailable")
)

const (
	CodeUnavailable = "home_unavailable"
	MsgUnavailable  = "The authenticated home is temporarily unavailable."
)

// ToAPIError maps home-domain errors to stable transport errors.
func ToAPIError(err error) *apphttp.APIError {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized.WithCause(err)
	case errors.Is(err, ErrUnavailable):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	default:
		return apphttp.ErrInternal.WithCause(err)
	}
}
