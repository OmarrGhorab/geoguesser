package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const matchmakingV2KeyVersion = "v2"

// MatchmakingV2Coordinator coordinates ephemeral team-ticket queue state in Redis.
// It is additive to MatchmakingCoordinator (v1 pair queues) for staged rollout.
type MatchmakingV2Coordinator struct {
	client *redis.Client
}

// NewMatchmakingV2Coordinator constructs a Redis v2 team-ticket coordinator.
func NewMatchmakingV2Coordinator(client *redis.Client) *MatchmakingV2Coordinator {
	return &MatchmakingV2Coordinator{client: client}
}

// Ticket field names stored in the ticket hash.
const (
	v2FieldTicketID       = "ticket_id"
	v2FieldPartyID        = "party_id"
	v2FieldMode           = "mode"
	v2FieldUserIDs        = "user_ids"
	v2FieldTeamSize       = "team_size"
	v2FieldTeamAvgRating  = "team_avg_rating"
	v2FieldEnqueuedAtMs   = "enqueued_at_ms"
	v2FieldLeaseExpiresAt = "lease_expires_at_ms"
	v2FieldState          = "state"
	v2FieldClaimID        = "claim_id"
	v2FieldPartyVersion   = "party_version"
)

// Ticket states.
const (
	V2StateSearching = "searching"
	V2StateClaimed   = "claimed"
)

// Sentinel errors for v2 ticket operations.
var (
	errV2Unavailable          = errors.New("matchmaking v2 redis unavailable")
	errV2UserConflict         = errors.New("matchmaking v2 user already queued on another ticket")
	errV2PartyVersionMismatch = errors.New("matchmaking v2 party version mismatch")
	errV2ClaimInProgress      = errors.New("matchmaking v2 claim in progress")
	errV2InvalidRoster        = errors.New("matchmaking v2 invalid roster")
)

// V2UserConflict reports whether err is a per-user exclusivity conflict.
func V2UserConflict(err error) bool { return errors.Is(err, errV2UserConflict) }

// V2PartyVersionMismatch reports whether err is a party-version mismatch on rejoin.
func V2PartyVersionMismatch(err error) bool { return errors.Is(err, errV2PartyVersionMismatch) }

// V2ClaimInProgress reports whether err is a leave/claim race on a v2 ticket.
func V2ClaimInProgress(err error) bool {
	return errors.Is(err, errV2ClaimInProgress) || errors.Is(err, errClaimInProgress)
}

// Versioned key builders for v2.
func matchmakingV2QueueKey(mode string) string {
	return fmt.Sprintf("matchmaking:%s:queue:%s", matchmakingV2KeyVersion, mode)
}

func matchmakingV2TicketKey(ticketID string) string {
	return fmt.Sprintf("matchmaking:%s:ticket:%s", matchmakingV2KeyVersion, ticketID)
}

func matchmakingV2UserKey(userID uuid.UUID) string {
	return fmt.Sprintf("matchmaking:%s:user:%s", matchmakingV2KeyVersion, userID.String())
}

func matchmakingV2ClaimKey(claimID string) string {
	return fmt.Sprintf("matchmaking:%s:claim:%s", matchmakingV2KeyVersion, claimID)
}

func matchmakingV2ClaimsIndexKey() string {
	return fmt.Sprintf("matchmaking:%s:claims", matchmakingV2KeyVersion)
}

// QueueTicket is the coordinator-local representation of a v2 ticket hash.
type QueueTicket struct {
	TicketID         string
	PartyID          uuid.UUID
	Mode             string
	UserIDs          []uuid.UUID
	TeamSize         int
	TeamAvgRating    int
	EnqueuedAtMs     int64
	LeaseExpiresAtMs int64
	State            string
	ClaimID          string
	PartyVersion     int64
}

// TeamClaim is the coordinator-local representation of a two-ticket claim.
type TeamClaim struct {
	ClaimID        string
	FormationKey   string
	Mode           string
	TicketIDA      string
	TicketIDB      string
	UserIDsA       []uuid.UUID
	UserIDsB       []uuid.UUID
	PartyIDA       uuid.UUID
	PartyIDB       uuid.UUID
	PartyVersionA  int64
	PartyVersionB  int64
	TeamSize       int
	TeamAvgRatingA int
	TeamAvgRatingB int
	EnqueuedAtMsA  int64
	EnqueuedAtMsB  int64
	ClaimedAtMs    int64
	RecoverAfterMs int64
}

// JoinTicketInput is the immutable roster snapshot used to enqueue a team ticket.
type JoinTicketInput struct {
	Mode          string
	UserIDs       []uuid.UUID // ordered roster; length must equal TeamSize
	PartyID       uuid.UUID   // uuid.Nil for solo / implicit party
	PartyVersion  int64
	TeamSize      int
	TeamAvgRating int // 0 for casual; hidden/visible rating average for ranked
}

