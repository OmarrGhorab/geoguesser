package redis

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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

// rateLimitSeq guarantees unique sorted-set members when wall-clock nanoseconds collide
// (common on Windows under tight loops).
var rateLimitSeq atomic.Uint64

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
	// Guaranteed-unique member: timestamp + process-local sequence + UUID so
	// concurrent/same-millisecond requests never overwrite the same ZSET member.
	member := fmt.Sprintf("%d-%d-%s", now.UnixNano(), rateLimitSeq.Add(1), uuid.NewString())

	// Redis EXPIRE is whole seconds; sub-second windows must still retain keys long enough
	// for the sliding window to be observed (ceil to at least 1s).
	expireSeconds := int((window + time.Second - 1) / time.Second)
	if expireSeconds < 1 {
		expireSeconds = 1
	}

	res, err := r.script.Run(
		ctx,
		r.client,
		[]string{key},
		now.UnixMilli(),
		windowStart.UnixMilli(),
		member,
		expireSeconds,
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
