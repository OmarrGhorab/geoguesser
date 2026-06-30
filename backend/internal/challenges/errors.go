package challenges

import "errors"

var (
	ErrChallengeNotFound     = errors.New("challenge not found")
	ErrInvalidChallengeInput = errors.New("invalid challenge input")
	ErrChallengeUnavailable  = errors.New("challenge unavailable")
	ErrForbidden             = errors.New("forbidden")
	ErrNotEnoughLocations    = errors.New("not enough locations")
	ErrDuplicateAttempt      = errors.New("duplicate challenge attempt")
	ErrResultsNotReady       = errors.New("challenge results not ready")
	ErrIdempotencyConflict   = errors.New("idempotency conflict")
)