// joinTicketScript atomically creates a ticket when all user pointers are free,
// or returns the existing ticket when every user already points at the same valid ticket.
//
// ARGV:
//
//	1 ticket_id
//	2 mode
//	3 party_id (empty string if none)
//	4 party_version
//	5 team_size
//	6 team_avg_rating
//	7 user_ids_csv
//	8 now_ms
//	9 lease_expires_ms
//
// 10 n
// 11.. user_id strings
//
// Returns: status, then ticket fields on success.
// status: created | existing | existing_claimed | err_user_conflict | err_party_version | err_invalid
var joinTicketScript = redis.NewScript(`
local newTicketID = ARGV[1]
local mode = ARGV[2]
local partyID = ARGV[3]
local partyVersion = ARGV[4]
local teamSize = ARGV[5]
local teamAvgRating = ARGV[6]
local userIDsCSV = ARGV[7]
local nowMs = tonumber(ARGV[8])
local leaseExpiresMs = tonumber(ARGV[9])
local n = tonumber(ARGV[10])
local stateSearching = 'searching'
local stateClaimed = 'claimed'

if n < 1 then
  return {'err_invalid'}
end

local userIDs = {}
for i = 1, n do
  userIDs[i] = ARGV[10 + i]
end

local function ticketKey(id)
  return 'matchmaking:v2:ticket:' .. id
end
local function userKey(uid)
  return 'matchmaking:v2:user:' .. uid
end
local function queueKeyFor(m)
  return 'matchmaking:v2:queue:' .. m
end

local function hmap(key)
  local existing = redis.call('HGETALL', key)
  local map = {}
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  return map
end

local function ticketResult(map, status)
  return {
    status,
    map['ticket_id'] or '',
    map['party_id'] or '',
    map['mode'] or '',
    map['user_ids'] or '',
    map['team_size'] or '0',
    map['team_avg_rating'] or '0',
    map['enqueued_at_ms'] or '0',
    map['lease_expires_at_ms'] or '0',
    map['state'] or '',
    map['claim_id'] or '',
    map['party_version'] or '0'
  }
end

local function clearTicket(tid, map)
  if map['user_ids'] and map['user_ids'] ~= '' then
    for uid in string.gmatch(map['user_ids'], '[^,]+') do
      local ptr = redis.call('GET', userKey(uid))
      if ptr == tid then
        redis.call('DEL', userKey(uid))
      end
    end
  end
  if map['mode'] and map['mode'] ~= '' then
    redis.call('ZREM', queueKeyFor(map['mode']), tid)
  end
  redis.call('DEL', ticketKey(tid))
end

-- Collect unique pointers across roster.
local existingTicketID = nil
local missing = 0
for i = 1, n do
  local ptr = redis.call('GET', userKey(userIDs[i]))
  if not ptr then
    missing = missing + 1
  else
    if existingTicketID == nil then
      existingTicketID = ptr
    elseif existingTicketID ~= ptr then
      return {'err_user_conflict'}
    end
  end
end

if missing > 0 and missing < n then
  -- Partial occupancy: some members free, some bound — exclusivity conflict.
  return {'err_user_conflict'}
end

if existingTicketID ~= nil then
  local map = hmap(ticketKey(existingTicketID))
  if map['ticket_id'] == nil or map['ticket_id'] == '' then
    -- Dangling user pointers; clear them and create fresh.
    for i = 1, n do
      redis.call('DEL', userKey(userIDs[i]))
    end
  else
    if map['state'] == stateClaimed then
      return ticketResult(map, 'existing_claimed')
    end
    local lease = tonumber(map['lease_expires_at_ms'] or '0')
    if map['state'] == stateSearching and lease >= nowMs and map['mode'] == mode then
      if map['user_ids'] ~= userIDsCSV then
        return {'err_user_conflict'}
      end
      if (map['party_version'] or '0') ~= partyVersion then
        return {'err_party_version'}
      end
      return ticketResult(map, 'existing')
    end
    -- Stale searching (expired lease or mode drift): cleanup and recreate.
    clearTicket(existingTicketID, map)
  end
end

-- Ensure no user still holds a foreign pointer (race with another join).
for i = 1, n do
  local ptr = redis.call('GET', userKey(userIDs[i]))
  if ptr then
    return {'err_user_conflict'}
  end
end

local tKey = ticketKey(newTicketID)
local qKey = queueKeyFor(mode)
redis.call('HSET', tKey,
  'ticket_id', newTicketID,
  'party_id', partyID,
  'mode', mode,
  'user_ids', userIDsCSV,
  'team_size', tostring(teamSize),
  'team_avg_rating', tostring(teamAvgRating),
  'enqueued_at_ms', tostring(nowMs),
  'lease_expires_at_ms', tostring(leaseExpiresMs),
  'state', stateSearching,
  'claim_id', '',
  'party_version', tostring(partyVersion)
)
redis.call('ZADD', qKey, nowMs, newTicketID)
for i = 1, n do
  redis.call('SET', userKey(userIDs[i]), newTicketID)
end

return {
  'created',
  newTicketID,
  partyID,
  mode,
  userIDsCSV,
  tostring(teamSize),
  tostring(teamAvgRating),
  tostring(nowMs),
  tostring(leaseExpiresMs),
  stateSearching,
  '',
  tostring(partyVersion)
}
`)

