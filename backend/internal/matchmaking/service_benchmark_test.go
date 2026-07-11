package matchmaking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
)

func BenchmarkJoinQueueIdempotent(b *testing.B) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	queue := newMemoryQueue()
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithClock(stubClock{now: time.Now().UTC()})
	sess := userSession(userID)
	req := matchmaking.JoinQueueRequest{Mode: matchmaking.ModeRankedStandard}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.JoinQueue(context.Background(), sess, req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetStatusSearching(b *testing.B) {
	userID := uuid.New()
	store := newMemoryStore()
	store.users[userID] = &matchmaking.ActiveUser{ID: userID, Status: "active"}
	queue := newMemoryQueue()
	now := time.Now().UTC()
	queue.entries[userID] = &matchmaking.QueueEntry{
		EntryID:          uuid.NewString(),
		UserID:           userID,
		Mode:             matchmaking.ModeRankedStandard,
		State:            matchmaking.QueueStateSearching,
		EnqueuedAtMs:     now.UnixMilli(),
		LeaseExpiresAtMs: now.Add(30 * time.Second).UnixMilli(),
	}
	svc := matchmaking.NewService(store, queue, testConfig(), nil, nil).WithClock(stubClock{now: now})
	sess := userSession(userID)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetStatus(context.Background(), sess); err != nil {
			b.Fatal(err)
		}
	}
}
