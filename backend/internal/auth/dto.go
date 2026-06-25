package auth

import (
	"time"

	"github.com/raven/geoguess/backend/internal/session"
)

// RegisterRequest is the payload for email/password registration.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginRequest is the payload for email/password login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ForgotPasswordRequest is the payload to request a password reset OTP.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the payload to reset a password with an OTP.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

// AuthResponse returns the authenticated user.
type AuthResponse struct {
	User UserDTO `json:"user"`
}

// UserDTO is the public representation of a registered user.
type UserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// SessionResponse returns the current session summary.
type SessionResponse struct {
	Kind          session.Kind `json:"kind"`
	Authenticated bool         `json:"authenticated"`
	User          *UserDTO     `json:"user,omitempty"`
	Guest         *GuestDTO    `json:"guest,omitempty"`
}

// GuestDTO is the public representation of a guest session.
type GuestDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// TokenPair holds issued access and refresh tokens.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}
