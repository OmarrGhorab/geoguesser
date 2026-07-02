package leaderboards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const leaderboardCacheTTL = 30 * time.Second

type pageCache interface {
	Get(ctx context.Context, key string) (*Response, error)
	Set(ctx context.Context, key string, response *Response) error
	Version(ctx context.Context, scope string) (int64, error)
	BumpVersion(ctx context.Context, scope string) error
}

// RedisPageCache stores short-lived public leaderboard pages as cache-aside data.
type RedisPageCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisPageCache(client *redis.Client) *RedisPageCache {
	return &RedisPageCache{client: client, ttl: leaderboardCacheTTL}
}

func (c *RedisPageCache) Get(ctx context.Context, key string) (*Response, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	raw, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get leaderboard cache: %w", err)
	}
	var cached cachedResponse
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, fmt.Errorf("decode leaderboard cache: %w", err)
	}
	return &Response{Data: cached.Data, Page: pageInfo(cached.Limit, cached.NextCursor)}, nil
}

func (c *RedisPageCache) Set(ctx context.Context, key string, response *Response) error {
	if c == nil || c.client == nil || response == nil {
		return nil
	}
	payload := cachedResponse{
		Data:       response.Data,
		Limit:      response.Page.Limit,
		NextCursor: response.Page.NextCursor,
		CachedAt:   time.Now().UTC(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode leaderboard cache: %w", err)
	}
	if err := c.client.Set(ctx, key, encoded, c.ttl).Err(); err != nil {
		return fmt.Errorf("set leaderboard cache: %w", err)
	}
	return nil
}

func (c *RedisPageCache) Version(ctx context.Context, scope string) (int64, error) {
	if c == nil || c.client == nil {
		return 1, nil
	}
	version, err := c.client.Get(ctx, versionKey(scope)).Int64()
	if errors.Is(err, redis.Nil) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get leaderboard cache version: %w", err)
	}
	if version < 1 {
		return 1, nil
	}
	return version, nil
}

func (c *RedisPageCache) BumpVersion(ctx context.Context, scope string) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Incr(ctx, versionKey(scope)).Err(); err != nil {
		return fmt.Errorf("bump leaderboard cache version: %w", err)
	}
	return nil
}

func versionKey(scope string) string {
	return "leaderboard:v1:" + scope + ":version"
}
