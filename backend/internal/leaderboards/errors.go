package leaderboards

import (
	"errors"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

var (
	ErrLeaderboardNotFound = errors.New("leaderboard not found")
	ErrInvalidLimit        = errors.New("invalid leaderboard limit")
	ErrInvalidCursor       = errors.New("invalid leaderboard cursor")
	ErrInvalidDate         = errors.New("invalid leaderboard date")
	ErrInvalidMapID        = errors.New("invalid map id")
)

func ToAPIError(err error) error {
	switch {
	case errors.Is(err, ErrLeaderboardNotFound):
		return apphttp.ErrNotFound.WithCause(err)
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidCursor), errors.Is(err, ErrInvalidDate), errors.Is(err, ErrInvalidMapID):
		return apphttp.ErrValidationFailed.WithCause(err)
	default:
		return err
	}
}
