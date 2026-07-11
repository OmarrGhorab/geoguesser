package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const matchmakingKeyVersion = "v1"

// MatchmakingCoordinator coordinates ephemeral ranked queue state in Redis.
type MatchmakingCoordinator struct {
	client *redis.Client
}

// NewMatchmakingCoordinator constructs a Redis matchmaking coordinator.
func NewMatchmakingCoordinator(client *redis.Client) *MatchmakingCoordinator {
	return &MatchmakingCoordinator{client: client}
}

// Queue entry field names stored in the player hash.
const (
	fieldEntryID        = "entry_id"
	fieldUserID         = "user_id"
	fieldMode           = "mode"
	fieldState          = "state"
	fieldEnqueuedAtMs   = "enqueued_at_ms"
	fieldLeaseExpiresAt = "lease_expires_at_ms"
	fieldClaimID        = "claim_id"
)

// Versioned key builders.
func matchmakingQueueKey(mode string) string {
	return fmt.Sprintf("matchmaking:%s:queue:%s", matchmakingKeyVersion, mode)
}

func matchmakingPlayerKey(userID uuid.UUID) string {
	return fmt.Sprintf("matchmaking:%s:player:%s", matchmakingKeyVersion, userID.String())
}

func matchmakingClaimKey(claimID string) string {
	return fmt.Sprintf("matchmaking:%s:claim:%s", matchmakingKeyVersion, claimID)
}

func matchmakingClaimsIndexKey() string {
	return fmt.Sprintf("matchmaking:%s:claims", matchmakingKeyVersion)
}

// QueueEntry is the coordinator-local representation of a Redis player hash.
type QueueEntry struct {
	EntryID          string
	UserID           uuid.UUID
	Mode             string
	State            string
	EnqueuedAtMs     int64
	LeaseExpiresAtMs int64
	ClaimID          string
}

// PairClaim is the coordinator-local claim record.
type PairClaim struct {
	ClaimID        string
	FormationKey   string
	Mode           string
	EntryIDA       string
	UserIDA        uuid.UUID
	EnqueuedAtMsA  int64
	EntryIDB       string
	UserIDB        uuid.UUID
	EnqueuedAtMsB  int64
	ClaimedAtMs    int64
	RecoverAfterMs int64
}

// joinScript atomically joins or returns an existing valid entry.
// Searching entries (same mode, unexpired lease) and claimed entries are immutable:
// claimed players must not be rewritten by a concurrent/retried Join — only
// explicit finalize/release may clear claim pointers.
// KEYS[1] = player hash, KEYS[2] = queue sorted set
// ARGV: user_id, mode, entry_id, now_ms, lease_expires_ms, state_searching, state_claimed
var joinScript = redis.NewScript(`
local playerKey = KEYS[1]
local queueKey = KEYS[2]
local userID = ARGV[1]
local mode = ARGV[2]
local newEntryID = ARGV[3]
local nowMs = tonumber(ARGV[4])
local leaseExpiresMs = tonumber(ARGV[5])
local stateSearching = ARGV[6]
local stateClaimed = ARGV[7]

local existing = redis.call('HGETALL', playerKey)
if #existing > 0 then
  local map = {}
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  local lease = tonumber(map['lease_expires_at_ms'] or '0')
  if map['state'] == stateSearching and lease >= nowMs and map['mode'] == mode then
    return {
      map['entry_id'] or '',
      map['user_id'] or userID,
      map['mode'] or mode,
      map['state'] or stateSearching,
      map['enqueued_at_ms'] or tostring(nowMs),
      map['lease_expires_at_ms'] or tostring(leaseExpiresMs),
      map['claim_id'] or '',
      'existing'
    }
  end
  -- Claimed is immutable until finalize/release (no lease-based destroy).
  if map['state'] == stateClaimed then
    return {
      map['entry_id'] or '',
      map['user_id'] or userID,
      map['mode'] or mode,
      map['state'] or stateClaimed,
      map['enqueued_at_ms'] or tostring(nowMs),
      map['lease_expires_at_ms'] or tostring(leaseExpiresMs),
      map['claim_id'] or '',
      'existing_claimed'
    }
  end
  -- Stale searching (expired lease / mode change) self-entry cleanup only.
  if map['entry_id'] and map['mode'] then
    redis.call('ZREM', 'matchmaking:v1:queue:' .. map['mode'], map['entry_id'])
    redis.call('DEL', 'matchmaking:v1:entry:' .. map['entry_id'])
  end
  redis.call('DEL', playerKey)
end

redis.call('HSET', playerKey,
  'entry_id', newEntryID,
  'user_id', userID,
  'mode', mode,
  'state', stateSearching,
  'enqueued_at_ms', tostring(nowMs),
  'lease_expires_at_ms', tostring(leaseExpiresMs)
)
redis.call('ZADD', queueKey, nowMs, newEntryID)
redis.call('SET', 'matchmaking:v1:entry:' .. newEntryID, userID)
return {
  newEntryID,
  userID,
  mode,
  stateSearching,
  tostring(nowMs),
  tostring(leaseExpiresMs),
  '',
  'created'
}
`)

