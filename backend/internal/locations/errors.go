package locations

import "errors"

var (
	ErrLocationNotFound  = errors.New("location not found")
	ErrInvalidLocationID = errors.New("invalid location id")
	ErrMediaAccessDenied = errors.New("media access denied")
	ErrMediaUnavailable  = errors.New("media unavailable")
)
