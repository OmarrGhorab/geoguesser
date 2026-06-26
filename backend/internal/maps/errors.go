package maps

import "errors"

var (
	ErrMapNotFound   = errors.New("map not found")
	ErrInvalidMapID  = errors.New("invalid map id")
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidFilter = errors.New("invalid filter")
)
