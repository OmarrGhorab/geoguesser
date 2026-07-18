package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Command idempotency Redis key layout:
//
//	cmd:idempotency:v1:{sha256(scope|callerID|idemKey)}
//
// Hash fields: hash (request body fingerprint), status (pending|done), response (bytes as string).
const commandIdempotencyKeyPrefix = "cmd:idempotency:v1:"

// Sentinel errors for callers that prefer error checks over BeginResult flags.
var (
	ErrCommandIdempotencyConflict = errors.New("command idempotency conflict")
	ErrCommandIdempotencyInFlight = errors.New("command idempotency in flight")
)

// begin script:
//
//	0 = claimed (proceed)
//	1 = hit with stored response (ARGV carries response as return bulk)
//	2 = conflict (different body hash)
//	3 = in-flight (same or unknown incomplete claim)
//
// KEYS[1] = storage key
// ARGV[1] = body hash
// ARGV[2] = TTL seconds
var commandIdempotencyBeginScript = redis.NewScript(`
local key = KEYS[1]
local bodyHash = ARGV[1]
local ttl = tonumber(ARGV[2])
local existing = redis.call('HMGET', key, 'hash', 'status', 'response')
if existing[1] then
  if existing[1] ~= bodyHash then
    return {2}
  end
  if existing[2] == 'done' and existing[3] then
    return {1, existing[3]}
  end
  return {3}
end
redis.call('HSET', key, 'hash', bodyHash, 'status', 'pending')
redis.call('EXPIRE', key, ttl)
return {0}
`)

// complete script:
//
//	0 = stored
//	2 = conflict (different body hash present)
//
// KEYS[1] = storage key
// ARGV[1] = body hash
// ARGV[2] = response payload
// ARGV[3] = TTL seconds
var commandIdempotencyCompleteScript = redis.NewScript(`
local key = KEYS[1]
local bodyHash = ARGV[1]
local response = ARGV[2]
local ttl = tonumber(ARGV[3])
local existing = redis.call('HGET', key, 'hash')
if existing and existing ~= bodyHash then
  return 2
end
redis.call('HSET', key, 'hash', bodyHash, 'status', 'done', 'response', response)
redis.call('EXPIRE', key, ttl)
return 0
`)

// CommandIdempotencyStore stores short-lived command request fingerprints and
// successful response snapshots for party/queue/message/report/upload/leave flows.
// PostgreSQL remains the durable source of truth for business facts.
type CommandIdempotencyStore struct {
	client *redis.Client
}

// BeginResult describes the outcome of Begin.
//
//	Hit=true:           same body; Response holds the stored snapshot (replay).
//	Conflict=true:      different body for the same key, or in-flight without result.
//	Hit=false,Conflict=false: claim acquired; caller should execute and Complete.
type BeginResult struct {
	Hit      bool
	Conflict bool
	InFlight bool
	Response []byte
}

// NewCommandIdempotencyStore returns a Redis-backed command idempotency store.
func NewCommandIdempotencyStore(client *redis.Client) *CommandIdempotencyStore {
	return &CommandIdempotencyStore{client: client}
}

// CommandIdempotencyKey builds the Redis key for a scoped caller command.
// The raw idempotency key is hashed so arbitrary client strings stay safe/bounded.
func CommandIdempotencyKey(scope, callerID, idemKey string) string {
	sum := sha256.Sum256([]byte(
		strings.TrimSpace(scope) + "|" +
			strings.TrimSpace(callerID) + "|" +
			strings.TrimSpace(idemKey),
	))
	return commandIdempotencyKeyPrefix + hex.EncodeToString(sum[:])
}

