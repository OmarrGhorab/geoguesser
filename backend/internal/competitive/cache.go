package competitive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DefaultLeaderboardCacheTTL is the maximum staleness for top-500/page cache.
const DefaultLeaderboardCacheTTL = 60 * time.Second

// pageCache is the Redis surface used by competitive read services.
type pageCache interface {
	GetLeaderboard(ctx context.Context, key string) (*LeaderboardResponse, error)
	SetLeaderboard(ctx context.Context, key string, response *LeaderboardResponse) error
	GetProfile(ctx context.Context, key string) (*ProfileResponse, error)
	SetProfile(ctx context.Context, key string, response *ProfileResponse) error
	Version(ctx context.Context, seasonID uuid.UUID) (int64, error)
	InvalidateSeason(ctx context.Context, seasonID uuid.UUID) error
}

// RedisPageCache stores short-lived competitive profile/leaderboard pages.
type RedisPageCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisPageCache constructs a competitive Redis page cache with ≤60s TTL.
func NewRedisPageCache(client *redis.Client) *RedisPageCache {
	return &RedisPageCache{client: client, ttl: DefaultLeaderboardCacheTTL}
}

// WithTTL overrides the cache TTL (tests); values above 60s are capped.
func (c *RedisPageCache) WithTTL(ttl time.Duration) *RedisPageCache {
	if c == nil {
		return c
	}
	if ttl <= 0 {
		ttl = DefaultLeaderboardCacheTTL
	}
	if ttl > DefaultLeaderboardCacheTTL {
		ttl = DefaultLeaderboardCacheTTL
	}
	c.ttl = ttl
	return c
}

// GetLeaderboard returns a cached leaderboard page or nil on miss.
func (c *RedisPageCache) GetLeaderboard(ctx context.Context, key string) (*LeaderboardResponse, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	raw, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get competitive leaderboard cache: %w", err)
	}
	var resp LeaderboardResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("decode competitive leaderboard cache: %w", err)
	}
	return &resp, nil
}

// SetLeaderboard stores a leaderboard page under key.
func (c *RedisPageCache) SetLeaderboard(ctx context.Context, key string, response *LeaderboardResponse) error {
	if c == nil || c.client == nil || response == nil {
		return nil
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode competitive leaderboard cache: %w", err)
	}
	if err := c.client.Set(ctx, key, encoded, c.ttl).Err(); err != nil {
		return fmt.Errorf("set competitive leaderboard cache: %w", err)
	}
	return nil
}

// GetProfile returns a cached profile page or nil on miss.
func (c *RedisPageCache) GetProfile(ctx context.Context, key string) (*ProfileResponse, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	raw, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get competitive profile cache: %w", err)
	}
	var resp ProfileResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("decode competitive profile cache: %w", err)
	}
	return &resp, nil
}

// SetProfile stores a profile page under key.
func (c *RedisPageCache) SetProfile(ctx context.Context, key string, response *ProfileResponse) error {
	if c == nil || c.client == nil || response == nil {
		return nil
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode competitive profile cache: %w", err)
	}
	if err := c.client.Set(ctx, key, encoded, c.ttl).Err(); err != nil {
		return fmt.Errorf("set competitive profile cache: %w", err)
	}
	return nil
}