// renewScript renews lease for a valid searching entry without changing priority.
var renewScript = redis.NewScript(`
local playerKey = KEYS[1]
local nowMs = tonumber(ARGV[1])
local leaseExpiresMs = tonumber(ARGV[2])
local stateSearching = ARGV[3]

local existing = redis.call('HGETALL', playerKey)
if #existing == 0 then
  return nil
end
local map = {}
for i = 1, #existing, 2 do
  map[existing[i]] = existing[i+1]
end
local lease = tonumber(map['lease_expires_at_ms'] or '0')
-- Claimed entries are owned by their recovery record, not the searching lease.
-- Never delete their pointers here: durable reconciliation needs them to
-- finalize or restore the original queue priority.
if map['state'] == stateSearching and lease < nowMs then
  if map['entry_id'] and map['mode'] then
    redis.call('ZREM', 'matchmaking:v1:queue:' .. map['mode'], map['entry_id'])
    redis.call('DEL', 'matchmaking:v1:entry:' .. map['entry_id'])
  end
  redis.call('DEL', playerKey)
  return nil
end
if map['state'] == stateSearching then
  redis.call('HSET', playerKey, 'lease_expires_at_ms', tostring(leaseExpiresMs))
  map['lease_expires_at_ms'] = tostring(leaseExpiresMs)
end
return {
  map['entry_id'] or '',
  map['user_id'] or '',
  map['mode'] or '',
  map['state'] or '',
  map['enqueued_at_ms'] or '0',
  map['lease_expires_at_ms'] or '0',
  map['claim_id'] or ''
}
`)

// leaveScript removes the caller's exact searching entry.
var leaveScript = redis.NewScript(`
local playerKey = KEYS[1]
local stateSearching = ARGV[1]
local stateClaimed = ARGV[2]

local existing = redis.call('HGETALL', playerKey)
if #existing == 0 then
  return {'absent'}
end
local map = {}
for i = 1, #existing, 2 do
  map[existing[i]] = existing[i+1]
end
if map['state'] == stateClaimed then
  return {'claimed'}
end
if map['state'] == stateSearching then
  if map['entry_id'] and map['mode'] then
    redis.call('ZREM', 'matchmaking:v1:queue:' .. map['mode'], map['entry_id'])
    redis.call('DEL', 'matchmaking:v1:entry:' .. map['entry_id'])
  end
  redis.call('DEL', playerKey)
  return {'left'}
end
if map['entry_id'] then
  redis.call('DEL', 'matchmaking:v1:entry:' .. map['entry_id'])
end
redis.call('DEL', playerKey)
return {'absent'}
`)

