package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Match live-state key layout (data-model.md):
//
//	matchplay:v1:{match_id}:version
//	matchplay:v1:{match_id}:presence:{user_id}
//	matchplay:v1:{match_id}:reconnect:{user_id}
//	matchplay:v1:{match_id}:round:{round_id}:markers:{team}
//	matchplay:v1:{match_id}:round:{round_id}:views
//	matchplay:v1:{match_id}:round:{round_id}:locked
//	matchplay:v1:{match_id}:command:{user_id}:{command_id}
//	matchplay:v1:{match_id}:throttle:{kind}:{user_id}
//
// Keys never embed opaque realtime tickets or access tokens.

const (
	// DefaultMarkerRateLimit is the max marker sets per user per window.
	DefaultMarkerRateLimit = 2
	// DefaultMarkerRateWindow is the marker throttle window.
	DefaultMarkerRateWindow = time.Second
	// DefaultViewRateLimit is the max view updates per user per window.
	DefaultViewRateLimit = 4
	// DefaultViewRateWindow is the view throttle window.
	DefaultViewRateWindow = time.Second
	// DefaultRoundLiveTTL is the TTL applied to round-scoped live keys.
	DefaultRoundLiveTTL = 15 * time.Minute
	// DefaultCommandAckTTL is how long command acknowledgement payloads are retained.
	DefaultCommandAckTTL = 5 * time.Minute
	// DefaultMatchPresenceTTL is the presence/reconnect grace window.
	DefaultMatchPresenceTTL = 90 * time.Second
)

// Sentinel errors for match live-state operations.
var (
	ErrMarkerThrottled     = errors.New("marker rate limited")
	ErrViewThrottled       = errors.New("view rate limited")
	ErrViewUnchanged       = errors.New("view not materially changed")
	ErrViewForbiddenFields = errors.New("view contains forbidden fields")
	ErrPlayerLocked        = errors.New("player guess locked")
	ErrLiveStateInvalid    = errors.New("match live state invalid")
	ErrCommandReplay       = errors.New("command already processed")
)

// Material-change thresholds for provider-safe scene comparison (matches matchplay policy).
const (
	viewHeadingEpsilon = 0.5
	viewPitchEpsilon   = 0.5
	viewZoomEpsilon    = 0.05
)

// forbiddenViewFields must never be stored in scene state.
var forbiddenViewFields = map[string]struct{}{
	"latitude": {}, "longitude": {}, "lat": {}, "lng": {},
	"marker": {}, "markers": {}, "map": {}, "cursor": {},
	"guess": {}, "guesses": {}, "score": {}, "scores": {},
	"answer": {}, "actual_location": {}, "distance": {}, "distance_meters": {},
	"accuracy_score": {}, "speed_bonus": {}, "total_score": {},
}

// MarkerRecord is the current proposed map marker for one teammate.
type MarkerRecord struct {
	UserID    uuid.UUID
	Latitude  float64
	Longitude float64
	Version   int64
}

// ViewRecord is a provider-safe panorama/image view (no map/marker/guess fields).
type ViewRecord struct {
	UserID     uuid.UUID       `json:"user_id"`
	PanoramaID string          `json:"panorama_id,omitempty"`
	Heading    float64         `json:"heading"`
	Pitch      float64         `json:"pitch"`
	Zoom       float64         `json:"zoom"`
	Extra      json.RawMessage `json:"-"` // rejected on write; never round-tripped from client map/marker fields
	Version    int64           `json:"version"`
}

// CommandAck is a stored WebSocket command acknowledgement for exact-once replay.
type CommandAck struct {
	CommandID string
	OK        bool
	Code      string
	Message   string
	Version   int64
	Payload   json.RawMessage
}

type markerPayload struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Ver int64   `json:"v"`
}

// MatchLiveStore manages disposable match collaboration state in Redis.
type MatchLiveStore struct {
	client *redis.Client
}

