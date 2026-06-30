package realtime

import "errors"

var (
	ErrOriginForbidden = errors.New("realtime origin forbidden")
	ErrAuthRequired    = errors.New("realtime auth required")
	ErrNotParticipant  = errors.New("realtime participant required")
)

const (
	CodeOriginForbidden = "realtime_origin_forbidden"
	CodeAuthRequired    = "realtime_auth_required"
)
