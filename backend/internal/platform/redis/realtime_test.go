package redis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRealtimeKeyBuilders(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	matchID := "11111111-1111-1111-1111-111111111111"
	partyID := "22222222-2222-2222-2222-222222222222"
	tokenHash := TicketKeyHash("opaque-token-value")

	if got := realtimeTicketKey(tokenHash); got != "realtime:v1:ticket:"+tokenHash {
		t.Fatalf("ticket key = %q", got)
	}
	if strings.Contains(realtimeTicketKey(tokenHash), "opaque-token-value") {
		t.Fatal("ticket key must not embed raw token")
	}
	if got := realtimeChannelVersionKey("match", matchID); got != "realtime:v1:channel:match:"+matchID+":version" {
		t.Fatalf("channel version key = %q", got)
	}
	if got := matchPresenceKey(matchID, userID); got != "matchplay:v1:"+matchID+":presence:"+userID.String() {
		t.Fatalf("match presence key = %q", got)
	}
	if got := matchReconnectKey(matchID, userID); got != "matchplay:v1:"+matchID+":reconnect:"+userID.String() {
		t.Fatalf("match reconnect key = %q", got)
	}
	if got := partyPresenceKey(partyID, userID); got != "party:v1:"+partyID+":presence:"+userID.String() {
		t.Fatalf("party presence key = %q", got)
	}
	if got := partyReconnectKey(partyID, userID); got != "party:v1:"+partyID+":reconnect:"+userID.String() {
		t.Fatalf("party reconnect key = %q", got)
	}
}

func TestRealtimeTicketIssueConsumeBindsUserAndChannel(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	userID := uuid.New()
	channelID := uuid.NewString()
	kind := "match"

	token, err := store.IssueTicket(ctx, userID, kind, channelID, DefaultRealtimeTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("expected opaque token")
	}
	key := realtimeTicketKey(TicketKeyHash(token))
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})

	claims, err := store.ConsumeTicket(ctx, token)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("user id = %s, want %s", claims.UserID, userID)
	}
	if claims.ChannelKind != kind {
		t.Fatalf("channel kind = %q, want %q", claims.ChannelKind, kind)
	}
	if claims.ChannelID != channelID {
		t.Fatalf("channel id = %q, want %q", claims.ChannelID, channelID)
	}
	if claims.ExpiresAt.IsZero() || claims.ExpiresAt.Before(time.Now().UTC().Add(-time.Minute)) {
		t.Fatalf("unexpected expires_at: %v", claims.ExpiresAt)
	}
}

func TestRealtimeTicketReplayRejected(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	userID := uuid.New()
	channelID := uuid.NewString()
	token, err := store.IssueTicket(ctx, userID, "party", channelID, DefaultRealtimeTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	key := realtimeTicketKey(TicketKeyHash(token))
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})

	if _, err := store.ConsumeTicket(ctx, token); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	_, err = store.ConsumeTicket(ctx, token)
	if !errors.Is(err, ErrTicketUsed) {
		t.Fatalf("second consume err = %v, want ErrTicketUsed", err)
	}
}

func TestRealtimeTicketExpiredRejected(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	userID := uuid.New()
	channelID := uuid.NewString()
	// Short logical TTL; Redis key retained long enough for expires_at check via script
	// by writing expires_at in the past after issue.
	token, err := store.IssueTicket(ctx, userID, "match", channelID, 30*time.Second)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	key := realtimeTicketKey(TicketKeyHash(token))
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})

	// Force logical expiry while the key still exists so Consume returns ErrTicketExpired.
	pastMs := time.Now().UTC().Add(-time.Second).UnixMilli()
	if err := client.HSet(ctx, key, ticketFieldExpiresAt, pastMs).Err(); err != nil {
		t.Fatalf("force expiry: %v", err)
	}

	_, err = store.ConsumeTicket(ctx, token)
	if !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("consume err = %v, want ErrTicketExpired", err)
	}

	// Ticket must be deleted on expired consume (no replay of expired payload).
	exists, err := client.Exists(ctx, key).Result()
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != 0 {
		t.Fatal("expired ticket key should be deleted after consume")
	}
}

