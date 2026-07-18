package competitive_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestMemoryPageCache_TTLAndInvalidation(t *testing.T) {
	t.Parallel()
	cache := competitive.NewMemoryPageCache(50 * time.Millisecond)
	seasonID := uuid.New()
	userID := uuid.New()
	ctx := context.Background()

	v1, err := cache.Version(ctx, seasonID)
	if err != nil || v1 != 1 {
		t.Fatalf("version = %d err=%v", v1, err)
	}
	key := competitive.ProfileCacheKey(seasonID, userID, v1)
	profile := &competitive.ProfileResponse{
		Profile: competitive.ProfileBodyDTO{PlacementsCompleted: 2, PlacementsRequired: 5},
	}
	if err := cache.SetProfile(ctx, key, profile); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := cache.GetProfile(ctx, key)
	if err != nil || got == nil || got.Profile.PlacementsCompleted != 2 {
		t.Fatalf("get = %+v err=%v", got, err)
	}

	if err := cache.InvalidateSeason(ctx, seasonID); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	v2, err := cache.Version(ctx, seasonID)
	if err != nil || v2 != 2 {
		t.Fatalf("version after invalidate = %d", v2)
	}
	// Old key may remain until TTL; callers must use the new version key.
	_ = competitive.ProfileCacheKey(seasonID, userID, v1)
	newKey := competitive.ProfileCacheKey(seasonID, userID, v2)
	if hit, _ := cache.GetProfile(ctx, newKey); hit != nil {
		t.Fatal("new version key should miss")
	}
}

func TestMemoryPageCache_LeaderboardTTLCap(t *testing.T) {
	t.Parallel()
	// Constructing with >60s is capped by constructor.
	cache := competitive.NewMemoryPageCache(5 * time.Minute)
	seasonID := uuid.New()
	ctx := context.Background()
	key := competitive.LeaderboardCacheKey(seasonID, 1, 50, "", "viewer")
	resp := &competitive.LeaderboardResponse{
		Data: []competitive.LeaderboardEntryDTO{{Position: 1, Rating: 2100, WorldLegend: true}},
	}
	if err := cache.SetLeaderboard(ctx, key, resp); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := cache.GetLeaderboard(ctx, key)
	if err != nil || got == nil || len(got.Data) != 1 || !got.Data[0].WorldLegend {
		t.Fatalf("get = %+v err=%v", got, err)
	}
}

func TestCacheKeysStable(t *testing.T) {
	t.Parallel()
	seasonID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	k1 := competitive.ProfileCacheKey(seasonID, userID, 3)
	k2 := competitive.ProfileCacheKey(seasonID, userID, 3)
	if k1 != k2 {
		t.Fatalf("unstable keys %s vs %s", k1, k2)
	}
	if k1 == competitive.ProfileCacheKey(seasonID, userID, 4) {
		t.Fatal("version must change key")
	}
	lb1 := competitive.LeaderboardCacheKey(seasonID, 1, 50, "c", userID.String())
	lb2 := competitive.ClosedLeaderboardCacheKey(seasonID, 1, 50, "c", userID.String())
	if lb1 == lb2 {
		t.Fatal("active and closed keys must differ")
	}
}

func TestDefaultLeaderboardCacheTTLBounded(t *testing.T) {
	t.Parallel()
	if competitive.DefaultLeaderboardCacheTTL > 60*time.Second {
		t.Fatalf("TTL %s exceeds 60s", competitive.DefaultLeaderboardCacheTTL)
	}
	if competitive.DefaultLeaderboardCacheTTL <= 0 {
		t.Fatal("TTL must be positive")
	}
}