// NewMatchLiveStore constructs a Redis-backed match live-state store.
func NewMatchLiveStore(client *redis.Client) *MatchLiveStore {
	return &MatchLiveStore{client: client}
}

// --- key builders (exported for redaction tests; never include secrets) ---

func matchplayVersionKey(matchID string) string {
	return fmt.Sprintf("matchplay:%s:%s:version", matchplayKeyVersion, matchID)
}

func matchplayMarkerKey(matchID, roundID string, teamSlot int) string {
	return fmt.Sprintf("matchplay:%s:%s:round:%s:markers:%d", matchplayKeyVersion, matchID, roundID, teamSlot)
}

func matchplayViewsKey(matchID, roundID string) string {
	return fmt.Sprintf("matchplay:%s:%s:round:%s:views", matchplayKeyVersion, matchID, roundID)
}

func matchplayLockedKey(matchID, roundID string) string {
	return fmt.Sprintf("matchplay:%s:%s:round:%s:locked", matchplayKeyVersion, matchID, roundID)
}

func matchplayCommandKey(matchID string, userID uuid.UUID, commandID string) string {
	return fmt.Sprintf("matchplay:%s:%s:command:%s:%s", matchplayKeyVersion, matchID, userID.String(), commandID)
}

func matchplayThrottleKey(matchID, kind string, userID uuid.UUID) string {
	return fmt.Sprintf("matchplay:%s:%s:throttle:%s:%s", matchplayKeyVersion, matchID, kind, userID.String())
}

// RedactedKeySample returns a key with UUIDs replaced so logs never leak identifiers
// that could be correlated with precise private state. Tests assert ticket tokens
// never appear in any key builder output.
func RedactedKeySample(key string) string {
	// Replace UUID-shaped segments with "*".
	parts := strings.Split(key, ":")
	for i, p := range parts {
		if len(p) == 36 && strings.Count(p, "-") == 4 {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, ":")
}

// IncrVersion atomically increments the match live version counter.
func (s *MatchLiveStore) IncrVersion(ctx context.Context, matchID string) (int64, error) {
	if err := s.require(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(matchID) == "" {
		return 0, fmt.Errorf("%w: match id", ErrLiveStateInvalid)
	}
	v, err := s.client.Incr(ctx, matchplayVersionKey(matchID)).Result()
	if err != nil {
		return 0, fmt.Errorf("incr match version: %w", err)
	}
	return v, nil
}

// GetVersion returns the current match live version, or 0 when unset.
func (s *MatchLiveStore) GetVersion(ctx context.Context, matchID string) (int64, error) {
	if err := s.require(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(matchID) == "" {
		return 0, fmt.Errorf("%w: match id", ErrLiveStateInvalid)
	}
	v, err := s.client.Get(ctx, matchplayVersionKey(matchID)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get match version: %w", err)
	}
	return v, nil
}

// SetPresence records online presence for a match participant.
func (s *MatchLiveStore) SetPresence(ctx context.Context, matchID string, userID uuid.UUID, status string, ttl time.Duration) error {
	if err := s.require(); err != nil {
		return err
	}
	if userID == uuid.Nil || strings.TrimSpace(matchID) == "" {
		return fmt.Errorf("%w: match/user", ErrLiveStateInvalid)
	}
	if ttl <= 0 {
		ttl = DefaultMatchPresenceTTL
	}
	if status == "" {
		status = "connected"
	}
	if err := s.client.Set(ctx, matchPresenceKey(matchID, userID), status, ttl).Err(); err != nil {
		return fmt.Errorf("set match presence: %w", err)
	}
	return nil
}

// GetPresence returns the presence status, or empty when absent/expired.
func (s *MatchLiveStore) GetPresence(ctx context.Context, matchID string, userID uuid.UUID) (string, error) {
	if err := s.require(); err != nil {
		return "", err
	}
	status, err := s.client.Get(ctx, matchPresenceKey(matchID, userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get match presence: %w", err)
	}
	return status, nil
}

// SetReconnectWindow stores last delivered version for reconnect recovery.
func (s *MatchLiveStore) SetReconnectWindow(ctx context.Context, matchID string, userID uuid.UUID, lastVersion int64, ttl time.Duration) error {
	if err := s.require(); err != nil {
		return err
	}
	if userID == uuid.Nil || strings.TrimSpace(matchID) == "" {
		return fmt.Errorf("%w: match/user", ErrLiveStateInvalid)
	}
	if ttl <= 0 {
		ttl = DefaultMatchPresenceTTL
	}
	key := matchReconnectKey(matchID, userID)
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, key, reconnectFieldLastVersion, lastVersion)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set match reconnect: %w", err)
	}
	return nil
}

// GetReconnectWindow returns the last version and whether the window exists.
func (s *MatchLiveStore) GetReconnectWindow(ctx context.Context, matchID string, userID uuid.UUID) (int64, bool, error) {
	if err := s.require(); err != nil {
		return 0, false, err
	}
	value, err := s.client.HGet(ctx, matchReconnectKey(matchID, userID), reconnectFieldLastVersion).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get match reconnect: %w", err)
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse reconnect version: %w", err)
	}
	return version, true, nil
}

// MarkGuessLocked records that a player may no longer change their proposed marker.
func (s *MatchLiveStore) MarkGuessLocked(ctx context.Context, matchID, roundID string, userID uuid.UUID, ttl time.Duration) error {
	if err := s.require(); err != nil {
		return err
	}
	if err := validateMatchRound(matchID, roundID, userID); err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = DefaultRoundLiveTTL
	}
	key := matchplayLockedKey(matchID, roundID)
	pipe := s.client.TxPipeline()
	pipe.SAdd(ctx, key, userID.String())
	pipe.PExpire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("mark guess locked: %w", err)
	}
	return nil
}