// Join adds the player to the mode queue or returns an existing valid entry.
func (c *MatchmakingCoordinator) Join(ctx context.Context, userID uuid.UUID, mode string, now time.Time, leaseTTL time.Duration) (*QueueEntry, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	entryID := uuid.NewString()
	nowMs := now.UTC().UnixMilli()
	leaseExpiresMs := now.UTC().Add(leaseTTL).UnixMilli()

	result, err := joinScript.Run(ctx, c.client, []string{
		matchmakingPlayerKey(userID),
		matchmakingQueueKey(mode),
	}, userID.String(), mode, entryID, nowMs, leaseExpiresMs, "searching", "claimed").Slice()
	if err != nil {
		return nil, fmt.Errorf("matchmaking join: %w", err)
	}
	entry, err := parseQueueEntryResult(result)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// Leave removes the player's searching entry when not claimed.
func (c *MatchmakingCoordinator) Leave(ctx context.Context, userID uuid.UUID) error {
	if c == nil || c.client == nil {
		return errors.New("matchmaking redis unavailable")
	}
	result, err := leaveScript.Run(ctx, c.client, []string{matchmakingPlayerKey(userID)}, "searching", "claimed").Slice()
	if err != nil {
		return fmt.Errorf("matchmaking leave: %w", err)
	}
	if len(result) > 0 {
		if status, _ := result[0].(string); status == "claimed" {
			return errClaimInProgress
		}
	}
	return nil
}

// ErrClaimInProgress is returned when leave races a pair claim.
var errClaimInProgress = errors.New("matchmaking claim in progress")

// ClaimInProgress reports whether err is a claim-in-progress leave race.
func ClaimInProgress(err error) bool {
	return errors.Is(err, errClaimInProgress)
}

// GetEntry reads the player hash without renewing the lease.
func (c *MatchmakingCoordinator) GetEntry(ctx context.Context, userID uuid.UUID) (*QueueEntry, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	vals, err := c.client.HGetAll(ctx, matchmakingPlayerKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking get entry: %w", err)
	}
	if len(vals) == 0 {
		return nil, nil
	}
	return queueEntryFromMap(vals)
}

// RenewLease extends the lease for a valid entry without changing queue priority.
func (c *MatchmakingCoordinator) RenewLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueEntry, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	nowMs := now.UTC().UnixMilli()
	leaseExpiresMs := now.UTC().Add(leaseTTL).UnixMilli()
	result, err := renewScript.Run(ctx, c.client, []string{matchmakingPlayerKey(userID)}, nowMs, leaseExpiresMs, "searching").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("matchmaking renew lease: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	slice, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("matchmaking renew lease: unexpected result type %T", result)
	}
	return parseQueueEntryResult(slice)
}