// renewTicketScript renews lease for a searching ticket without changing queue priority.
// KEYS unused; ARGV: user_id, now_ms, lease_expires_ms
var renewTicketScript = redis.NewScript(`
local userID = ARGV[1]
local nowMs = tonumber(ARGV[2])
local leaseExpiresMs = tonumber(ARGV[3])
local stateSearching = 'searching'
local stateClaimed = 'claimed'

local function userKey(uid) return 'matchmaking:v2:user:' .. uid end
local function ticketKey(id) return 'matchmaking:v2:ticket:' .. id end
local function queueKeyFor(m) return 'matchmaking:v2:queue:' .. m end

local function hmap(key)
  local existing = redis.call('HGETALL', key)
  local map = {}
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  return map
end

local function ticketResult(map)
  return {
    map['ticket_id'] or '',
    map['party_id'] or '',
    map['mode'] or '',
    map['user_ids'] or '',
    map['team_size'] or '0',
    map['team_avg_rating'] or '0',
    map['enqueued_at_ms'] or '0',
    map['lease_expires_at_ms'] or '0',
    map['state'] or '',
    map['claim_id'] or '',
    map['party_version'] or '0'
  }
end

local function clearTicket(tid, map)
  if map['user_ids'] and map['user_ids'] ~= '' then
    for uid in string.gmatch(map['user_ids'], '[^,]+') do
      if redis.call('GET', userKey(uid)) == tid then
        redis.call('DEL', userKey(uid))
      end
    end
  end
  if map['mode'] and map['mode'] ~= '' then
    redis.call('ZREM', queueKeyFor(map['mode']), tid)
  end
  redis.call('DEL', ticketKey(tid))
end

local tid = redis.call('GET', userKey(userID))
if not tid then
  return nil
end
local map = hmap(ticketKey(tid))
if map['ticket_id'] == nil or map['ticket_id'] == '' then
  redis.call('DEL', userKey(userID))
  return nil
end

-- Claimed tickets are owned by recovery records; never destroy on search-lease expiry.
if map['state'] == stateClaimed then
  return ticketResult(map)
end

local lease = tonumber(map['lease_expires_at_ms'] or '0')
if map['state'] == stateSearching and lease < nowMs then
  clearTicket(tid, map)
  return nil
end

if map['state'] == stateSearching then
  redis.call('HSET', ticketKey(tid), 'lease_expires_at_ms', tostring(leaseExpiresMs))
  map['lease_expires_at_ms'] = tostring(leaseExpiresMs)
end
return ticketResult(map)
`)

// leaveTicketScript removes a searching ticket for the caller's roster.
// ARGV: user_id
// Returns: left | absent | claimed
var leaveTicketScript = redis.NewScript(`
local userID = ARGV[1]
local stateSearching = 'searching'
local stateClaimed = 'claimed'

local function userKey(uid) return 'matchmaking:v2:user:' .. uid end
local function ticketKey(id) return 'matchmaking:v2:ticket:' .. id end
local function queueKeyFor(m) return 'matchmaking:v2:queue:' .. m end

local tid = redis.call('GET', userKey(userID))
if not tid then
  return {'absent'}
end

local existing = redis.call('HGETALL', ticketKey(tid))
if #existing == 0 then
  redis.call('DEL', userKey(userID))
  return {'absent'}
end
local map = {}
for i = 1, #existing, 2 do
  map[existing[i]] = existing[i+1]
end

if map['state'] == stateClaimed then
  return {'claimed'}
end

if map['user_ids'] and map['user_ids'] ~= '' then
  for uid in string.gmatch(map['user_ids'], '[^,]+') do
    if redis.call('GET', userKey(uid)) == tid then
      redis.call('DEL', userKey(uid))
    end
  end
end
if map['mode'] and map['mode'] ~= '' then
  redis.call('ZREM', queueKeyFor(map['mode']), tid)
end
redis.call('DEL', ticketKey(tid))
return {'left'}
`)