// Version returns the monotonic invalidation generation for a season (default 1).
func (c *RedisPageCache) Version(ctx context.Context, seasonID uuid.UUID) (int64, error) {
	if c == nil || c.client == nil || seasonID == uuid.Nil {
		return 1, nil
	}
	version, err := c.client.Get(ctx, seasonVersionKey(seasonID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get competitive cache version: %w", err)
	}
	if version < 1 {
		return 1, nil
	}
	return version, nil
}

// InvalidateSeason bumps the season cache generation so prior keys miss.
func (c *RedisPageCache) InvalidateSeason(ctx context.Context, seasonID uuid.UUID) error {
	if c == nil || c.client == nil || seasonID == uuid.Nil {
		return nil
	}
	if err := c.client.Incr(ctx, seasonVersionKey(seasonID)).Err(); err != nil {
		return fmt.Errorf("bump competitive cache version: %w", err)
	}
	return nil
}

func seasonVersionKey(seasonID uuid.UUID) string {
	return "competitive:v1:season:" + seasonID.String() + ":version"
}

// LeaderboardCacheKey builds a stable cache key including version and page identity.
func LeaderboardCacheKey(seasonID uuid.UUID, version int64, limit int, cursor, viewerID string) string {
	return fmt.Sprintf("competitive:v1:lb:%s:v%d:l%d:c:%s:u:%s",
		seasonID.String(), version, limit, cursor, viewerID)
}

// ProfileCacheKey builds a stable profile cache key.
func ProfileCacheKey(seasonID, userID uuid.UUID, version int64) string {
	return fmt.Sprintf("competitive:v1:profile:%s:v%d:u:%s",
		seasonID.String(), version, userID.String())
}

// ClosedLeaderboardCacheKey builds a key for frozen closed-season top 500 pages.
func ClosedLeaderboardCacheKey(seasonID uuid.UUID, version int64, limit int, cursor, viewerID string) string {
	return fmt.Sprintf("competitive:v1:closed-lb:%s:v%d:l%d:c:%s:u:%s",
		seasonID.String(), version, limit, cursor, viewerID)
}

// MemoryPageCache is an in-process cache for unit tests (no Redis).
type MemoryPageCache struct {
	ttl      time.Duration
	versions map[uuid.UUID]int64
	lb       map[string]cachedLB
	profiles map[string]cachedProfile
	now      func() time.Time
}

type cachedLB struct {
	resp      LeaderboardResponse
	expiresAt time.Time
}

type cachedProfile struct {
	resp      ProfileResponse
	expiresAt time.Time
}

// NewMemoryPageCache constructs a test double cache.
func NewMemoryPageCache(ttl time.Duration) *MemoryPageCache {
	if ttl <= 0 || ttl > DefaultLeaderboardCacheTTL {
		ttl = DefaultLeaderboardCacheTTL
	}
	return &MemoryPageCache{
		ttl:      ttl,
		versions: map[uuid.UUID]int64{},
		lb:       map[string]cachedLB{},
		profiles: map[string]cachedProfile{},
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// GetLeaderboard implements pageCache.
func (c *MemoryPageCache) GetLeaderboard(_ context.Context, key string) (*LeaderboardResponse, error) {
	if c == nil {
		return nil, nil
	}
	entry, ok := c.lb[key]
	if !ok || !entry.expiresAt.After(c.now()) {
		return nil, nil
	}
	copyResp := entry.resp
	return &copyResp, nil
}

// SetLeaderboard implements pageCache.
func (c *MemoryPageCache) SetLeaderboard(_ context.Context, key string, response *LeaderboardResponse) error {
	if c == nil || response == nil {
		return nil
	}
	c.lb[key] = cachedLB{resp: *response, expiresAt: c.now().Add(c.ttl)}
	return nil
}

// GetProfile implements pageCache.
func (c *MemoryPageCache) GetProfile(_ context.Context, key string) (*ProfileResponse, error) {
	if c == nil {
		return nil, nil
	}
	entry, ok := c.profiles[key]
	if !ok || !entry.expiresAt.After(c.now()) {
		return nil, nil
	}
	copyResp := entry.resp
	return &copyResp, nil
}

// SetProfile implements pageCache.
func (c *MemoryPageCache) SetProfile(_ context.Context, key string, response *ProfileResponse) error {
	if c == nil || response == nil {
		return nil
	}
	c.profiles[key] = cachedProfile{resp: *response, expiresAt: c.now().Add(c.ttl)}
	return nil
}

// Version implements pageCache.
func (c *MemoryPageCache) Version(_ context.Context, seasonID uuid.UUID) (int64, error) {
	if c == nil {
		return 1, nil
	}
	if v, ok := c.versions[seasonID]; ok && v >= 1 {
		return v, nil
	}
	return 1, nil
}

// InvalidateSeason implements pageCache.
func (c *MemoryPageCache) InvalidateSeason(_ context.Context, seasonID uuid.UUID) error {
	if c == nil {
		return nil
	}
	v, ok := c.versions[seasonID]
	if !ok || v < 1 {
		v = 1
	}
	c.versions[seasonID] = v + 1
	return nil
}