// claimPairScript atomically claims the oldest two valid compatible candidates.
// KEYS[1] = queue sorted set, KEYS[2] = claims recovery index
// ARGV: mode, now_ms, recover_after_ms, claim_id, scan_limit, state_searching, state_claimed
// Returns empty table when fewer than two valid candidates, or claim fields on success.
var claimPairScript = redis.NewScript(`
local queueKey = KEYS[1]
local claimsIndexKey = KEYS[2]
local mode = ARGV[1]
local nowMs = tonumber(ARGV[2])
local recoverAfterMs = tonumber(ARGV[3])
local claimID = ARGV[4]
local scanLimit = tonumber(ARGV[5])
local stateSearching = ARGV[6]
local stateClaimed = ARGV[7]

if scanLimit < 2 then
  scanLimit = 2
end

local candidates = redis.call('ZRANGE', queueKey, 0, scanLimit - 1, 'WITHSCORES')
local valid = {}

local i = 1
while i <= #candidates do
  local entryID = candidates[i]
  local score = tonumber(candidates[i+1])
  i = i + 2

  -- entry_id is the sorted-set member; player hash is keyed by user_id via entry map.
  local entryMapKey = 'matchmaking:v1:entry:' .. entryID
  local userID = redis.call('GET', entryMapKey)
  if not userID then
    redis.call('ZREM', queueKey, entryID)
  else
    local playerKey = 'matchmaking:v1:player:' .. userID
    local map = {}
    local existing = redis.call('HGETALL', playerKey)
    if #existing == 0 then
      redis.call('ZREM', queueKey, entryID)
      redis.call('DEL', entryMapKey)
    else
      for j = 1, #existing, 2 do
        map[existing[j]] = existing[j+1]
      end
      local lease = tonumber(map['lease_expires_at_ms'] or '0')
      if map['entry_id'] ~= entryID or map['mode'] ~= mode or map['state'] ~= stateSearching or lease < nowMs then
        redis.call('ZREM', queueKey, entryID)
        if map['entry_id'] == entryID then
          redis.call('DEL', playerKey)
          redis.call('DEL', entryMapKey)
        end
      else
        table.insert(valid, {
          entry_id = entryID,
          user_id = userID,
          enqueued_at_ms = tonumber(map['enqueued_at_ms'] or tostring(score)),
          player_key = playerKey,
          entry_map_key = entryMapKey
        })
        if #valid >= 2 then
          break
        end
      end
    end
  end
end

if #valid < 2 then
  return {}
end

local a = valid[1]
local b = valid[2]
if a.user_id == b.user_id then
  return {}
end

local claimKey = 'matchmaking:v1:claim:' .. claimID
redis.call('HSET', claimKey,
  'claim_id', claimID,
  'formation_key', claimID,
  'mode', mode,
  'entry_id_a', a.entry_id,
  'user_id_a', a.user_id,
  'enqueued_at_ms_a', tostring(a.enqueued_at_ms),
  'entry_id_b', b.entry_id,
  'user_id_b', b.user_id,
  'enqueued_at_ms_b', tostring(b.enqueued_at_ms),
  'claimed_at_ms', tostring(nowMs),
  'recover_after_ms', tostring(recoverAfterMs)
)
-- Claim records are deleted only by finalize/release. Expiring this record can
-- strand claimed player hashes with no payload available for recovery.
redis.call('ZADD', claimsIndexKey, recoverAfterMs, claimID)

redis.call('HSET', a.player_key,
  'state', stateClaimed,
  'claim_id', claimID,
  'lease_expires_at_ms', tostring(recoverAfterMs)
)
redis.call('HSET', b.player_key,
  'state', stateClaimed,
  'claim_id', claimID,
  'lease_expires_at_ms', tostring(recoverAfterMs)
)
redis.call('ZREM', queueKey, a.entry_id)
redis.call('ZREM', queueKey, b.entry_id)

return {
  claimID,
  claimID,
  mode,
  a.entry_id,
  a.user_id,
  tostring(a.enqueued_at_ms),
  b.entry_id,
  b.user_id,
  tostring(b.enqueued_at_ms),
  tostring(nowMs),
  tostring(recoverAfterMs)
}
`)

// finalizeClaimScript removes claimed player hashes and claim record after durable commit.
// KEYS[1]=claim key, KEYS[2]=claims index
// ARGV: claim_id, entry_id_a, user_id_a, entry_id_b, user_id_b, state_claimed
var finalizeClaimScript = redis.NewScript(`
local claimKey = KEYS[1]
local claimsIndexKey = KEYS[2]
local claimID = ARGV[1]
local entryIDA = ARGV[2]
local userIDA = ARGV[3]
local entryIDB = ARGV[4]
local userIDB = ARGV[5]
local stateClaimed = ARGV[6]

local function clearPlayer(userID, entryID)
  local playerKey = 'matchmaking:v1:player:' .. userID
  local map = {}
  local existing = redis.call('HGETALL', playerKey)
  if #existing == 0 then
    redis.call('DEL', 'matchmaking:v1:entry:' .. entryID)
    return
  end
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  if map['claim_id'] == claimID and map['entry_id'] == entryID and map['state'] == stateClaimed then
    redis.call('DEL', playerKey)
  end
  redis.call('DEL', 'matchmaking:v1:entry:' .. entryID)
end

clearPlayer(userIDA, entryIDA)
clearPlayer(userIDB, entryIDB)
redis.call('DEL', claimKey)
redis.call('ZREM', claimsIndexKey, claimID)
return {'ok'}
`)

