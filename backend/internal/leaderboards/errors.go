package leaderboards

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

var (
	ErrUnauthorized        = errors.New("leaderboard unauthorized")
	ErrLeaderboardNotFound = errors.New("leaderboard not found")
	ErrInvalidLimit        = errors.New("invalid leaderboard limit")
	ErrInvalidCursor       = errors.New("invalid leaderboard cursor")
	ErrInvalidDate         = errors.New("invalid leaderboard date")
	ErrInvalidMapID        = errors.New("invalid map id")
	ErrDependencyFailure   = errors.New("leaderboard dependency failure")
)

func ToAPIError(err error) error {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized.WithCause(err)
	case errors.Is(err, ErrLeaderboardNotFound):
		return apphttp.ErrNotFound.WithCause(err)
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidCursor), errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidMapID):
		return apphttp.ErrValidationFailed.WithCause(err)
	case errors.Is(err, ErrDependencyFailure):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, "leaderboards_unavailable", "Leaderboards are temporarily unavailable.").WithCause(err)
	default:
		return err
	}
}
