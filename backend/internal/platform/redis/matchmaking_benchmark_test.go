package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func BenchmarkMatchmakingClaimPairWithQueueLoad(b *testing.B) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/13"
	}
	opt, err := goredis.ParseURL(url)
	if err != nil {
		b.Skipf("invalid REDIS_URL: %v", err)
	}
	client := goredis.NewClient(opt)
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("redis unavailable: %v", err)
	}
	defer func() { _ = client.Close() }()

	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()

	// SC performance fixture: 10,000 queue members with production scan bound of 20.
	const members = 10000
	userIDs := make([]uuid.UUID, members)
	entryIDs := make([]string, members)
	for i := 0; i < members; i++ {
		userIDs[i] = uuid.New()
		entry, err := coord.Join(ctx, userIDs[i], mode, now.Add(time.Duration(i)*time.Millisecond), 5*time.Minute)
		if err != nil {
			b.Fatalf("seed join %d: %v", i, err)
		}
		entryIDs[i] = entry.EntryID
	}
	b.Cleanup(func() {
		for _, id := range userIDs {
			_ = client.Del(ctx, matchmakingPlayerKey(id)).Err()
		}
		_ = client.Del(ctx, matchmakingQueueKey(mode), matchmakingClaimsIndexKey()).Err()
		for _, entryID := range entryIDs {
			_ = client.Del(ctx, "matchmaking:v1:entry:"+entryID).Err()
		}
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Production scan bound is 20 candidates.
		_, err := coord.ClaimPair(ctx, mode, now.Add(time.Minute), 15*time.Second, 20)
		if err != nil {
			b.Fatalf("claim: %v", err)
		}
	}
}