// releaseClaimScript releases a claim and independently requeues each player when eligible.
// KEYS[1]=claim key, KEYS[2]=claims index, KEYS[3]=queue key
// ARGV: claim_id, requeue_a (0/1), requeue_b (0/1), now_ms, lease_expires_ms, state_searching, state_claimed
var releaseClaimScript = redis.NewScript(`
local claimKey = KEYS[1]
local claimsIndexKey = KEYS[2]
local queueKey = KEYS[3]
local claimID = ARGV[1]
local requeueA = ARGV[2] == '1'
local requeueB = ARGV[3] == '1'
local nowMs = tonumber(ARGV[4])
local leaseExpiresMs = tonumber(ARGV[5])
local stateSearching = ARGV[6]
local stateClaimed = ARGV[7]

local existing = redis.call('HGETALL', claimKey)
if #existing == 0 then
  redis.call('ZREM', claimsIndexKey, claimID)
  return {'absent'}
end
local claim = {}
for i = 1, #existing, 2 do
  claim[existing[i]] = existing[i+1]
end
if claim['claim_id'] ~= claimID then
  return {'mismatch'}
end

local function releasePlayer(userID, entryID, enqueuedAt, requeue)
  local playerKey = 'matchmaking:v1:player:' .. userID
  local map = {}
  local vals = redis.call('HGETALL', playerKey)
  if #vals == 0 then
    redis.call('DEL', 'matchmaking:v1:entry:' .. entryID)
    return
  end
  for i = 1, #vals, 2 do
    map[vals[i]] = vals[i+1]
  end
  if map['claim_id'] ~= claimID or map['entry_id'] ~= entryID then
    return
  end
  if requeue then
    redis.call('HSET', playerKey,
      'state', stateSearching,
      'claim_id', '',
      'lease_expires_at_ms', tostring(leaseExpiresMs),
      'enqueued_at_ms', tostring(enqueuedAt)
    )
    redis.call('ZADD', queueKey, tonumber(enqueuedAt), entryID)
    redis.call('SET', 'matchmaking:v1:entry:' .. entryID, userID)
  else
    redis.call('DEL', playerKey)
    redis.call('DEL', 'matchmaking:v1:entry:' .. entryID)
  end
end

releasePlayer(claim['user_id_a'], claim['entry_id_a'], tonumber(claim['enqueued_at_ms_a'] or '0'), requeueA)
releasePlayer(claim['user_id_b'], claim['entry_id_b'], tonumber(claim['enqueued_at_ms_b'] or '0'), requeueB)
redis.call('DEL', claimKey)
redis.call('ZREM', claimsIndexKey, claimID)
return {'released'}
`)

