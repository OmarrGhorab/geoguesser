package redis

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	realtimeKeyVersion  = "v1"
	matchplayKeyVersion = "v1"
	partyKeyVersion     = "v1"

	// DefaultRealtimeTicketTTL is the default one-time WebSocket ticket lifetime.
	DefaultRealtimeTicketTTL = 30 * time.Second

	channelKindParty = "party"
	channelKindMatch = "match"

	ticketFieldUserID      = "user_id"
	ticketFieldChannelKind = "channel_kind"
	ticketFieldChannelID   = "channel_id"
	ticketFieldExpiresAt   = "expires_at"

	reconnectFieldLastVersion = "last_version"
)

// Sentinel errors for realtime ticket consumption.
var (
	ErrTicketInvalid = errors.New("realtime ticket invalid")
	ErrTicketUsed    = errors.New("realtime ticket already used")
	ErrTicketExpired = errors.New("realtime ticket expired")
)

// RealtimeStore manages ephemeral realtime tickets, channel versions, presence, and reconnect state.
type RealtimeStore struct {
	client *redis.Client
}

// NewRealtimeStore constructs a Redis-backed realtime store.
func NewRealtimeStore(client *redis.Client) *RealtimeStore {
	return &RealtimeStore{client: client}
}

// TicketClaims are the bound authorization claims recovered when a ticket is consumed.
type TicketClaims struct {
	UserID      uuid.UUID
	ChannelKind string // "party" | "match"
	ChannelID   string
	ExpiresAt   time.Time
}

// TicketKeyHash returns the SHA-256 hex digest used in Redis ticket keys.
// The opaque token itself must never appear in Redis keys or logs.
func TicketKeyHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// realtimeTicketKey builds realtime:v1:ticket:{sha256hex}.
func realtimeTicketKey(tokenHash string) string {
	return fmt.Sprintf("realtime:%s:ticket:%s", realtimeKeyVersion, tokenHash)
}

// realtimeChannelVersionKey builds realtime:v1:channel:{kind}:{id}:version.
func realtimeChannelVersionKey(channelKind, channelID string) string {
	return fmt.Sprintf("realtime:%s:channel:%s:%s:version", realtimeKeyVersion, channelKind, channelID)
}

// matchPresenceKey builds matchplay:v1:{match_id}:presence:{user_id}.
func matchPresenceKey(matchID string, userID uuid.UUID) string {
	return fmt.Sprintf("matchplay:%s:%s:presence:%s", matchplayKeyVersion, matchID, userID.String())
}

// matchReconnectKey builds matchplay:v1:{match_id}:reconnect:{user_id}.
func matchReconnectKey(matchID string, userID uuid.UUID) string {
	return fmt.Sprintf("matchplay:%s:%s:reconnect:%s", matchplayKeyVersion, matchID, userID.String())
}

// partyPresenceKey builds party:v1:{party_id}:presence:{user_id}.
func partyPresenceKey(partyID string, userID uuid.UUID) string {
	return fmt.Sprintf("party:%s:%s:presence:%s", partyKeyVersion, partyID, userID.String())
}

// partyReconnectKey builds party:v1:{party_id}:reconnect:{user_id}.
func partyReconnectKey(partyID string, userID uuid.UUID) string {
	return fmt.Sprintf("party:%s:%s:reconnect:%s", partyKeyVersion, partyID, userID.String())
}

func presenceConnectionsKey(channelKind, channelID string, userID uuid.UUID) (string, error) {
	key, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return "", err
	}
	return key + ":connections", nil
}

func presenceKey(channelKind, channelID string, userID uuid.UUID) (string, error) {
	switch channelKind {
	case channelKindMatch:
		return matchPresenceKey(channelID, userID), nil
	case channelKindParty:
		return partyPresenceKey(channelID, userID), nil
	default:
		return "", fmt.Errorf("unsupported channel kind %q", channelKind)
	}
}

func reconnectKey(channelKind, channelID string, userID uuid.UUID) (string, error) {
	switch channelKind {
	case channelKindMatch:
		return matchReconnectKey(channelID, userID), nil
	case channelKindParty:
		return partyReconnectKey(channelID, userID), nil
	default:
		return "", fmt.Errorf("unsupported channel kind %q", channelKind)
	}
}

func validateChannel(channelKind, channelID string) error {
	switch channelKind {
	case channelKindParty, channelKindMatch:
	default:
		return fmt.Errorf("%w: channel kind", ErrTicketInvalid)
	}
	if strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("%w: channel id", ErrTicketInvalid)
	}
	return nil
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate realtime ticket: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// IssueTicket creates a one-time realtime ticket bound to user and channel.
// Returns the opaque token (never store or log this value). Default TTL is 30s when ttl <= 0.
func (s *RealtimeStore) IssueTicket(ctx context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("realtime redis unavailable")
	}
	if userID == uuid.Nil {
		return "", fmt.Errorf("%w: user id", ErrTicketInvalid)
	}
	if err := validateChannel(channelKind, channelID); err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = DefaultRealtimeTicketTTL
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	key := realtimeTicketKey(TicketKeyHash(token))

	// Pipeline HSET + EXPIRE so the ticket is fully written with TTL.
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key,
		ticketFieldUserID, userID.String(),
		ticketFieldChannelKind, channelKind,
		ticketFieldChannelID, channelID,
		ticketFieldExpiresAt, strconv.FormatInt(expiresAt.UnixMilli(), 10),
	)
	pipe.PExpire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("issue realtime ticket: %w", err)
	}
	return token, nil
}