func TestRealtimeTicketWrongAndMissingRejected(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	_, err := store.ConsumeTicket(ctx, "")
	if !errors.Is(err, ErrTicketInvalid) {
		t.Fatalf("empty token err = %v, want ErrTicketInvalid", err)
	}

	_, err = store.ConsumeTicket(ctx, "   ")
	if !errors.Is(err, ErrTicketInvalid) {
		t.Fatalf("whitespace token err = %v, want ErrTicketInvalid", err)
	}

	_, err = store.ConsumeTicket(ctx, "never-issued-opaque-token-value")
	if !errors.Is(err, ErrTicketUsed) {
		t.Fatalf("missing token err = %v, want ErrTicketUsed", err)
	}
}

func TestRealtimeTicketRedisKeysUseHashNotRawToken(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	userID := uuid.New()
	channelID := uuid.NewString()
	token, err := store.IssueTicket(ctx, userID, "match", channelID, DefaultRealtimeTicketTTL)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	hash := TicketKeyHash(token)
	hashedKey := realtimeTicketKey(hash)
	t.Cleanup(func() {
		_ = client.Del(ctx, hashedKey).Err()
	})

	exists, err := client.Exists(ctx, hashedKey).Result()
	if err != nil {
		t.Fatalf("exists hashed: %v", err)
	}
	if exists != 1 {
		t.Fatalf("expected hashed ticket key %q to exist", hashedKey)
	}

	// Raw token must never be used as a Redis key segment.
	rawKey := "realtime:v1:ticket:" + token
	if rawKey == hashedKey {
		t.Fatal("hash collision with raw token encoding; regenerate assertion inputs")
	}
	rawExists, err := client.Exists(ctx, rawKey).Result()
	if err != nil {
		t.Fatalf("exists raw: %v", err)
	}
	if rawExists != 0 {
		t.Fatal("redis must not store a key containing the raw opaque token")
	}

	// Hash fields must not echo the raw token.
	vals, err := client.HGetAll(ctx, hashedKey).Result()
	if err != nil {
		t.Fatalf("hgetall: %v", err)
	}
	for field, value := range vals {
		if strings.Contains(value, token) || strings.Contains(field, token) {
			t.Fatalf("ticket payload embeds raw token in %q=%q", field, value)
		}
	}
	if vals[ticketFieldUserID] != userID.String() {
		t.Fatalf("user_id field = %q", vals[ticketFieldUserID])
	}
	if vals[ticketFieldChannelKind] != "match" {
		t.Fatalf("channel_kind = %q", vals[ticketFieldChannelKind])
	}
	if vals[ticketFieldChannelID] != channelID {
		t.Fatalf("channel_id = %q", vals[ticketFieldChannelID])
	}
}

func TestRealtimeTicketDefaultTTL(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	userID := uuid.New()
	channelID := uuid.NewString()
	token, err := store.IssueTicket(ctx, userID, "party", channelID, 0)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	key := realtimeTicketKey(TicketKeyHash(token))
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})

	ttl, err := client.PTTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("pttl: %v", err)
	}
	// Default 30s; allow generous bounds for slow CI.
	if ttl < 20*time.Second || ttl > 31*time.Second {
		t.Fatalf("ticket TTL = %v, want ~30s", ttl)
	}
}

func TestRealtimeChannelVersionIncrAndGet(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)
	channelID := uuid.NewString()
	key := realtimeChannelVersionKey("match", channelID)
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})
	_ = client.Del(ctx, key).Err()

	v0, err := store.GetChannelVersion(ctx, "match", channelID)
	if err != nil || v0 != 0 {
		t.Fatalf("initial version = %d err=%v", v0, err)
	}
	v1, err := store.IncrChannelVersion(ctx, "match", channelID)
	if err != nil || v1 != 1 {
		t.Fatalf("incr1 = %d err=%v", v1, err)
	}
	v2, err := store.IncrChannelVersion(ctx, "match", channelID)
	if err != nil || v2 != 2 {
		t.Fatalf("incr2 = %d err=%v", v2, err)
	}
	got, err := store.GetChannelVersion(ctx, "match", channelID)
	if err != nil || got != 2 {
		t.Fatalf("get = %d err=%v", got, err)
	}
}

