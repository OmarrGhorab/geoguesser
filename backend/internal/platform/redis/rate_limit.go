package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const allowSlidingWindowLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowStart = tonumber(ARGV[2])
local member = ARGV[3]
local windowSeconds = tonumber(ARGV[4])

redis.call('ZREMRANGEBYSCORE', key, 0, windowStart)
local count = redis.call('ZCARD', key)
redis.call('ZADD', key, now, member)
redis.call('EXPIRE', key, windowSeconds)

return count + 1
`

// RateLimiter provides an atomic sliding-window rate limit backed by Redis.
type RateLimiter struct {
	client *redis.Client
	script *redis.Script
}

// NewRateLimiter returns a Redis-backed rate limiter.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
		script: redis.NewScript(allowSlidingWindowLua),
	}
}

// Allow checks whether a request is allowed under the given key and limit. It
// records the current attempt atomically and returns true if the request is
// within the maximum allowed attempts for the window.
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	now := time.Now().UTC()
	windowStart := now.Add(-window)
	member := now.UnixNano()

	res, err := r.script.Run(
		ctx,
		r.client,
		[]string{key},
		now.UnixMilli(),
		windowStart.UnixMilli(),
		member,
		int(window.Seconds()),
	).Result()
	if err != nil {
		return false, 0, fmt.Errorf("rate limit script failed: %w", err)
	}

	count, ok := res.(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected rate limit script result type %T", res)
	}
	return int(count) <= limit, int(count), nil
}