// consumeTicketScript atomically reads and deletes a ticket hash.
// KEYS[1] = ticket key
// ARGV[1] = now_ms
// Returns: status, user_id, channel_kind, channel_id, expires_at_ms
// status: ok | missing | expired
var consumeTicketScript = redis.NewScript(`
local key = KEYS[1]
local nowMs = tonumber(ARGV[1])

local vals = redis.call('HGETALL', key)
if #vals == 0 then
  return {'missing', '', '', '', '0'}
end

local map = {}
for i = 1, #vals, 2 do
  map[vals[i]] = vals[i+1]
end

local expiresAt = tonumber(map['expires_at'] or '0')
redis.call('DEL', key)

if expiresAt > 0 and expiresAt < nowMs then
  return {'expired', map['user_id'] or '', map['channel_kind'] or '', map['channel_id'] or '', tostring(expiresAt)}
end

return {'ok', map['user_id'] or '', map['channel_kind'] or '', map['channel_id'] or '', tostring(expiresAt)}
`)

var acquirePresenceScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
redis.call('PEXPIRE', KEYS[1], ARGV[1])
redis.call('SET', KEYS[2], 'connected', 'PX', ARGV[1])
redis.call('DEL', KEYS[3])
return count
`)

var releasePresenceScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current <= 0 then
  return -1
end
local remaining = current - 1
if remaining > 0 then
  redis.call('SET', KEYS[1], remaining, 'PX', ARGV[1])
  redis.call('SET', KEYS[2], 'connected', 'PX', ARGV[1])
  return remaining
end
redis.call('DEL', KEYS[1], KEYS[2])
redis.call('HSET', KEYS[3], 'last_version', ARGV[2])
redis.call('PEXPIRE', KEYS[3], ARGV[1])
return 0
`)

var refreshPresenceScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current <= 0 then
  return 0
