package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

func TestRateLimiterAllowWithinLimit(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	key := fmt.Sprintf("rate:test:%d", time.Now().UnixNano())
	limiter := redisplatform.NewRateLimiter(client)

	for i := 0; i < 3; i++ {
		allowed, count, err := limiter.Allow(ctx, key, 3, time.Minute)
		if err != nil {
			t.Fatalf("allow failed: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		if count != i+1 {
			t.Fatalf("count = %d, want %d", count, i+1)
		}
	}

	allowed, count, err := limiter.Allow(ctx, key, 3, time.Minute)
	if err != nil {
		t.Fatalf("allow failed: %v", err)
	}
	if allowed {
		t.Fatal("request over limit should be denied")
	}
	if count != 4 {
		t.Fatalf("count = %d, want 4", count)
	}
}

func TestRateLimiterWindowSlides(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	key := fmt.Sprintf("rate:slide:%d", time.Now().UnixNano())
	limiter := redisplatform.NewRateLimiter(client)

	if _, _, err := limiter.Allow(ctx, key, 1, time.Millisecond*50); err != nil {
		t.Fatalf("allow failed: %v", err)
	}

	allowed, _, err := limiter.Allow(ctx, key, 1, time.Millisecond*50)
	if err != nil {
		t.Fatalf("allow failed: %v", err)
	}
	if allowed {
		t.Fatal("second request should be denied inside window")
	}

	time.Sleep(time.Millisecond * 60)

	allowed, _, err = limiter.Allow(ctx, key, 1, time.Millisecond*50)
	if err != nil {
		t.Fatalf("allow failed: %v", err)
	}
	if !allowed {
		t.Fatal("request after window slide should be allowed")
	}
}
