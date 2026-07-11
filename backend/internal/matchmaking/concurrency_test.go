package matchmaking_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testRedisClient(t *testing.T) *goredis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/14"
	}
	opt, err := goredis.ParseURL(url)
	if err != nil {
		t.Skipf("invalid REDIS_URL: %v", err)
	}
	client := goredis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestConcurrentFormationRace_ExactlyOneMatch(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping concurrency integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Skip("postgres unavailable")
	}
	if !db.Migrator().HasTable("matches") {
		t.Skip("matches table missing; run migration 00014")
	}

	client := testRedisClient(t)
	coord := redisplatform.NewMatchmakingCoordinator(client)
	queue := matchmaking.NewRedisQueueAdapter(coord)
	repo := matchmaking.NewRepository(db)

	mapID, userA, userB, locationIDs := seedFormationFixtures(t, db)
	mode := matchmaking.ModeRankedStandard
	now := time.Now().UTC()
	ctx := context.Background()

	t.Cleanup(func() {
		_ = client.Del(ctx,
			"matchmaking:v1:player:"+userA.String(),
			"matchmaking:v1:player:"+userB.String(),
			"matchmaking:v1:queue:"+mode,
			"matchmaking:v1:claims",
		).Err()
	})

	if _, err := queue.Join(ctx, userA, mode, now, 30*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := queue.Join(ctx, userB, mode, now.Add(time.Millisecond), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}

	locs := &fixedLocations{ids: locationIDs}
	cfg := matchmaking.Config{
		DefaultMapID:       mapID,
		QueueLease:         30 * time.Second,
		ClaimTTL:           15 * time.Second,
		StartDelay:         5 * time.Second,
		RoundCount:         5,
		TimerSeconds:       60,
		CandidateScanLimit: 20,
	}

	const workers = 8
	var wg sync.WaitGroup
	results := make(chan *matchmaking.StatusResponse, workers)
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := matchmaking.NewService(repo, queue, cfg, nil, nil).WithLocations(locs)
			// Drive formation via status (request-driven) for both users.
			resp, err := svc.GetStatus(ctx, userSession(userA))
			if err != nil {
				errs <- err
				return
			}
			results <- resp
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("worker error: %v", err)
		}
	}

	var matchIDs = map[uuid.UUID]struct{}{}
	for resp := range results {
		if resp != nil && resp.Status == matchmaking.PublicStatusMatched && resp.Match != nil {
			matchIDs[resp.Match.MatchID] = struct{}{}
		}
	}

	// Durable store must contain exactly one match for these two users.
	asgA, err := repo.FindActiveAssignment(ctx, userA)
	if err != nil || asgA == nil {
		t.Fatalf("assignment A missing: %+v err=%v", asgA, err)
	}
	asgB, err := repo.FindActiveAssignment(ctx, userB)
	if err != nil || asgB == nil {
		t.Fatalf("assignment B missing: %+v err=%v", asgB, err)
	}
	if asgA.MatchID != asgB.MatchID || asgA.GameID != asgB.GameID {
		t.Fatalf("players assigned to different matches: A=%+v B=%+v", asgA, asgB)
	}
	if len(matchIDs) > 1 {
		t.Fatalf("workers observed multiple match IDs: %v", matchIDs)
	}

	var matchCount int64
	if err := db.Table("matches").Where("id = ?", asgA.MatchID).Count(&matchCount).Error; err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if matchCount != 1 {
		t.Fatalf("match rows = %d, want 1", matchCount)
	}
}

type fixedLocations struct {
	ids []uuid.UUID
}

func (f *fixedLocations) SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]uuid.UUID, error) {
	if len(f.ids) < count {
		return f.ids, nil
	}
	return f.ids[:count], nil
}

func TestCrashWindow_FormationKeyReplayAfterDurableCommit(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if !db.Migrator().HasTable("matches") {
		t.Skip("matches table missing")
	}
	repo := matchmaking.NewRepository(db)
	mapID, userA, userB, locationIDs := seedFormationFixtures(t, db)
	formationKey := "crash-" + uuid.NewString()
	matchedAt := time.Now().UTC()

	first, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: formationKey,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt,
	})
	if err != nil {
		t.Fatalf("first formation: %v", err)
	}

	// Simulate retry after durable commit / before Redis finalize: formation key replay.
	second, err := repo.CreateFormationBundle(t.Context(), matchmaking.FormationInput{
		FormationKey: formationKey,
		Mode:         matchmaking.ModeRankedStandard,
		MapID:        mapID,
		RoundCount:   5,
		TimerSeconds: 60,
		StartDelay:   5 * time.Second,
		UserIDs:      [2]uuid.UUID{userA, userB},
		LocationIDs:  locationIDs,
		MatchedAt:    matchedAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if second.Match.ID != first.Match.ID || second.Match.GameID != first.Match.GameID {
		t.Fatalf("replay diverged: %+v vs %+v", first.Match, second.Match)
	}

	// Recovery via formation key lookup.
	found, err := repo.FindMatchByFormationKey(t.Context(), formationKey)
	if err != nil || found == nil || found.ID != first.Match.ID {
		t.Fatalf("formation key recovery failed: %+v err=%v", found, err)
	}
}