// claimTicketsScript scans up to scan_limit oldest tickets and claims two equal-roster,
// disjoint-user tickets atomically.
//
// KEYS[1] = queue sorted set, KEYS[2] = claims recovery index
// ARGV: mode, now_ms, recover_after_ms, claim_id, scan_limit, max_rating_delta
//
//	max_rating_delta <= 0 means no rating filter (Casual / unlimited).
//
// Returns empty table when no pair, or claim field list on success.
var claimTicketsScript = redis.NewScript(`
local queueKey = KEYS[1]
local claimsIndexKey = KEYS[2]
local mode = ARGV[1]
local nowMs = tonumber(ARGV[2])
local recoverAfterMs = tonumber(ARGV[3])
local claimID = ARGV[4]
local scanLimit = tonumber(ARGV[5])
local maxRatingDelta = tonumber(ARGV[6])
local stateSearching = 'searching'
local stateClaimed = 'claimed'

if scanLimit < 2 then
  scanLimit = 2
end

local function ticketKey(id) return 'matchmaking:v2:ticket:' .. id end
local function userKey(uid) return 'matchmaking:v2:user:' .. uid end
local function claimKey(id) return 'matchmaking:v2:claim:' .. id end

local function hmap(key)
  local existing = redis.call('HGETALL', key)
  local map = {}
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  return map
end

local function clearTicket(tid, map)
  if map['user_ids'] and map['user_ids'] ~= '' then
    for uid in string.gmatch(map['user_ids'], '[^,]+') do
      if redis.call('GET', userKey(uid)) == tid then
        redis.call('DEL', userKey(uid))
      end
    end
  end
  redis.call('ZREM', queueKey, tid)
  redis.call('DEL', ticketKey(tid))
end

local function parseUsers(csv)
  local users = {}
  if not csv or csv == '' then
    return users
  end
  for uid in string.gmatch(csv, '[^,]+') do
    table.insert(users, uid)
  end
  return users
end

local function disjoint(a, b)
  local set = {}
  for i = 1, #a do set[a[i]] = true end
  for i = 1, #b do
    if set[b[i]] then return false end
  end
  return true
end

local candidates = redis.call('ZRANGE', queueKey, 0, scanLimit - 1, 'WITHSCORES')
local valid = {}

local i = 1
while i <= #candidates do
  local tid = candidates[i]
  local score = tonumber(candidates[i+1])
  i = i + 2

  local map = hmap(ticketKey(tid))
  if map['ticket_id'] == nil or map['ticket_id'] == '' then
    redis.call('ZREM', queueKey, tid)
  else
    local lease = tonumber(map['lease_expires_at_ms'] or '0')
    if map['ticket_id'] ~= tid or map['mode'] ~= mode or map['state'] ~= stateSearching or lease < nowMs then
      if map['ticket_id'] == tid and map['state'] == stateSearching then
        clearTicket(tid, map)
      else
        redis.call('ZREM', queueKey, tid)
      end
    else
      local users = parseUsers(map['user_ids'])
      local teamSize = tonumber(map['team_size'] or '0')
      if #users ~= teamSize or teamSize < 1 then
        clearTicket(tid, map)
      else
        -- Verify every user pointer still references this ticket.
        local pointersOK = true
        for u = 1, #users do
          if redis.call('GET', userKey(users[u])) ~= tid then
            pointersOK = false
            break
          end
        end
        if not pointersOK then
          clearTicket(tid, map)
        else
          table.insert(valid, {
            ticket_id = tid,
            party_id = map['party_id'] or '',
            user_ids = map['user_ids'],
            users = users,
            team_size = teamSize,
            team_avg_rating = tonumber(map['team_avg_rating'] or '0'),
            enqueued_at_ms = tonumber(map['enqueued_at_ms'] or tostring(score)),
            party_version = map['party_version'] or '0'
          })
        end
      end
    end
  end
end

if #valid < 2 then
  return {}
end

local a, b = nil, nil
for x = 1, #valid - 1 do
  for y = x + 1, #valid do
    local left = valid[x]
    local right = valid[y]
    if left.team_size == right.team_size and disjoint(left.users, right.users) then
      if maxRatingDelta <= 0 then
        a, b = left, right
        break
      end
      -- Expanding ranked windows: start ±100, +50/30s, cap at maxRatingDelta (typically 400).
      local function halfWidth(enqueuedAtMs)
        local waited = nowMs - (enqueuedAtMs or nowMs)
        if waited < 0 then waited = 0 end
        local steps = math.floor(waited / 30000)
        local half = 100 + 50 * steps
        if half > maxRatingDelta then half = maxRatingDelta end
        if half < 100 then half = 100 end
        return half
      end
      local delta = left.team_avg_rating - right.team_avg_rating
      if delta < 0 then delta = -delta end
      local allowed = halfWidth(left.enqueued_at_ms)
      local other = halfWidth(right.enqueued_at_ms)
      if other > allowed then allowed = other end
      if delta <= allowed then
        a, b = left, right
        break
      end
    end
  end
  if a ~= nil then break end
end

if a == nil or b == nil then
  return {}
end

local cKey = claimKey(claimID)
redis.call('HSET', cKey,
  'claim_id', claimID,
  'formation_key', claimID,
  'mode', mode,
  'ticket_id_a', a.ticket_id,
  'ticket_id_b', b.ticket_id,
  'user_ids_a', a.user_ids,
  'user_ids_b', b.user_ids,
  'party_id_a', a.party_id,
  'party_id_b', b.party_id,
  'party_version_a', a.party_version,
  'party_version_b', b.party_version,
  'team_size', tostring(a.team_size),
  'team_avg_rating_a', tostring(a.team_avg_rating),
  'team_avg_rating_b', tostring(b.team_avg_rating),
  'enqueued_at_ms_a', tostring(a.enqueued_at_ms),
  'enqueued_at_ms_b', tostring(b.enqueued_at_ms),
  'claimed_at_ms', tostring(nowMs),
  'recover_after_ms', tostring(recoverAfterMs)
)
-- Claim records are durable until finalize/release (no Redis TTL).
redis.call('ZADD', claimsIndexKey, recoverAfterMs, claimID)

redis.call('HSET', ticketKey(a.ticket_id),
  'state', stateClaimed,
  'claim_id', claimID,
  'lease_expires_at_ms', tostring(recoverAfterMs)
)
redis.call('HSET', ticketKey(b.ticket_id),
  'state', stateClaimed,
  'claim_id', claimID,
  'lease_expires_at_ms', tostring(recoverAfterMs)
)
redis.call('ZREM', queueKey, a.ticket_id)
redis.call('ZREM', queueKey, b.ticket_id)

return {
  claimID,
  claimID,
  mode,
  a.ticket_id,
  b.ticket_id,
  a.user_ids,
  b.user_ids,
  a.party_id,
  b.party_id,
  a.party_version,
  b.party_version,
  tostring(a.team_size),
  tostring(a.team_avg_rating),
  tostring(b.team_avg_rating),
  tostring(a.enqueued_at_ms),
  tostring(b.enqueued_at_ms),
  tostring(nowMs),
  tostring(recoverAfterMs)
}
`)