// IsGuessLocked reports whether the player is locked for the round.
func (s *MatchLiveStore) IsGuessLocked(ctx context.Context, matchID, roundID string, userID uuid.UUID) (bool, error) {
	if err := s.require(); err != nil {
		return false, err
	}
	if err := validateMatchRound(matchID, roundID, userID); err != nil {
		return false, err
	}
	ok, err := s.client.SIsMember(ctx, matchplayLockedKey(matchID, roundID), userID.String()).Result()
	if err != nil {
		return false, fmt.Errorf("is guess locked: %w", err)
	}
	return ok, nil
}

// setMarkerScript atomically throttles, rejects locked players, stores marker, and bumps version.
// KEYS[1]=throttle KEYS[2]=markers KEYS[3]=locked KEYS[4]=version
// ARGV: userID, payloadJSON, nowMs, limit, windowMs, ttlMs
// Returns: {status, version} status=ok|throttled|locked
var setMarkerScript = redis.NewScript(`
local throttleKey = KEYS[1]
local markersKey = KEYS[2]
local lockedKey = KEYS[3]
local versionKey = KEYS[4]
local userID = ARGV[1]
local payload = ARGV[2]
local nowMs = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local windowMs = tonumber(ARGV[5])
local ttlMs = tonumber(ARGV[6])

if redis.call('SISMEMBER', lockedKey, userID) == 1 then
  return {'locked', '0'}
end

redis.call('ZREMRANGEBYSCORE', throttleKey, 0, nowMs - windowMs)
local count = redis.call('ZCARD', throttleKey)
if count >= limit then
  return {'throttled', '0'}
end
redis.call('ZADD', throttleKey, nowMs, tostring(nowMs) .. '-' .. userID .. '-' .. tostring(count))
redis.call('PEXPIRE', throttleKey, windowMs)

local ver = redis.call('INCR', versionKey)
local decoded = cjson.decode(payload)
decoded['v'] = ver
local stored = cjson.encode(decoded)
redis.call('HSET', markersKey, userID, stored)
if ttlMs > 0 then
  redis.call('PEXPIRE', markersKey, ttlMs)
  redis.call('PEXPIRE', lockedKey, ttlMs)
end
return {'ok', tostring(ver)}
`)

