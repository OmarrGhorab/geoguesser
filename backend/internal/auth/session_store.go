package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RefreshSession is a rotated refresh-token session.
type RefreshSession struct {
	UserID    uuid.UUID
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore manages refresh-token sessions.
type SessionStore interface {
	Create(ctx context.Context, tokenHash string, session *RefreshSession, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (*RefreshSession, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
	// MarkRevoked preserves rotation metadata for token-reuse detection.
	MarkRevoked(ctx context.Context, tokenHash string, userID uuid.UUID, ttl time.Duration) error
	// RevokedUserID returns the owning user for a revoked token hash.
	RevokedUserID(ctx context.Context, tokenHash string) (uuid.UUID, bool, error)
}

// RedisSessionStore stores refresh-token sessions in Redis with TTL.
type RedisSessionStore struct {
	client *redis.Client
}

// NewRedisSessionStore returns a Redis-backed session store.
func NewRedisSessionStore(client *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

// Create stores a new refresh session and indexes it by user.
func (r *RedisSessionStore) Create(ctx context.Context, tokenHash string, session *RefreshSession, ttl time.Duration) error {
	key := sessionKey(tokenHash)
	userSessionsKey := userSessionsKey(session.UserID)

	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, map[string]any{
		"user_id":    session.UserID.String(),
		"role":       session.Role,
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"expires_at": session.ExpiresAt.Format(time.RFC3339),
	})
	pipe.Expire(ctx, key, ttl)
	pipe.SAdd(ctx, userSessionsKey, tokenHash)
	pipe.Expire(ctx, userSessionsKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create redis session: %w", err)
	}
	return nil
}

// Get retrieves a refresh session by token hash.
func (r *RedisSessionStore) Get(ctx context.Context, tokenHash string) (*RefreshSession, error) {
	key := sessionKey(tokenHash)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis session: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	userID, err := uuid.Parse(result["user_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid user id in session: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339, result["created_at"])
	if err != nil {
		return nil, fmt.Errorf("invalid created_at in session: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, result["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at in session: %w", err)
	}

	return &RefreshSession{
		UserID:    userID,
		Role:      result["role"],
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

// Revoke removes a refresh session.
func (r *RedisSessionStore) Revoke(ctx context.Context, tokenHash string) error {
	key := sessionKey(tokenHash)

	// Get user id before deleting to clean up the index.
	userIDStr, err := r.client.HGet(ctx, key, "user_id").Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get session user id: %w", err)
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, userSessionsKeyUUIDString(userIDStr), tokenHash)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke redis session: %w", err)
	}
	return nil
}

// RevokeAll removes all refresh sessions for a user.
func (r *RedisSessionStore) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	key := userSessionsKey(userID)
	tokenHashes, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to list user sessions: %w", err)
	}

	pipe := r.client.Pipeline()
	for _, hash := range tokenHashes {
		pipe.Del(ctx, sessionKey(hash))
	}
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke all user sessions: %w", err)
	}
	return nil
}

// MarkRevoked stores the revoked token owner for reuse detection.
func (r *RedisSessionStore) MarkRevoked(ctx context.Context, tokenHash string, userID uuid.UUID, ttl time.Duration) error {
	key := revokedSessionKey(tokenHash)
	if err := r.client.Set(ctx, key, userID.String(), ttl).Err(); err != nil {
		return fmt.Errorf("failed to mark revoked session: %w", err)
	}
	return nil
}

// RevokedUserID reports whether a token hash was recently revoked.
func (r *RedisSessionStore) RevokedUserID(ctx context.Context, tokenHash string) (uuid.UUID, bool, error) {
	key := revokedSessionKey(tokenHash)
	value, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to check revoked session: %w", err)
	}
	userID, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("invalid user id in revoked session: %w", err)
	}
	return userID, true, nil
}

func sessionKey(tokenHash string) string {
	return fmt.Sprintf("session:%s", tokenHash)
}

func revokedSessionKey(tokenHash string) string {
	return fmt.Sprintf("revoked_session:%s", tokenHash)
}

func userSessionsKey(userID uuid.UUID) string {
	return fmt.Sprintf("user_sessions:%s", userID.String())
}

func userSessionsKeyUUIDString(userID string) string {
	return fmt.Sprintf("user_sessions:%s", userID)
}
