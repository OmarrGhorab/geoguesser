package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RoomCoordinator struct {
	client *redis.Client
}

func NewRoomCoordinator(client *redis.Client) *RoomCoordinator {
	return &RoomCoordinator{client: client}
}

func (c *RoomCoordinator) IncrementVersion(ctx context.Context, roomCode string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, nil
	}
	version, err := c.client.Incr(ctx, roomVersionKey(roomCode)).Result()
	if err != nil {
		return 0, fmt.Errorf("increment room version: %w", err)
	}
	return version, nil
}

func (c *RoomCoordinator) GetVersion(ctx context.Context, roomCode string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, nil
	}
	version, err := c.client.Get(ctx, roomVersionKey(roomCode)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get room version: %w", err)
	}
	return version, nil
}

func (c *RoomCoordinator) StoreSnapshot(ctx context.Context, roomCode string, snapshot any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal room snapshot: %w", err)
	}
	if err := c.client.Set(ctx, roomSnapshotKey(roomCode), payload, ttl).Err(); err != nil {
		return fmt.Errorf("store room snapshot: %w", err)
	}
	return nil
}

func (c *RoomCoordinator) GetSnapshot(ctx context.Context, roomCode string, dst any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	payload, err := c.client.Get(ctx, roomSnapshotKey(roomCode)).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get room snapshot: %w", err)
	}
	if err := json.Unmarshal(payload, dst); err != nil {
		return false, fmt.Errorf("unmarshal room snapshot: %w", err)
	}
	return true, nil
}

func (c *RoomCoordinator) SetPresence(ctx context.Context, roomCode string, playerID uuid.UUID, status string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Set(ctx, roomPresenceKey(roomCode, playerID), status, ttl).Err(); err != nil {
		return fmt.Errorf("set room presence: %w", err)
	}
	return nil
}

func (c *RoomCoordinator) GetPresence(ctx context.Context, roomCode string, playerID uuid.UUID) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}
	status, err := c.client.Get(ctx, roomPresenceKey(roomCode, playerID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get room presence: %w", err)
	}
	return status, nil
}

func (c *RoomCoordinator) SetReconnectWindow(ctx context.Context, roomCode string, playerID uuid.UUID, lastVersion int64, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	key := roomReconnectKey(roomCode, playerID)
	if err := c.client.HSet(ctx, key, "last_version", lastVersion).Err(); err != nil {
		return fmt.Errorf("set room reconnect window: %w", err)
	}
	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("expire room reconnect window: %w", err)
	}
	return nil
}

func (c *RoomCoordinator) GetReconnectWindow(ctx context.Context, roomCode string, playerID uuid.UUID) (int64, bool, error) {
	if c == nil || c.client == nil {
		return 0, false, nil
	}
	value, err := c.client.HGet(ctx, roomReconnectKey(roomCode, playerID), "last_version").Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get room reconnect window: %w", err)
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse room reconnect version: %w", err)
	}
	return version, true, nil
}

func (c *RoomCoordinator) GetReadyPlayerIDs(ctx context.Context, roomCode string) ([]uuid.UUID, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	values, err := c.client.SMembers(ctx, roomReadyKey(roomCode)).Result()
	if err != nil {
		return nil, fmt.Errorf("get room ready state: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *RoomCoordinator) SetReady(ctx context.Context, roomCode string, playerID uuid.UUID, ready bool) error {
	if c == nil || c.client == nil {
		return nil
	}
	key := roomReadyKey(roomCode)
	if ready {
		return c.client.SAdd(ctx, key, playerID.String()).Err()
	}
	return c.client.SRem(ctx, key, playerID.String()).Err()
}

func (c *RoomCoordinator) ClearReady(ctx context.Context, roomCode string) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, roomReadyKey(roomCode)).Err()
}

func (c *RoomCoordinator) ClaimCommand(ctx context.Context, roomCode, command, idempotencyKey string, ttl time.Duration) (bool, error) {
	if c == nil || c.client == nil {
		return true, nil
	}
	ok, err := c.client.SetNX(ctx, roomCommandKey(roomCode, command, idempotencyKey), "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("claim room command: %w", err)
	}
	return ok, nil
}

func (c *RoomCoordinator) Publish(ctx context.Context, roomCode string, event any) error {
	if c == nil || c.client == nil {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal room event: %w", err)
	}
	if err := c.client.Publish(ctx, roomEventsKey(roomCode), payload).Err(); err != nil {
		return fmt.Errorf("publish room event: %w", err)
	}
	return nil
}

func (c *RoomCoordinator) ClaimLock(ctx context.Context, roomCode, purpose string, ttl time.Duration) (bool, func(context.Context) error, error) {
	if c == nil || c.client == nil {
		return true, func(context.Context) error { return nil }, nil
	}
	key := roomLockKey(roomCode, purpose)
	ok, err := c.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, nil, fmt.Errorf("claim room lock: %w", err)
	}
	release := func(releaseCtx context.Context) error {
		if err := c.client.Del(releaseCtx, key).Err(); err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("release room lock: %w", err)
		}
		return nil
	}
	return ok, release, nil
}

func roomSnapshotKey(roomCode string) string {
	return "rooms:" + roomCode + ":snapshot"
}

func roomVersionKey(roomCode string) string {
	return "rooms:" + roomCode + ":version"
}

func roomPresenceKey(roomCode string, playerID uuid.UUID) string {
	return "rooms:" + roomCode + ":presence:" + playerID.String()
}

func roomReconnectKey(roomCode string, playerID uuid.UUID) string {
	return "rooms:" + roomCode + ":reconnect:" + playerID.String()
}

func roomReadyKey(roomCode string) string {
	return "rooms:" + roomCode + ":ready"
}

func roomLockKey(roomCode, purpose string) string {
	return "rooms:" + roomCode + ":lock:" + purpose
}

func roomCommandKey(roomCode, command, idempotencyKey string) string {
	return "rooms:" + roomCode + ":command:" + command + ":" + idempotencyKey
}

func roomEventsKey(roomCode string) string {
	return "rooms:" + roomCode + ":events"
}