// finalizeTeamClaimScript clears claimed tickets after durable match commit.
// ARGV: claim_id, ticket_id_a, ticket_id_b, user_ids_a_csv, user_ids_b_csv
var finalizeTeamClaimScript = redis.NewScript(`
local claimID = ARGV[1]
local ticketIDA = ARGV[2]
local ticketIDB = ARGV[3]
local userIDsA = ARGV[4]
local userIDsB = ARGV[5]
local stateClaimed = 'claimed'

local function userKey(uid) return 'matchmaking:v2:user:' .. uid end
local function ticketKey(id) return 'matchmaking:v2:ticket:' .. id end
local function claimKey(id) return 'matchmaking:v2:claim:' .. id end

local function clearClaimedTicket(tid, csv)
  local map = {}
  local existing = redis.call('HGETALL', ticketKey(tid))
  for i = 1, #existing, 2 do
    map[existing[i]] = existing[i+1]
  end
  if map['claim_id'] == claimID and map['ticket_id'] == tid and map['state'] == stateClaimed then
    redis.call('DEL', ticketKey(tid))
  end
  if csv and csv ~= '' then
    for uid in string.gmatch(csv, '[^,]+') do
      if redis.call('GET', userKey(uid)) == tid then
        redis.call('DEL', userKey(uid))
      end
    end
  end
end

clearClaimedTicket(ticketIDA, userIDsA)
clearClaimedTicket(ticketIDB, userIDsB)
redis.call('DEL', claimKey(claimID))
redis.call('ZREM', 'matchmaking:v2:claims', claimID)
return {'ok'}
`)

// releaseTeamClaimScript releases a claim and independently requeues each ticket.
// ARGV: claim_id, requeue_a (0/1), requeue_b (0/1), now_ms, lease_expires_ms
var releaseTeamClaimScript = redis.NewScript(`
local claimID = ARGV[1]
local requeueA = ARGV[2] == '1'
local requeueB = ARGV[3] == '1'
local nowMs = tonumber(ARGV[4])
local leaseExpiresMs = tonumber(ARGV[5])
local stateSearching = 'searching'
local stateClaimed = 'claimed'

local function userKey(uid) return 'matchmaking:v2:user:' .. uid end
local function ticketKey(id) return 'matchmaking:v2:ticket:' .. id end
local function claimKey(id) return 'matchmaking:v2:claim:' .. id end
local function queueKeyFor(m) return 'matchmaking:v2:queue:' .. m end

local cKey = claimKey(claimID)
local existing = redis.call('HGETALL', cKey)
if #existing == 0 then
  redis.call('ZREM', 'matchmaking:v2:claims', claimID)
  return {'absent'}
end
local claim = {}
for i = 1, #existing, 2 do
  claim[existing[i]] = existing[i+1]
end
if claim['claim_id'] ~= claimID then
  return {'mismatch'}
end

local function releaseTicket(tid, userCSV, enqueuedAt, partyID, partyVersion, mode, teamSize, teamAvg, requeue)
  local map = {}
  local vals = redis.call('HGETALL', ticketKey(tid))
  if #vals == 0 then
    if userCSV and userCSV ~= '' then
      for uid in string.gmatch(userCSV, '[^,]+') do
        if redis.call('GET', userKey(uid)) == tid then
          redis.call('DEL', userKey(uid))
        end
      end
    end
    return
  end
  for i = 1, #vals, 2 do
    map[vals[i]] = vals[i+1]
  end
  if map['claim_id'] ~= claimID or map['ticket_id'] ~= tid then
    return
  end
  if requeue then
    redis.call('HSET', ticketKey(tid),
      'state', stateSearching,
      'claim_id', '',
      'lease_expires_at_ms', tostring(leaseExpiresMs),
      'enqueued_at_ms', tostring(enqueuedAt),
      'party_id', partyID or map['party_id'] or '',
      'party_version', tostring(partyVersion or map['party_version'] or '0'),
      'mode', mode or map['mode'] or '',
      'team_size', tostring(teamSize or map['team_size'] or '0'),
      'team_avg_rating', tostring(teamAvg or map['team_avg_rating'] or '0'),
      'user_ids', userCSV or map['user_ids'] or ''
    )
    redis.call('ZADD', queueKeyFor(mode or map['mode']), tonumber(enqueuedAt), tid)
    if userCSV and userCSV ~= '' then
      for uid in string.gmatch(userCSV, '[^,]+') do
        redis.call('SET', userKey(uid), tid)
      end
    end
  else
    if userCSV and userCSV ~= '' then
      for uid in string.gmatch(userCSV, '[^,]+') do
        if redis.call('GET', userKey(uid)) == tid then
          redis.call('DEL', userKey(uid))
        end
      end
    end
    redis.call('DEL', ticketKey(tid))
  end
end

releaseTicket(
  claim['ticket_id_a'], claim['user_ids_a'], tonumber(claim['enqueued_at_ms_a'] or '0'),
  claim['party_id_a'], claim['party_version_a'], claim['mode'], claim['team_size'], claim['team_avg_rating_a'], requeueA
)
releaseTicket(
  claim['ticket_id_b'], claim['user_ids_b'], tonumber(claim['enqueued_at_ms_b'] or '0'),
  claim['party_id_b'], claim['party_version_b'], claim['mode'], claim['team_size'], claim['team_avg_rating_b'], requeueB
)
redis.call('DEL', cKey)
redis.call('ZREM', 'matchmaking:v2:claims', claimID)
return {'released'}
`)

