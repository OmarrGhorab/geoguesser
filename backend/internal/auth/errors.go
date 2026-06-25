package auth

import "errors"

// Domain errors used by the auth service and mapped to HTTP errors by handlers.
var (
	ErrEmailRequired        = errors.New("email is required")
	ErrInvalidEmail         = errors.New("email is invalid")
	ErrPasswordRequired     = errors.New("password is required")
	ErrPasswordTooShort     = errors.New("password is too short")
	ErrDisplayNameRequired  = errors.New("display_name is required")
	ErrDisplayNameLength    = errors.New("display_name must be between 2 and 32 characters")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
	ErrSessionRevoked       = errors.New("session revoked")
	ErrTokenReuseDetected   = errors.New("refresh token reuse detected")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrInvalidAccessToken   = errors.New("invalid access token")
	ErrInvalidCSRFToken     = errors.New("invalid csrf token")
	ErrInvalidOAuthProvider = errors.New("invalid oauth provider")
	ErrOAuthStateMismatch   = errors.New("oauth state mismatch")
	ErrOAuthAccountLinked   = errors.New("oauth account already linked to another user")
	ErrGuestSessionInvalid  = errors.New("guest session invalid")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidOTP           = errors.New("invalid or expired otp")
	ErrOTPRateLimited       = errors.New("otp requests rate limited")
	ErrSamePassword         = errors.New("new password must be different from old password")
)