// SetMarker stores a team-scoped proposed marker with 2/s throttle and lock checks.
// teamSlot must be 1 or 2. Returns the new channel/marker version on success.
func (s *MatchLiveStore) SetMarker(
	ctx context.Context,
	matchID, roundID string,
	teamSlot int,
	userID uuid.UUID,
	lat, lng float64,
	ttl time.Duration,
) (MarkerRecord, error) {
	var zero MarkerRecord
	if err := s.require(); err != nil {
		return zero, err
	}
	if err := validateMatchRound(matchID, roundID, userID); err != nil {
		return zero, err
	}
	if teamSlot != 1 && teamSlot != 2 {
		return zero, fmt.Errorf("%w: team slot", ErrLiveStateInvalid)
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return zero, fmt.Errorf("%w: coordinates", ErrLiveStateInvalid)
	}
	if ttl <= 0 {
		ttl = DefaultRoundLiveTTL
	}

	body, err := json.Marshal(markerPayload{Lat: lat, Lng: lng})
	if err != nil {
		return zero, err
	}
	nowMs := time.Now().UTC().UnixMilli()
	ttlMs := ttl.Milliseconds()
	result, err := setMarkerScript.Run(ctx, s.client, []string{
		matchplayThrottleKey(matchID, "marker", userID),
		matchplayMarkerKey(matchID, roundID, teamSlot),
		matchplayLockedKey(matchID, roundID),
		matchplayVersionKey(matchID),
	}, userID.String(), string(body), nowMs, DefaultMarkerRateLimit, DefaultMarkerRateWindow.Milliseconds(), ttlMs).Slice()
	if err != nil {
		return zero, fmt.Errorf("set marker: %w", err)
	}
	if len(result) < 2 {
		return zero, ErrLiveStateInvalid
	}
	switch asString(result[0]) {
	case "ok":
		ver, _ := strconv.ParseInt(asString(result[1]), 10, 64)
		return MarkerRecord{UserID: userID, Latitude: lat, Longitude: lng, Version: ver}, nil
	case "throttled":
		return zero, ErrMarkerThrottled
	case "locked":
		return zero, ErrPlayerLocked
	default:
		return zero, ErrLiveStateInvalid
	}
}

// GetTeamMarkers returns all proposed markers for a team on a round (snapshot recovery).
func (s *MatchLiveStore) GetTeamMarkers(ctx context.Context, matchID, roundID string, teamSlot int) ([]MarkerRecord, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(roundID) == "" {
		return nil, fmt.Errorf("%w: match/round", ErrLiveStateInvalid)
	}
	if teamSlot != 1 && teamSlot != 2 {
		return nil, fmt.Errorf("%w: team slot", ErrLiveStateInvalid)
	}
	raw, err := s.client.HGetAll(ctx, matchplayMarkerKey(matchID, roundID, teamSlot)).Result()
	if err != nil {
		return nil, fmt.Errorf("get team markers: %w", err)
	}
	out := make([]MarkerRecord, 0, len(raw))
	for userStr, payload := range raw {
		uid, err := uuid.Parse(userStr)
		if err != nil {
			continue
		}
		var m markerPayload
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			continue
		}
		out = append(out, MarkerRecord{
			UserID:    uid,
			Latitude:  m.Lat,
			Longitude: m.Lng,
			Version:   m.Ver,
		})
	}
	return out, nil
}