end
redis.call('PEXPIRE', KEYS[1], ARGV[1])
redis.call('SET', KEYS[2], 'connected', 'PX', ARGV[1])
return current
`)

// ConsumeTicket atomically consumes a one-time ticket.
// A successful consume binds the WebSocket upgrade to the stored user and channel.
// Replay of the same token fails with ErrTicketUsed.
func (s *RealtimeStore) ConsumeTicket(ctx context.Context, opaqueToken string) (TicketClaims, error) {
	var zero TicketClaims
	if s == nil || s.client == nil {
		return zero, errors.New("realtime redis unavailable")
	}
	if strings.TrimSpace(opaqueToken) == "" {
		return zero, ErrTicketInvalid
	}

	nowMs := time.Now().UTC().UnixMilli()
	key := realtimeTicketKey(TicketKeyHash(opaqueToken))
	result, err := consumeTicketScript.Run(ctx, s.client, []string{key}, nowMs).Slice()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, ErrTicketUsed
		}
		return zero, fmt.Errorf("consume realtime ticket: %w", err)
	}
	if len(result) < 5 {
		return zero, ErrTicketInvalid
	}

	status := asString(result[0])
	switch status {
	case "missing":
		return zero, ErrTicketUsed
	case "expired":
		return zero, ErrTicketExpired
	case "ok":
		// continue
	default:
		return zero, ErrTicketInvalid
	}

	userID, err := uuid.Parse(asString(result[1]))
	if err != nil {
		return zero, ErrTicketInvalid
	}
	channelKind := asString(result[2])
	channelID := asString(result[3])
	if err := validateChannel(channelKind, channelID); err != nil {
		return zero, ErrTicketInvalid
	}
	expiresMs, err := strconv.ParseInt(asString(result[4]), 10, 64)
	if err != nil {
		return zero, ErrTicketInvalid
	}

	return TicketClaims{
		UserID:      userID,
		ChannelKind: channelKind,
		ChannelID:   channelID,
		ExpiresAt:   time.UnixMilli(expiresMs).UTC(),
	}, nil
}

// IncrChannelVersion atomically increments the monotonic channel version counter.
func (s *RealtimeStore) IncrChannelVersion(ctx context.Context, channelKind, channelID string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("realtime redis unavailable")
	}
	if err := validateChannel(channelKind, channelID); err != nil {
		return 0, err
	}
	version, err := s.client.Incr(ctx, realtimeChannelVersionKey(channelKind, channelID)).Result()
	if err != nil {
		return 0, fmt.Errorf("incr channel version: %w", err)
	}
	return version, nil
}

// GetChannelVersion returns the current channel version, or 0 when unset.
func (s *RealtimeStore) GetChannelVersion(ctx context.Context, channelKind, channelID string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("realtime redis unavailable")
	}
	if err := validateChannel(channelKind, channelID); err != nil {
		return 0, err
	}
	version, err := s.client.Get(ctx, realtimeChannelVersionKey(channelKind, channelID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get channel version: %w", err)
	}
	return version, nil
}

// SetPresence records connection presence for a party or match channel member.
func (s *RealtimeStore) SetPresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID, status string, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return errors.New("realtime redis unavailable")
	}
	if userID == uuid.Nil {
		return fmt.Errorf("%w: user id", ErrTicketInvalid)
	}
	key, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	if err := s.client.Set(ctx, key, status, ttl).Err(); err != nil {
		return fmt.Errorf("set presence: %w", err)
	}
	return nil
}

// AcquirePresence atomically registers one socket and clears any reconnect window.
// The returned count is global across API instances.
func (s *RealtimeStore) AcquirePresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID, ttl time.Duration) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("realtime redis unavailable")
	}
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	countKey, err := presenceConnectionsKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	presence, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	reconnect, err := reconnectKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	count, err := acquirePresenceScript.Run(ctx, s.client, []string{countKey, presence, reconnect}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("acquire presence: %w", err)
	}
	return count, nil
}

// ReleasePresence atomically unregisters one socket. The final socket creates
// the reconnect grace window and returns zero; duplicate releases return -1.
func (s *RealtimeStore) ReleasePresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID, lastVersion int64, grace time.Duration) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("realtime redis unavailable")
	}
	if grace <= 0 {
		grace = 90 * time.Second
	}
	countKey, err := presenceConnectionsKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	presence, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	reconnect, err := reconnectKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	remaining, err := releasePresenceScript.Run(ctx, s.client, []string{countKey, presence, reconnect}, grace.Milliseconds(), lastVersion).Int64()
	if err != nil {
		return 0, fmt.Errorf("release presence: %w", err)
	}
	return remaining, nil
}

// RefreshPresence extends the socket-count and visible presence TTLs without
// changing the number of active connections. Zero means the count was lost.
func (s *RealtimeStore) RefreshPresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID, ttl time.Duration) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("realtime redis unavailable")
	}
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	countKey, err := presenceConnectionsKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	presence, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return 0, err
	}
	count, err := refreshPresenceScript.Run(ctx, s.client, []string{countKey, presence}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("refresh presence: %w", err)
	}
	return count, nil
}

// GetPresence returns the presence status string, or empty when absent.
func (s *RealtimeStore) GetPresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("realtime redis unavailable")
	}
	key, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return "", err
	}
	status, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get presence: %w", err)
	}
	return status, nil
}

// DeletePresence removes a presence key.
func (s *RealtimeStore) DeletePresence(ctx context.Context, channelKind, channelID string, userID uuid.UUID) error {
	if s == nil || s.client == nil {
		return errors.New("realtime redis unavailable")
	}
	key, err := presenceKey(channelKind, channelID, userID)
	if err != nil {
		return err
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete presence: %w", err)
	}
	return nil
}

// SetReconnectWindow stores the last delivered channel version for reconnect recovery.
func (s *RealtimeStore) SetReconnectWindow(ctx context.Context, channelKind, channelID string, userID uuid.UUID, lastVersion int64, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return errors.New("realtime redis unavailable")
	}
	if userID == uuid.Nil {
		return fmt.Errorf("%w: user id", ErrTicketInvalid)
	}
	key, err := reconnectKey(channelKind, channelID, userID)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, reconnectFieldLastVersion, lastVersion)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set reconnect window: %w", err)
	}
	return nil
}

// GetReconnectWindow returns the last version and whether a reconnect window exists.
func (s *RealtimeStore) GetReconnectWindow(ctx context.Context, channelKind, channelID string, userID uuid.UUID) (int64, bool, error) {
	if s == nil || s.client == nil {
		return 0, false, errors.New("realtime redis unavailable")
	}
	key, err := reconnectKey(channelKind, channelID, userID)
	if err != nil {
		return 0, false, err
	}
	value, err := s.client.HGet(ctx, key, reconnectFieldLastVersion).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get reconnect window: %w", err)
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse reconnect version: %w", err)
	}
	return version, true, nil
}

// DeleteReconnectWindow removes a reconnect window hash.
func (s *RealtimeStore) DeleteReconnectWindow(ctx context.Context, channelKind, channelID string, userID uuid.UUID) error {
	if s == nil || s.client == nil {
		return errors.New("realtime redis unavailable")
	}
	key, err := reconnectKey(channelKind, channelID, userID)
	if err != nil {
		return err
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete reconnect window: %w", err)
	}
	return nil
}