// ClaimPair scans up to scanLimit oldest queue members and claims exactly two valid players.
func (c *MatchmakingCoordinator) ClaimPair(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int) (*PairClaim, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	if scanLimit < 2 {
		scanLimit = 2
	}
	nowMs := now.UTC().UnixMilli()
	recoverAfterMs := now.UTC().Add(claimTTL).UnixMilli()
	claimID := uuid.NewString()

	result, err := claimPairScript.Run(ctx, c.client, []string{
		matchmakingQueueKey(mode),
		matchmakingClaimsIndexKey(),
	}, mode, nowMs, recoverAfterMs, claimID, scanLimit, "searching", "claimed").Slice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("matchmaking claim pair: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return parsePairClaimResult(result)
}

// FinalizeClaim removes exact claimed player state after durable match commit.
func (c *MatchmakingCoordinator) FinalizeClaim(ctx context.Context, claim *PairClaim) error {
	if c == nil || c.client == nil {
		return errors.New("matchmaking redis unavailable")
	}
	if claim == nil {
		return nil
	}
	_, err := finalizeClaimScript.Run(ctx, c.client, []string{
		matchmakingClaimKey(claim.ClaimID),
		matchmakingClaimsIndexKey(),
	}, claim.ClaimID, claim.EntryIDA, claim.UserIDA.String(), claim.EntryIDB, claim.UserIDB.String(), "claimed").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("matchmaking finalize claim: %w", err)
	}
	return nil
}

// GetClaim loads a claim record by id.
func (c *MatchmakingCoordinator) GetClaim(ctx context.Context, claimID string) (*PairClaim, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	if claimID == "" {
		return nil, nil
	}
	vals, err := c.client.HGetAll(ctx, matchmakingClaimKey(claimID)).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking get claim: %w", err)
	}
	if len(vals) == 0 {
		return nil, nil
	}
	return pairClaimFromMap(vals)
}

// ReleaseClaim releases a claim and independently requeues each player when eligible.
// requeueA/requeueB control player A and B respectively so an ineligible candidate
// does not discard an eligible opponent.
func (c *MatchmakingCoordinator) ReleaseClaim(ctx context.Context, claim *PairClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	if c == nil || c.client == nil {
		return errors.New("matchmaking redis unavailable")
	}
	if claim == nil {
		return nil
	}
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	flagA, flagB := "0", "0"
	if requeueA {
		flagA = "1"
	}
	if requeueB {
		flagB = "1"
	}
	now := time.Now().UTC()
	nowMs := now.UnixMilli()
	leaseExpiresMs := now.Add(leaseTTL).UnixMilli()
	_, err := releaseClaimScript.Run(ctx, c.client, []string{
		matchmakingClaimKey(claim.ClaimID),
		matchmakingClaimsIndexKey(),
		matchmakingQueueKey(claim.Mode),
	}, claim.ClaimID, flagA, flagB, nowMs, leaseExpiresMs, "searching", "claimed").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("matchmaking release claim: %w", err)
	}
	return nil
}

// ListExpiredClaims returns up to limit claim IDs whose recover_after is in the past.
// Callers must check PostgreSQL formation precedence before requeueing.
func (c *MatchmakingCoordinator) ListExpiredClaims(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("matchmaking redis unavailable")
	}
	if limit <= 0 {
		limit = 20
	}
	nowMs := now.UTC().UnixMilli()
	ids, err := c.client.ZRangeByScore(ctx, matchmakingClaimsIndexKey(), &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(nowMs, 10),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking list expired claims: %w", err)
	}
	return ids, nil
}

// DropClaimIndex removes a stale claim id from the recovery index when the claim hash is gone.
func (c *MatchmakingCoordinator) DropClaimIndex(ctx context.Context, claimID string) error {
	if c == nil || c.client == nil {
		return errors.New("matchmaking redis unavailable")
	}
	if claimID == "" {
		return nil
	}
	if err := c.client.ZRem(ctx, matchmakingClaimsIndexKey(), claimID).Err(); err != nil {
		return fmt.Errorf("matchmaking drop claim index: %w", err)
	}
	return nil
}

