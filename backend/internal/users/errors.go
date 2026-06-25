package users

import "errors"

// Domain errors used by the users package.
var (
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidUserID = errors.New("invalid user id")
)
