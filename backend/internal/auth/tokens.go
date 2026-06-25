package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenClaims are the JWT access token claims.
type TokenClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager handles access token signing/verification and refresh token generation.
type TokenManager struct {
	accessSecret []byte
	accessTTL    time.Duration
}

// NewTokenManager returns a token manager.
func NewTokenManager(accessSecret string, accessTTL time.Duration) (*TokenManager, error) {
	if len(accessSecret) < 32 {
		return nil, fmt.Errorf("access token secret must be at least 32 bytes")
	}
	return &TokenManager{
		accessSecret: []byte(accessSecret),
		accessTTL:    accessTTL,
	}, nil
}

// GenerateAccessToken issues a signed JWT access token.
func (t *TokenManager) GenerateAccessToken(userID uuid.UUID, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(t.accessTTL)
	claims := TokenClaims{
		UserID: userID.String(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.accessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// VerifyAccessToken validates and parses a signed JWT access token.
func (t *TokenManager) VerifyAccessToken(tokenString string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.accessSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify access token: %w", err)
	}
	if !token.Valid {
		return nil, ErrInvalidAccessToken
	}
	return claims, nil
}

// GenerateRefreshToken returns a new cryptographically secure refresh token and its hash.
func GenerateRefreshToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	raw := hex.EncodeToString(b)
	hash := HashRefreshToken(raw)
	return raw, hash, nil
}

// HashRefreshToken returns a SHA-256 hash of a refresh token.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
