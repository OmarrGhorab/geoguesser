package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCommandIdempotencyKeyScoped(t *testing.T) {
	a := CommandIdempotencyKey("party.invite", "user-1", "key-1")
	b := CommandIdempotencyKey("party.invite", "user-2", "key-1")
	c := CommandIdempotencyKey("party.invite", "user-1", "key-2")
	d := CommandIdempotencyKey("queue.join", "user-1", "key-1")
	if a == b || a == c || a == d {
		t.Fatalf("keys should differ by caller/scope/idem key: %q %q %q %q", a, b, c, d)
	}
	if a != CommandIdempotencyKey("party.invite", "user-1", "key-1") {
		t.Fatal("key should be stable")
	}
	if a[:len(commandIdempotencyKeyPrefix)] != commandIdempotencyKeyPrefix {
		t.Fatalf("prefix = %q", a[:len(commandIdempotencyKeyPrefix)])
	}
}

func TestCommandIdempotencySameBodyReplay(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewCommandIdempotencyStore(client)

	scope := "party.create"
	caller := uuid.NewString()
	idemKey := "retry-1"
	bodyHash := "hash-abc"
	ttl := 30 * time.Second
	key := CommandIdempotencyKey(scope, caller, idemKey)
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})
	_ = client.Del(ctx, key).Err()

	first, err := store.Begin(ctx, scope, caller, idemKey, bodyHash, ttl)
	if err != nil {
		t.Fatalf("first begin: %v", err)
	}
	if first.Hit || first.Conflict {
		t.Fatalf("first begin unexpected: %+v", first)
	}

	response := []byte(`{"party_id":"p1"}`)
	if err := store.Complete(ctx, scope, caller, idemKey, bodyHash, response, ttl); err != nil {
		t.Fatalf("complete: %v", err)
	}

	replay, err := store.Begin(ctx, scope, caller, idemKey, bodyHash, ttl)
	if err != nil {
		t.Fatalf("replay begin: %v", err)
	}
	if !replay.Hit || replay.Conflict {
		t.Fatalf("replay unexpected: %+v", replay)
	}
	if string(replay.Response) != string(response) {
		t.Fatalf("replay response = %q, want %q", replay.Response, response)
	}
}

func TestCommandIdempotencyConflictingBody(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewCommandIdempotencyStore(client)

	scope := "message.send"
	caller := uuid.NewString()
	idemKey := "msg-1"
	ttl := 30 * time.Second
	key := CommandIdempotencyKey(scope, caller, idemKey)
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})
	_ = client.Del(ctx, key).Err()

	if _, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", ttl); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := store.Complete(ctx, scope, caller, idemKey, "hash-a", []byte(`{"ok":true}`), ttl); err != nil {
		t.Fatalf("complete: %v", err)
	}

	conflict, err := store.Begin(ctx, scope, caller, idemKey, "hash-b", ttl)
	if err != nil {
		t.Fatalf("conflict begin: %v", err)
	}
	if !conflict.Conflict || conflict.Hit {
		t.Fatalf("expected conflict, got %+v", conflict)
	}

	// Completing with a different hash after a done record also conflicts.
	if err := store.Complete(ctx, scope, caller, idemKey, "hash-b", []byte(`{"ok":false}`), ttl); !errors.Is(err, ErrCommandIdempotencyConflict) {
		t.Fatalf("complete conflict err = %v", err)
	}
}

func TestCommandIdempotencyInFlight(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewCommandIdempotencyStore(client)

	scope := "queue.join"
	caller := uuid.NewString()
	idemKey := "join-1"
	ttl := 30 * time.Second
	key := CommandIdempotencyKey(scope, caller, idemKey)
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})
	_ = client.Del(ctx, key).Err()

	first, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", ttl)
	if err != nil || first.Hit || first.Conflict {
		t.Fatalf("first begin: res=%+v err=%v", first, err)
	}
	second, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", ttl)
	if err != nil {
		t.Fatalf("second begin: %v", err)
	}
	if !second.Conflict || !second.InFlight {
		t.Fatalf("expected in-flight conflict, got %+v", second)
	}

	// Release pending claim allows retry.
	if err := store.Release(ctx, scope, caller, idemKey); err != nil {
		t.Fatalf("release: %v", err)
	}
	retry, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", ttl)
	if err != nil || retry.Hit || retry.Conflict {
		t.Fatalf("retry after release: res=%+v err=%v", retry, err)
	}
}

func TestCommandIdempotencyTTLExpiry(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewCommandIdempotencyStore(client)

	scope := "report.create"
	caller := uuid.NewString()
	idemKey := "report-1"
	ttl := 1 * time.Second
	key := CommandIdempotencyKey(scope, caller, idemKey)
	t.Cleanup(func() {
		_ = client.Del(ctx, key).Err()
	})
	_ = client.Del(ctx, key).Err()

	if _, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", ttl); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := store.Complete(ctx, scope, caller, idemKey, "hash-a", []byte(`{"id":1}`), ttl); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Wait for TTL to elapse (Redis second-granularity EXPIRE).
	deadline := time.Now().Add(5 * time.Second)
	for {
		n, err := client.Exists(ctx, key).Result()
		if err != nil {
			t.Fatalf("exists: %v", err)
		}
		if n == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("key did not expire")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// After expiry, same key+body claims again (not a hit).
	again, err := store.Begin(ctx, scope, caller, idemKey, "hash-a", 30*time.Second)
	if err != nil {
		t.Fatalf("begin after expiry: %v", err)
	}
	if again.Hit || again.Conflict {
		t.Fatalf("expected fresh claim after TTL, got %+v", again)
	}
}

func TestCommandIdempotencyCallerScoping(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	store := NewCommandIdempotencyStore(client)

	scope := "leave.match"
	idemKey := "shared-key"
	ttl := 30 * time.Second
	userA := uuid.NewString()
	userB := uuid.NewString()
	keyA := CommandIdempotencyKey(scope, userA, idemKey)
	keyB := CommandIdempotencyKey(scope, userB, idemKey)
	t.Cleanup(func() {
		_ = client.Del(ctx, keyA, keyB).Err()
	})
	_ = client.Del(ctx, keyA, keyB).Err()

	if _, err := store.Begin(ctx, scope, userA, idemKey, "hash-a", ttl); err != nil {
		t.Fatalf("userA begin: %v", err)
	}
	if err := store.Complete(ctx, scope, userA, idemKey, "hash-a", []byte(`{"user":"a"}`), ttl); err != nil {
		t.Fatalf("userA complete: %v", err)
	}

	// Same idempotency key, different caller: independent claim, no shared replay.
	b, err := store.Begin(ctx, scope, userB, idemKey, "hash-a", ttl)
	if err != nil {
		t.Fatalf("userB begin: %v", err)
	}
	if b.Hit || b.Conflict {
		t.Fatalf("userB should not see userA record: %+v", b)
	}
	if err := store.Complete(ctx, scope, userB, idemKey, "hash-a", []byte(`{"user":"b"}`), ttl); err != nil {
		t.Fatalf("userB complete: %v", err)
	}

	replayA, err := store.Begin(ctx, scope, userA, idemKey, "hash-a", ttl)
	if err != nil || !replayA.Hit || string(replayA.Response) != `{"user":"a"}` {
		t.Fatalf("userA replay = %+v err=%v", replayA, err)
	}
	replayB, err := store.Begin(ctx, scope, userB, idemKey, "hash-a", ttl)
	if err != nil || !replayB.Hit || string(replayB.Response) != `{"user":"b"}` {
		t.Fatalf("userB replay = %+v err=%v", replayB, err)
	}
}