// setViewScript throttles view updates (4/s) and stores provider-safe scene JSON.
// KEYS[1]=throttle KEYS[2]=views KEYS[3]=version
// ARGV: userID, payloadJSON, nowMs, limit, windowMs, ttlMs
var setViewScript = redis.NewScript(`
local throttleKey = KEYS[1]
local viewsKey = KEYS[2]
local versionKey = KEYS[3]
local userID = ARGV[1]
local payload = ARGV[2]
local nowMs = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local windowMs = tonumber(ARGV[5])
local ttlMs = tonumber(ARGV[6])

redis.call('ZREMRANGEBYSCORE', throttleKey, 0, nowMs - windowMs)
local count = redis.call('ZCARD', throttleKey)
if count >= limit then
  return {'throttled', '0'}
end
redis.call('ZADD', throttleKey, nowMs, tostring(nowMs) .. '-' .. userID .. '-' .. tostring(count))
redis.call('PEXPIRE', throttleKey, windowMs)

local ver = redis.call('INCR', versionKey)
local decoded = cjson.decode(payload)
decoded['version'] = ver
local stored = cjson.encode(decoded)
redis.call('HSET', viewsKey, userID, stored)
if ttlMs > 0 then
  redis.call('PEXPIRE', viewsKey, ttlMs)
end
return {'ok', tostring(ver)}
`)

// ValidateViewFields rejects map/marker/guess/answer private fields and unknown keys.
// Allowed: panorama_id, heading, pitch, zoom, and optional image pan/zoom helpers.
func ValidateViewFields(fields map[string]any) error {
	if fields == nil {
		return fmt.Errorf("%w: empty view", ErrLiveStateInvalid)
	}
	for key := range fields {
		k := strings.ToLower(strings.TrimSpace(key))
		if _, bad := forbiddenViewFields[k]; bad {
			return ErrViewForbiddenFields
		}
		switch k {
		case "panorama_id", "heading", "pitch", "zoom",
			"user_id", "version", "v",
			"image_pan_x", "image_pan_y", "image_zoom",
			"round_id":
			continue
		default:
			return ErrViewForbiddenFields
		}
	}
	return nil
}