// JoinTicket adds a team roster ticket or returns an existing valid ticket for the same roster.
func (c *MatchmakingV2Coordinator) JoinTicket(ctx context.Context, input JoinTicketInput, now time.Time, leaseTTL time.Duration) (*QueueTicket, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	if err := validateJoinInput(input); err != nil {
		return nil, err
	}
	ticketID := uuid.NewString()
	nowMs := now.UTC().UnixMilli()
	leaseExpiresMs := now.UTC().Add(leaseTTL).UnixMilli()
	partyID := ""
	if input.PartyID != uuid.Nil {
		partyID = input.PartyID.String()
	}
	userCSV := joinUserIDs(input.UserIDs)
	args := make([]interface{}, 0, 10+len(input.UserIDs))
	args = append(args,
		ticketID,
		input.Mode,
		partyID,
		strconv.FormatInt(input.PartyVersion, 10),
		strconv.Itoa(input.TeamSize),
		strconv.Itoa(input.TeamAvgRating),
		userCSV,
		nowMs,
		leaseExpiresMs,
		len(input.UserIDs),
	)
	for _, id := range input.UserIDs {
		args = append(args, id.String())
	}

	result, err := joinTicketScript.Run(ctx, c.client, nil, args...).Slice()
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 join: %w", err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("matchmaking v2 join: empty result")
	}
	status := asString(result[0])
	switch status {
	case "err_user_conflict":
		return nil, errV2UserConflict
	case "err_party_version":
		return nil, errV2PartyVersionMismatch
	case "err_invalid":
		return nil, errV2InvalidRoster
	case "created", "existing", "existing_claimed":
		ticket, parseErr := parseQueueTicketResult(result[1:])
		if parseErr != nil {
			return nil, parseErr
		}
		return ticket, nil
	default:
		return nil, fmt.Errorf("matchmaking v2 join: unexpected status %q", status)
	}
}

// LeaveTicket removes the searching ticket that contains userID (whole roster).
func (c *MatchmakingV2Coordinator) LeaveTicket(ctx context.Context, userID uuid.UUID) error {
	if c == nil || c.client == nil {
		return errV2Unavailable
	}
	result, err := leaveTicketScript.Run(ctx, c.client, nil, userID.String()).Slice()
	if err != nil {
		return fmt.Errorf("matchmaking v2 leave: %w", err)
	}
	if len(result) > 0 {
		if status := asString(result[0]); status == "claimed" {
			return errV2ClaimInProgress
		}
	}
	return nil
}

// GetTicketByUser loads the ticket currently bound to userID, if any.
func (c *MatchmakingV2Coordinator) GetTicketByUser(ctx context.Context, userID uuid.UUID) (*QueueTicket, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	tid, err := c.client.Get(ctx, matchmakingV2UserKey(userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("matchmaking v2 get user pointer: %w", err)
	}
	return c.GetTicket(ctx, tid)
}

// GetTicket loads a ticket hash by id.
func (c *MatchmakingV2Coordinator) GetTicket(ctx context.Context, ticketID string) (*QueueTicket, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	if ticketID == "" {
		return nil, nil
	}
	vals, err := c.client.HGetAll(ctx, matchmakingV2TicketKey(ticketID)).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 get ticket: %w", err)
	}
	if len(vals) == 0 {
		return nil, nil
	}
	return queueTicketFromMap(vals)
}

// RenewTicketLease extends the lease for a searching ticket without changing priority.
// Claimed tickets are returned unchanged (owned by the recovery claim).
func (c *MatchmakingV2Coordinator) RenewTicketLease(ctx context.Context, userID uuid.UUID, leaseTTL time.Duration, now time.Time) (*QueueTicket, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	nowMs := now.UTC().UnixMilli()
	leaseExpiresMs := now.UTC().Add(leaseTTL).UnixMilli()
	result, err := renewTicketScript.Run(ctx, c.client, nil, userID.String(), nowMs, leaseExpiresMs).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("matchmaking v2 renew lease: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	slice, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("matchmaking v2 renew lease: unexpected result type %T", result)
	}
	return parseQueueTicketResult(slice)
}

// ClaimTickets scans up to scanLimit oldest candidates and claims two equal-roster tickets.
// maxRatingDelta <= 0 disables the rating proximity filter (Casual).
func (c *MatchmakingV2Coordinator) ClaimTickets(ctx context.Context, mode string, now time.Time, claimTTL time.Duration, scanLimit int, maxRatingDelta int) (*TeamClaim, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	if scanLimit < 2 {
		scanLimit = 2
	}
	nowMs := now.UTC().UnixMilli()
	recoverAfterMs := now.UTC().Add(claimTTL).UnixMilli()
	claimID := uuid.NewString()

	result, err := claimTicketsScript.Run(ctx, c.client, []string{
		matchmakingV2QueueKey(mode),
		matchmakingV2ClaimsIndexKey(),
	}, mode, nowMs, recoverAfterMs, claimID, scanLimit, maxRatingDelta).Slice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("matchmaking v2 claim: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return parseTeamClaimResult(result)
}

// FinalizeTeamClaim removes claimed ticket state after durable match commit.
func (c *MatchmakingV2Coordinator) FinalizeTeamClaim(ctx context.Context, claim *TeamClaim) error {
	if c == nil || c.client == nil {
		return errV2Unavailable
	}
	if claim == nil {
		return nil
	}
	_, err := finalizeTeamClaimScript.Run(ctx, c.client, nil,
		claim.ClaimID,
		claim.TicketIDA,
		claim.TicketIDB,
		joinUserIDs(claim.UserIDsA),
		joinUserIDs(claim.UserIDsB),
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("matchmaking v2 finalize claim: %w", err)
	}
	return nil
}

// GetTeamClaim loads a claim record by id.
func (c *MatchmakingV2Coordinator) GetTeamClaim(ctx context.Context, claimID string) (*TeamClaim, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	if claimID == "" {
		return nil, nil
	}
	vals, err := c.client.HGetAll(ctx, matchmakingV2ClaimKey(claimID)).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 get claim: %w", err)
	}
	if len(vals) == 0 {
		return nil, nil
	}
	return teamClaimFromMap(vals)
}