func TestRealtimePresenceAndReconnect(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)
	userID := uuid.New()
	matchID := uuid.NewString()
	partyID := uuid.NewString()

	t.Cleanup(func() {
		_ = client.Del(ctx,
			matchPresenceKey(matchID, userID),
			matchReconnectKey(matchID, userID),
			partyPresenceKey(partyID, userID),
			partyReconnectKey(partyID, userID),
		).Err()
	})

	if err := store.SetPresence(ctx, "match", matchID, userID, "connected", 90*time.Second); err != nil {
		t.Fatalf("set match presence: %v", err)
	}
	status, err := store.GetPresence(ctx, "match", matchID, userID)
	if err != nil || status != "connected" {
		t.Fatalf("match presence = %q err=%v", status, err)
	}
	if err := store.DeletePresence(ctx, "match", matchID, userID); err != nil {
		t.Fatalf("delete match presence: %v", err)
	}
	status, err = store.GetPresence(ctx, "match", matchID, userID)
	if err != nil || status != "" {
		t.Fatalf("after delete presence = %q err=%v", status, err)
	}

	if err := store.SetPresence(ctx, "party", partyID, userID, "connected", 90*time.Second); err != nil {
		t.Fatalf("set party presence: %v", err)
	}
	status, err = store.GetPresence(ctx, "party", partyID, userID)
	if err != nil || status != "connected" {
		t.Fatalf("party presence = %q err=%v", status, err)
	}

	if err := store.SetReconnectWindow(ctx, "match", matchID, userID, 42, 90*time.Second); err != nil {
		t.Fatalf("set reconnect: %v", err)
	}
	version, ok, err := store.GetReconnectWindow(ctx, "match", matchID, userID)
	if err != nil || !ok || version != 42 {
		t.Fatalf("reconnect = %d ok=%v err=%v", version, ok, err)
	}
	if err := store.DeleteReconnectWindow(ctx, "match", matchID, userID); err != nil {
		t.Fatalf("delete reconnect: %v", err)
	}
	_, ok, err = store.GetReconnectWindow(ctx, "match", matchID, userID)
	if err != nil || ok {
		t.Fatalf("reconnect should be absent after delete, ok=%v err=%v", ok, err)
	}
}

func TestRealtimePresenceCountsMultipleSocketsBeforeReconnect(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)
	userID, matchID := uuid.New(), uuid.NewString()
	t.Cleanup(func() {
		countKey, _ := presenceConnectionsKey("match", matchID, userID)
		_ = client.Del(ctx, countKey, matchPresenceKey(matchID, userID), matchReconnectKey(matchID, userID)).Err()
	})

	if count, err := store.AcquirePresence(ctx, "match", matchID, userID, time.Minute); err != nil || count != 1 {
		t.Fatalf("first acquire = %d err=%v", count, err)
	}
	if count, err := store.AcquirePresence(ctx, "match", matchID, userID, time.Minute); err != nil || count != 2 {
		t.Fatalf("second acquire = %d err=%v", count, err)
	}
	if count, err := store.RefreshPresence(ctx, "match", matchID, userID, time.Minute); err != nil || count != 2 {
		t.Fatalf("refresh changed socket count: %d err=%v", count, err)
	}
	if remaining, err := store.ReleasePresence(ctx, "match", matchID, userID, 7, time.Minute); err != nil || remaining != 1 {
		t.Fatalf("first release = %d err=%v", remaining, err)
	}
	if _, ok, err := store.GetReconnectWindow(ctx, "match", matchID, userID); err != nil || ok {
		t.Fatalf("reconnect created while another socket was live: ok=%v err=%v", ok, err)
	}
	if remaining, err := store.ReleasePresence(ctx, "match", matchID, userID, 9, time.Minute); err != nil || remaining != 0 {
		t.Fatalf("final release = %d err=%v", remaining, err)
	}
	if version, ok, err := store.GetReconnectWindow(ctx, "match", matchID, userID); err != nil || !ok || version != 9 {
		t.Fatalf("reconnect = %d ok=%v err=%v", version, ok, err)
	}
}

func TestRealtimeTicketIssueValidation(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewRealtimeStore(client)

	_, err := store.IssueTicket(ctx, uuid.Nil, "match", uuid.NewString(), time.Second)
	if !errors.Is(err, ErrTicketInvalid) {
		t.Fatalf("nil user err = %v, want ErrTicketInvalid", err)
	}
	_, err = store.IssueTicket(ctx, uuid.New(), "lobby", uuid.NewString(), time.Second)
	if err == nil {
		t.Fatal("expected error for bad channel kind")
	}
	_, err = store.IssueTicket(ctx, uuid.New(), "match", "", time.Second)
	if !errors.Is(err, ErrTicketInvalid) {
		t.Fatalf("empty channel err = %v, want ErrTicketInvalid", err)
	}
}