// ViewMateriallyChanged reports whether next differs from prev enough to store/fanout.
func ViewMateriallyChanged(prev, next ViewRecord) bool {
	if strings.TrimSpace(prev.PanoramaID) != strings.TrimSpace(next.PanoramaID) {
		return true
	}
	if absFloat(prev.Heading-next.Heading) >= viewHeadingEpsilon {
		return true
	}
	if absFloat(prev.Pitch-next.Pitch) >= viewPitchEpsilon {
		return true
	}
	if absFloat(prev.Zoom-next.Zoom) >= viewZoomEpsilon {
		return true
	}
	return false
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// SetView stores a provider-safe view state with 4/s throttle and material-change dedup.
// Returns ErrViewUnchanged (with the prior record/version) when the scene did not change materially.
// Rejects out-of-range heading/pitch/zoom. Never persists map/marker/guess fields.
func (s *MatchLiveStore) SetView(
	ctx context.Context,
	matchID, roundID string,
	userID uuid.UUID,
	view ViewRecord,
	ttl time.Duration,
) (ViewRecord, error) {
	var zero ViewRecord
	if err := s.require(); err != nil {
		return zero, err
	}
	if err := validateMatchRound(matchID, roundID, userID); err != nil {
		return zero, err
	}
	if err := validateViewRecord(view); err != nil {
		return zero, err
	}
	if ttl <= 0 {
		ttl = DefaultRoundLiveTTL
	}
	view.UserID = userID

	// Material-change dedup: skip throttle/version when scene is effectively unchanged.
	if existing, err := s.GetView(ctx, matchID, roundID, userID); err == nil && existing != nil {
		if !ViewMateriallyChanged(*existing, view) {
			return *existing, ErrViewUnchanged
		}
	}

	// Only allow provider-safe fields on the wire to Redis.
	safe := map[string]any{
		"user_id": userID.String(),
		"heading": view.Heading,
		"pitch":   view.Pitch,
		"zoom":    view.Zoom,
	}
	if view.PanoramaID != "" {
		safe["panorama_id"] = view.PanoramaID
	}
	if err := ValidateViewFields(safe); err != nil {
		return zero, err
	}
	body, err := json.Marshal(safe)
	if err != nil {
		return zero, err
	}
	nowMs := time.Now().UTC().UnixMilli()
	result, err := setViewScript.Run(ctx, s.client, []string{
		matchplayThrottleKey(matchID, "view", userID),
		matchplayViewsKey(matchID, roundID),
		matchplayVersionKey(matchID),
	}, userID.String(), string(body), nowMs, DefaultViewRateLimit, DefaultViewRateWindow.Milliseconds(), ttl.Milliseconds()).Slice()
	if err != nil {
		return zero, fmt.Errorf("set view: %w", err)
	}
	if len(result) < 2 {
		return zero, ErrLiveStateInvalid
	}
	switch asString(result[0]) {
	case "ok":
		ver, _ := strconv.ParseInt(asString(result[1]), 10, 64)
		view.Version = ver
		return view, nil
	case "throttled":
		return zero, ErrViewThrottled
	default:
		return zero, ErrLiveStateInvalid
	}
}

// SetViewFromMap validates a client field map then stores provider-safe scene state.
// Explicitly rejects map/marker/guess/answer fields before write.
func (s *MatchLiveStore) SetViewFromMap(
	ctx context.Context,
	matchID, roundID string,
	userID uuid.UUID,
	fields map[string]any,
	ttl time.Duration,
) (ViewRecord, error) {
	var zero ViewRecord
	if err := ValidateViewFields(fields); err != nil {
		return zero, err
	}
	view := ViewRecord{UserID: userID}
	if v, ok := fields["panorama_id"].(string); ok {
		view.PanoramaID = v
	}
	if v, ok := asViewFloat(fields["heading"]); ok {
		view.Heading = v
	}
	if v, ok := asViewFloat(fields["pitch"]); ok {
		view.Pitch = v
	}
	if v, ok := asViewFloat(fields["zoom"]); ok {
		view.Zoom = v
	}
	return s.SetView(ctx, matchID, roundID, userID, view, ttl)
}

func validateViewRecord(view ViewRecord) error {
	if view.Heading < 0 || view.Heading > 360 {
		return fmt.Errorf("%w: heading", ErrLiveStateInvalid)
	}
	if view.Pitch < -90 || view.Pitch > 90 {
		return fmt.Errorf("%w: pitch", ErrLiveStateInvalid)
	}
	if view.Zoom < 0 || view.Zoom > 10 {
		return fmt.Errorf("%w: zoom", ErrLiveStateInvalid)
	}
	return nil
}

func asViewFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// GetView returns one user's view state when present.
func (s *MatchLiveStore) GetView(ctx context.Context, matchID, roundID string, userID uuid.UUID) (*ViewRecord, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	raw, err := s.client.HGet(ctx, matchplayViewsKey(matchID, roundID), userID.String()).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get view: %w", err)
	}
	var view ViewRecord
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		return nil, fmt.Errorf("decode view: %w", err)
	}
	view.UserID = userID
	return &view, nil
}

// claimCommandScript SETNX-stores a command ack or returns the prior payload.
// KEYS[1]=command key
// ARGV[1]=ack JSON, ARGV[2]=ttl seconds
// Returns: {status, ackJSON} status=claimed|replay
var claimCommandScript = redis.NewScript(`
local key = KEYS[1]
local ack = ARGV[1]
local ttl = tonumber(ARGV[2])
local existing = redis.call('GET', key)
if existing then
  return {'replay', existing}
end
redis.call('SET', key, ack, 'EX', ttl)
return {'claimed', ack}
`)