// ReleaseTeamClaim releases a claim and independently requeues each ticket when eligible.
// requeueA/requeueB control ticket A and B so an ineligible side does not discard the other.
// Callers must perform durable-first formation lookup before requeueing (service-owned).
func (c *MatchmakingV2Coordinator) ReleaseTeamClaim(ctx context.Context, claim *TeamClaim, requeueA, requeueB bool, leaseTTL time.Duration) error {
	if c == nil || c.client == nil {
		return errV2Unavailable
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
	_, err := releaseTeamClaimScript.Run(ctx, c.client, nil,
		claim.ClaimID, flagA, flagB, nowMs, leaseExpiresMs,
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("matchmaking v2 release claim: %w", err)
	}
	return nil
}

// ListExpiredTeamClaims returns up to limit claim IDs whose recover_after is in the past.
// Callers must check PostgreSQL formation precedence before requeueing (durable-first).
func (c *MatchmakingV2Coordinator) ListExpiredTeamClaims(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if c == nil || c.client == nil {
		return nil, errV2Unavailable
	}
	if limit <= 0 {
		limit = 20
	}
	nowMs := now.UTC().UnixMilli()
	ids, err := c.client.ZRangeByScore(ctx, matchmakingV2ClaimsIndexKey(), &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(nowMs, 10),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 list expired claims: %w", err)
	}
	return ids, nil
}

// DropTeamClaimIndex removes a stale claim id from the recovery index when the claim hash is gone.
func (c *MatchmakingV2Coordinator) DropTeamClaimIndex(ctx context.Context, claimID string) error {
	if c == nil || c.client == nil {
		return errV2Unavailable
	}
	if claimID == "" {
		return nil
	}
	if err := c.client.ZRem(ctx, matchmakingV2ClaimsIndexKey(), claimID).Err(); err != nil {
		return fmt.Errorf("matchmaking v2 drop claim index: %w", err)
	}
	return nil
}

func validateJoinInput(input JoinTicketInput) error {
	if strings.TrimSpace(input.Mode) == "" {
		return errV2InvalidRoster
	}
	if input.TeamSize < 1 {
		return errV2InvalidRoster
	}
	if len(input.UserIDs) != input.TeamSize {
		return errV2InvalidRoster
	}
	seen := make(map[uuid.UUID]struct{}, len(input.UserIDs))
	for _, id := range input.UserIDs {
		if id == uuid.Nil {
			return errV2InvalidRoster
		}
		if _, ok := seen[id]; ok {
			return errV2InvalidRoster
		}
		seen[id] = struct{}{}
	}
	return nil
}

func joinUserIDs(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return strings.Join(parts, ",")
}

func parseUserIDsCSV(csv string) ([]uuid.UUID, error) {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil, nil
	}
	parts := strings.Split(csv, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("matchmaking v2 user id: %w", err)
		}
		out = append(out, id)
	}
	return out, nil
}