// Begin claims or replays an idempotent command.
// ttl controls both pending claim and completed record lifetime when first set.
func (s *CommandIdempotencyStore) Begin(
	ctx context.Context,
	scope, callerID, idemKey, bodyHash string,
	ttl time.Duration,
) (BeginResult, error) {
	if s == nil || s.client == nil {
		return BeginResult{}, nil
	}
	scope = strings.TrimSpace(scope)
	callerID = strings.TrimSpace(callerID)
	idemKey = strings.TrimSpace(idemKey)
	bodyHash = strings.TrimSpace(bodyHash)
	if scope == "" || callerID == "" || idemKey == "" || bodyHash == "" {
		return BeginResult{}, fmt.Errorf("command idempotency begin: scope, caller, key, and body hash are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	ttlSeconds := int(ttl / time.Second)
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	key := CommandIdempotencyKey(scope, callerID, idemKey)
	raw, err := commandIdempotencyBeginScript.Run(ctx, s.client, []string{key}, bodyHash, ttlSeconds).Result()
	if err != nil {
		return BeginResult{}, fmt.Errorf("command idempotency begin: %w", err)
	}
	return parseBeginResult(raw)
}

// Complete stores the successful response snapshot for a previously claimed key.
func (s *CommandIdempotencyStore) Complete(
	ctx context.Context,
	scope, callerID, idemKey, bodyHash string,
	response []byte,
	ttl time.Duration,
) error {
	if s == nil || s.client == nil {
		return nil
	}
	scope = strings.TrimSpace(scope)
	callerID = strings.TrimSpace(callerID)
	idemKey = strings.TrimSpace(idemKey)
	bodyHash = strings.TrimSpace(bodyHash)
	if scope == "" || callerID == "" || idemKey == "" || bodyHash == "" {
		return fmt.Errorf("command idempotency complete: scope, caller, key, and body hash are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	ttlSeconds := int(ttl / time.Second)
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	key := CommandIdempotencyKey(scope, callerID, idemKey)
	raw, err := commandIdempotencyCompleteScript.Run(
		ctx,
		s.client,
		[]string{key},
		bodyHash,
		string(response),
		ttlSeconds,
	).Result()
	if err != nil {
		return fmt.Errorf("command idempotency complete: %w", err)
	}
	code, ok := raw.(int64)
	if !ok {
		return fmt.Errorf("command idempotency complete: unexpected result type %T", raw)
	}
	if code == 2 {
		return ErrCommandIdempotencyConflict
	}
	if code != 0 {
		return fmt.Errorf("command idempotency complete: unexpected code %d", code)
	}
	return nil
}

// Release drops a pending claim so a failed command can be retried with the same key.
// Completed records are left intact (Delete only if still pending).
func (s *CommandIdempotencyStore) Release(ctx context.Context, scope, callerID, idemKey string) error {
	if s == nil || s.client == nil {
		return nil
	}
	key := CommandIdempotencyKey(scope, callerID, idemKey)
	// Only delete if still pending so a raced Complete is preserved.
	script := redis.NewScript(`
local status = redis.call('HGET', KEYS[1], 'status')
if status == 'pending' then
  return redis.call('DEL', KEYS[1])
end
return 0
`)
	if err := script.Run(ctx, s.client, []string{key}).Err(); err != nil {
		return fmt.Errorf("command idempotency release: %w", err)
	}
	return nil
}

func parseBeginResult(raw any) (BeginResult, error) {
	arr, ok := raw.([]any)
	if !ok || len(arr) < 1 {
		return BeginResult{}, fmt.Errorf("command idempotency begin: unexpected result type %T", raw)
	}
	code, ok := arr[0].(int64)
	if !ok {
		return BeginResult{}, fmt.Errorf("command idempotency begin: unexpected code type %T", arr[0])
	}
	switch code {
	case 0:
		return BeginResult{}, nil
	case 1:
		if len(arr) < 2 {
			return BeginResult{}, fmt.Errorf("command idempotency begin: hit missing response")
		}
		switch v := arr[1].(type) {
		case string:
			return BeginResult{Hit: true, Response: []byte(v)}, nil
		case []byte:
			return BeginResult{Hit: true, Response: v}, nil
		default:
			return BeginResult{}, fmt.Errorf("command idempotency begin: unexpected response type %T", arr[1])
		}
	case 2:
		return BeginResult{Conflict: true}, nil
	case 3:
		return BeginResult{Conflict: true, InFlight: true}, nil
	default:
		return BeginResult{}, fmt.Errorf("command idempotency begin: unexpected code %d", code)
	}
}