func pairClaimFromMap(vals map[string]string) (*PairClaim, error) {
	userA, err := uuid.Parse(vals["user_id_a"])
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim user a: %w", err)
	}
	userB, err := uuid.Parse(vals["user_id_b"])
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim user b: %w", err)
	}
	enqueuedA, _ := strconv.ParseInt(vals["enqueued_at_ms_a"], 10, 64)
	enqueuedB, _ := strconv.ParseInt(vals["enqueued_at_ms_b"], 10, 64)
	claimedAt, _ := strconv.ParseInt(vals["claimed_at_ms"], 10, 64)
	recoverAfter, _ := strconv.ParseInt(vals["recover_after_ms"], 10, 64)
	return &PairClaim{
		ClaimID:        vals["claim_id"],
		FormationKey:   vals["formation_key"],
		Mode:           vals["mode"],
		EntryIDA:       vals["entry_id_a"],
		UserIDA:        userA,
		EnqueuedAtMsA:  enqueuedA,
		EntryIDB:       vals["entry_id_b"],
		UserIDB:        userB,
		EnqueuedAtMsB:  enqueuedB,
		ClaimedAtMs:    claimedAt,
		RecoverAfterMs: recoverAfter,
	}, nil
}

func parsePairClaimResult(result []interface{}) (*PairClaim, error) {
	if len(result) < 11 {
		return nil, fmt.Errorf("matchmaking claim result: expected >=11 fields, got %d", len(result))
	}
	userA, err := uuid.Parse(asString(result[4]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim user a: %w", err)
	}
	userB, err := uuid.Parse(asString(result[7]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim user b: %w", err)
	}
	enqueuedA, err := strconv.ParseInt(asString(result[5]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim enqueued a: %w", err)
	}
	enqueuedB, err := strconv.ParseInt(asString(result[8]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim enqueued b: %w", err)
	}
	claimedAt, err := strconv.ParseInt(asString(result[9]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim claimed_at: %w", err)
	}
	recoverAfter, err := strconv.ParseInt(asString(result[10]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking claim recover_after: %w", err)
	}
	return &PairClaim{
		ClaimID:        asString(result[0]),
		FormationKey:   asString(result[1]),
		Mode:           asString(result[2]),
		EntryIDA:       asString(result[3]),
		UserIDA:        userA,
		EnqueuedAtMsA:  enqueuedA,
		EntryIDB:       asString(result[6]),
		UserIDB:        userB,
		EnqueuedAtMsB:  enqueuedB,
		ClaimedAtMs:    claimedAt,
		RecoverAfterMs: recoverAfter,
	}, nil
}

func parseQueueEntryResult(result []interface{}) (*QueueEntry, error) {
	if len(result) < 7 {
		return nil, fmt.Errorf("matchmaking entry result: expected >=7 fields, got %d", len(result))
	}
	userID, err := uuid.Parse(asString(result[1]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking entry user id: %w", err)
	}
	enqueuedAt, err := strconv.ParseInt(asString(result[4]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking entry enqueued_at: %w", err)
	}
	leaseExpires, err := strconv.ParseInt(asString(result[5]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking entry lease: %w", err)
	}
	return &QueueEntry{
		EntryID:          asString(result[0]),
		UserID:           userID,
		Mode:             asString(result[2]),
		State:            asString(result[3]),
		EnqueuedAtMs:     enqueuedAt,
		LeaseExpiresAtMs: leaseExpires,
		ClaimID:          asString(result[6]),
	}, nil
}

func queueEntryFromMap(vals map[string]string) (*QueueEntry, error) {
	userID, err := uuid.Parse(vals[fieldUserID])
	if err != nil {
		return nil, fmt.Errorf("matchmaking player user id: %w", err)
	}
	enqueuedAt, _ := strconv.ParseInt(vals[fieldEnqueuedAtMs], 10, 64)
	leaseExpires, _ := strconv.ParseInt(vals[fieldLeaseExpiresAt], 10, 64)
	return &QueueEntry{
		EntryID:          vals[fieldEntryID],
		UserID:           userID,
		Mode:             vals[fieldMode],
		State:            vals[fieldState],
		EnqueuedAtMs:     enqueuedAt,
		LeaseExpiresAtMs: leaseExpires,
		ClaimID:          vals[fieldClaimID],
	}, nil
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