func parseOptionalUUID(s string) uuid.UUID {
	s = strings.TrimSpace(s)
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func parseQueueTicketResult(result []interface{}) (*QueueTicket, error) {
	if len(result) < 11 {
		return nil, fmt.Errorf("matchmaking v2 ticket result: expected >=11 fields, got %d", len(result))
	}
	userIDs, err := parseUserIDsCSV(asString(result[3]))
	if err != nil {
		return nil, err
	}
	teamSize, err := strconv.Atoi(asString(result[4]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 team size: %w", err)
	}
	avg, err := strconv.Atoi(asString(result[5]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 team avg rating: %w", err)
	}
	enqueuedAt, err := strconv.ParseInt(asString(result[6]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 enqueued_at: %w", err)
	}
	leaseExpires, err := strconv.ParseInt(asString(result[7]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 lease: %w", err)
	}
	partyVersion, err := strconv.ParseInt(asString(result[10]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 party version: %w", err)
	}
	return &QueueTicket{
		TicketID:         asString(result[0]),
		PartyID:          parseOptionalUUID(asString(result[1])),
		Mode:             asString(result[2]),
		UserIDs:          userIDs,
		TeamSize:         teamSize,
		TeamAvgRating:    avg,
		EnqueuedAtMs:     enqueuedAt,
		LeaseExpiresAtMs: leaseExpires,
		State:            asString(result[8]),
		ClaimID:          asString(result[9]),
		PartyVersion:     partyVersion,
	}, nil
}

func queueTicketFromMap(vals map[string]string) (*QueueTicket, error) {
	userIDs, err := parseUserIDsCSV(vals[v2FieldUserIDs])
	if err != nil {
		return nil, err
	}
	teamSize, _ := strconv.Atoi(vals[v2FieldTeamSize])
	avg, _ := strconv.Atoi(vals[v2FieldTeamAvgRating])
	enqueuedAt, _ := strconv.ParseInt(vals[v2FieldEnqueuedAtMs], 10, 64)
	leaseExpires, _ := strconv.ParseInt(vals[v2FieldLeaseExpiresAt], 10, 64)
	partyVersion, _ := strconv.ParseInt(vals[v2FieldPartyVersion], 10, 64)
	return &QueueTicket{
		TicketID:         vals[v2FieldTicketID],
		PartyID:          parseOptionalUUID(vals[v2FieldPartyID]),
		Mode:             vals[v2FieldMode],
		UserIDs:          userIDs,
		TeamSize:         teamSize,
		TeamAvgRating:    avg,
		EnqueuedAtMs:     enqueuedAt,
		LeaseExpiresAtMs: leaseExpires,
		State:            vals[v2FieldState],
		ClaimID:          vals[v2FieldClaimID],
		PartyVersion:     partyVersion,
	}, nil
}

func parseTeamClaimResult(result []interface{}) (*TeamClaim, error) {
	// claim_id, formation_key, mode, tid_a, tid_b, users_a, users_b, party_a, party_b,
	// party_ver_a, party_ver_b, team_size, avg_a, avg_b, enq_a, enq_b, claimed_at, recover_after
	if len(result) < 18 {
		return nil, fmt.Errorf("matchmaking v2 claim result: expected >=18 fields, got %d", len(result))
	}
	usersA, err := parseUserIDsCSV(asString(result[5]))
	if err != nil {
		return nil, err
	}
	usersB, err := parseUserIDsCSV(asString(result[6]))
	if err != nil {
		return nil, err
	}
	teamSize, err := strconv.Atoi(asString(result[11]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim team size: %w", err)
	}
	avgA, err := strconv.Atoi(asString(result[12]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim avg a: %w", err)
	}
	avgB, err := strconv.Atoi(asString(result[13]))
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim avg b: %w", err)
	}
	enqA, err := strconv.ParseInt(asString(result[14]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim enqueued a: %w", err)
	}
	enqB, err := strconv.ParseInt(asString(result[15]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim enqueued b: %w", err)
	}
	claimedAt, err := strconv.ParseInt(asString(result[16]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim claimed_at: %w", err)
	}
	recoverAfter, err := strconv.ParseInt(asString(result[17]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("matchmaking v2 claim recover_after: %w", err)
	}
	partyVerA, _ := strconv.ParseInt(asString(result[9]), 10, 64)
	partyVerB, _ := strconv.ParseInt(asString(result[10]), 10, 64)
	return &TeamClaim{
		ClaimID:        asString(result[0]),
		FormationKey:   asString(result[1]),
		Mode:           asString(result[2]),
		TicketIDA:      asString(result[3]),
		TicketIDB:      asString(result[4]),
		UserIDsA:       usersA,
		UserIDsB:       usersB,
		PartyIDA:       parseOptionalUUID(asString(result[7])),
		PartyIDB:       parseOptionalUUID(asString(result[8])),
		PartyVersionA:  partyVerA,
		PartyVersionB:  partyVerB,
		TeamSize:       teamSize,
		TeamAvgRatingA: avgA,
		TeamAvgRatingB: avgB,
		EnqueuedAtMsA:  enqA,
		EnqueuedAtMsB:  enqB,
		ClaimedAtMs:    claimedAt,
		RecoverAfterMs: recoverAfter,
	}, nil
}

func teamClaimFromMap(vals map[string]string) (*TeamClaim, error) {
	usersA, err := parseUserIDsCSV(vals["user_ids_a"])
	if err != nil {
		return nil, err
	}
	usersB, err := parseUserIDsCSV(vals["user_ids_b"])
	if err != nil {
		return nil, err
	}
	teamSize, _ := strconv.Atoi(vals["team_size"])
	avgA, _ := strconv.Atoi(vals["team_avg_rating_a"])
	avgB, _ := strconv.Atoi(vals["team_avg_rating_b"])
	enqA, _ := strconv.ParseInt(vals["enqueued_at_ms_a"], 10, 64)
	enqB, _ := strconv.ParseInt(vals["enqueued_at_ms_b"], 10, 64)
	claimedAt, _ := strconv.ParseInt(vals["claimed_at_ms"], 10, 64)
	recoverAfter, _ := strconv.ParseInt(vals["recover_after_ms"], 10, 64)
	partyVerA, _ := strconv.ParseInt(vals["party_version_a"], 10, 64)
	partyVerB, _ := strconv.ParseInt(vals["party_version_b"], 10, 64)
	return &TeamClaim{
		ClaimID:        vals["claim_id"],
		FormationKey:   vals["formation_key"],
		Mode:           vals["mode"],
		TicketIDA:      vals["ticket_id_a"],
		TicketIDB:      vals["ticket_id_b"],
		UserIDsA:       usersA,
		UserIDsB:       usersB,
		PartyIDA:       parseOptionalUUID(vals["party_id_a"]),
		PartyIDB:       parseOptionalUUID(vals["party_id_b"]),
		PartyVersionA:  partyVerA,
		PartyVersionB:  partyVerB,
		TeamSize:       teamSize,
		TeamAvgRatingA: avgA,
		TeamAvgRatingB: avgB,
		EnqueuedAtMsA:  enqA,
		EnqueuedAtMsB:  enqB,
		ClaimedAtMs:    claimedAt,
		RecoverAfterMs: recoverAfter,
	}, nil
}