// StoreCommandAck claims a command id and stores its acknowledgement for replay.
// On first claim, stores ack and returns (ack, false, nil).
// On replay, returns the previously stored ack with replay=true.
func (s *MatchLiveStore) StoreCommandAck(
	ctx context.Context,
	matchID string,
	userID uuid.UUID,
	ack CommandAck,
	ttl time.Duration,
) (CommandAck, bool, error) {
	var zero CommandAck
	if err := s.require(); err != nil {
		return zero, false, err
	}
	if strings.TrimSpace(matchID) == "" || userID == uuid.Nil || strings.TrimSpace(ack.CommandID) == "" {
		return zero, false, fmt.Errorf("%w: command", ErrLiveStateInvalid)
	}
	if ttl <= 0 {
		ttl = DefaultCommandAckTTL
	}
	ttlSec := int(ttl / time.Second)
	if ttlSec < 1 {
		ttlSec = 1
	}
	body, err := json.Marshal(ack)
	if err != nil {
		return zero, false, err
	}
	result, err := claimCommandScript.Run(ctx, s.client, []string{
		matchplayCommandKey(matchID, userID, ack.CommandID),
	}, string(body), ttlSec).Slice()
	if err != nil {
		return zero, false, fmt.Errorf("store command ack: %w", err)
	}
	if len(result) < 2 {
		return zero, false, ErrLiveStateInvalid
	}
	status := asString(result[0])
	var stored CommandAck
	if err := json.Unmarshal([]byte(asString(result[1])), &stored); err != nil {
		return zero, false, fmt.Errorf("decode command ack: %w", err)
	}
	return stored, status == "replay", nil
}

// GetCommandAck returns a previously stored acknowledgement when present.
func (s *MatchLiveStore) GetCommandAck(ctx context.Context, matchID string, userID uuid.UUID, commandID string) (*CommandAck, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	raw, err := s.client.Get(ctx, matchplayCommandKey(matchID, userID, commandID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get command ack: %w", err)
	}
	var ack CommandAck
	if err := json.Unmarshal([]byte(raw), &ack); err != nil {
		return nil, fmt.Errorf("decode command ack: %w", err)
	}
	return &ack, nil
}

// ExpireRoundKeys applies/extends TTL on round-scoped live keys (markers/views/locked).
func (s *MatchLiveStore) ExpireRoundKeys(ctx context.Context, matchID, roundID string, ttl time.Duration) error {
	if err := s.require(); err != nil {
		return err
	}
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(roundID) == "" {
		return fmt.Errorf("%w: match/round", ErrLiveStateInvalid)
	}
	if ttl <= 0 {
		ttl = DefaultRoundLiveTTL
	}
	keys := []string{
		matchplayMarkerKey(matchID, roundID, 1),
		matchplayMarkerKey(matchID, roundID, 2),
		matchplayViewsKey(matchID, roundID),
		matchplayLockedKey(matchID, roundID),
	}
	pipe := s.client.Pipeline()
	for _, k := range keys {
		pipe.PExpire(ctx, k, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("expire round keys: %w", err)
	}
	return nil
}

// DeleteRoundKeys removes disposable round collaboration state after reveal.
func (s *MatchLiveStore) DeleteRoundKeys(ctx context.Context, matchID, roundID string) error {
	if err := s.require(); err != nil {
		return err
	}
	keys := []string{
		matchplayMarkerKey(matchID, roundID, 1),
		matchplayMarkerKey(matchID, roundID, 2),
		matchplayViewsKey(matchID, roundID),
		matchplayLockedKey(matchID, roundID),
	}
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("delete round keys: %w", err)
	}
	return nil
}

func (s *MatchLiveStore) require() error {
	if s == nil || s.client == nil {
		return errors.New("match live redis unavailable")
	}
	return nil
}

func validateMatchRound(matchID, roundID string, userID uuid.UUID) error {
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(roundID) == "" || userID == uuid.Nil {
		return fmt.Errorf("%w: match/round/user", ErrLiveStateInvalid)
	}
	return nil
}
