package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/auth"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

func TestRedisSessionStoreCreateAndGet(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()
	ctx := context.Background()
	store := auth.NewRedisSessionStore(client)

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	session := &auth.RefreshSession{
		UserID:    userID,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	hash := "test-hash-create"

	if err := store.Create(ctx, hash, session, time.Minute); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.UserID != userID {
		t.Fatalf("user id = %v, want %v", got.UserID, userID)
	}
	if got.Role != "user" {
		t.Fatalf("role = %v, want user", got.Role)
	}
}

func TestRedisSessionStoreRevoke(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()
	ctx := context.Background()
	store := auth.NewRedisSessionStore(client)

	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	session := &auth.RefreshSession{
		UserID:    userID,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	hash := "test-hash-revoke"

	if err := store.Create(ctx, hash, session, time.Minute); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := store.Revoke(ctx, hash); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	got, err := store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil session after revoke, got %v", got)
	}
}

func TestRedisSessionStoreRevokeAll(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()
	ctx := context.Background()
	store := auth.NewRedisSessionStore(client)

	userID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	base := &auth.RefreshSession{
		UserID:    userID,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	for _, h := range []string{"hash-a", "hash-b"} {
		if err := store.Create(ctx, h, base, time.Minute); err != nil {
			t.Fatalf("create failed: %v", err)
		}
	}

	if err := store.RevokeAll(ctx, userID); err != nil {
		t.Fatalf("revoke all failed: %v", err)
	}

	for _, h := range []string{"hash-a", "hash-b"} {
		got, err := store.Get(ctx, h)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if got != nil {
			t.Fatalf("expected session %s revoked", h)
		}
	}
}

func TestRedisSessionStoreReuseDetection(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()
	ctx := context.Background()
	store := auth.NewRedisSessionStore(client)

	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	session := &auth.RefreshSession{
		UserID:    userID,
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	hash := "test-hash-reuse"

	if err := store.Create(ctx, hash, session, time.Minute); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := store.Revoke(ctx, hash); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	if err := store.MarkRevoked(ctx, hash, userID, time.Minute); err != nil {
		t.Fatalf("mark revoked failed: %v", err)
	}

	revokedUserID, reused, err := store.RevokedUserID(ctx, hash)
	if err != nil {
		t.Fatalf("is revoked failed: %v", err)
	}
	if !reused {
		t.Fatal("expected revoked token to be detected as reuse")
	}
	if revokedUserID != userID {
		t.Fatalf("revoked user id = %v, want %v", revokedUserID, userID)
	}
}
