package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *goredis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/15"
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
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func TestMatchmakingKeyBuilders(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if got := matchmakingQueueKey("ranked_standard"); got != "matchmaking:v1:queue:ranked_standard" {
		t.Fatalf("queue key = %q", got)
	}
	if got := matchmakingPlayerKey(userID); got != "matchmaking:v1:player:00000000-0000-0000-0000-000000000001" {
		t.Fatalf("player key = %q", got)
	}
	if got := matchmakingClaimKey("claim-1"); got != "matchmaking:v1:claim:claim-1" {
		t.Fatalf("claim key = %q", got)
	}
	if got := matchmakingClaimsIndexKey(); got != "matchmaking:v1:claims" {
		t.Fatalf("claims index key = %q", got)
	}
}

func TestMatchmakingJoinIdempotentAndLease(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	userID := uuid.New()
	mode := "ranked_standard"
	now := time.Now().UTC()

	// Isolate keys for this user.
	t.Cleanup(func() {
		_ = client.Del(ctx, matchmakingPlayerKey(userID), matchmakingQueueKey(mode)).Err()
	})
	_ = client.Del(ctx, matchmakingPlayerKey(userID)).Err()

	first, err := coord.Join(ctx, userID, mode, now, 30*time.Second)
	if err != nil {
		t.Fatalf("first join: %v", err)
	}
	if first.State != "searching" || first.Mode != mode {
		t.Fatalf("first entry unexpected: %+v", first)
	}

	second, err := coord.Join(ctx, userID, mode, now.Add(5*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("second join: %v", err)
	}
	if second.EntryID != first.EntryID {
		t.Fatalf("entry id changed on duplicate join: %s vs %s", first.EntryID, second.EntryID)
	}
	if second.EnqueuedAtMs != first.EnqueuedAtMs {
		t.Fatalf("priority score changed: %d vs %d", first.EnqueuedAtMs, second.EnqueuedAtMs)
	}

	renewed, err := coord.RenewLease(ctx, userID, 30*time.Second, now.Add(10*time.Second))
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed == nil {
		t.Fatal("expected renewed entry")
	}
	if renewed.EnqueuedAtMs != first.EnqueuedAtMs {
		t.Fatalf("renew changed priority")
	}
	if renewed.LeaseExpiresAtMs <= first.LeaseExpiresAtMs {
		t.Fatalf("lease not extended")
	}

	if err := coord.Leave(ctx, userID); err != nil {
		t.Fatalf("leave: %v", err)
	}
	entry, err := coord.GetEntry(ctx, userID)
	if err != nil {
		t.Fatalf("get after leave: %v", err)
	}
	if entry != nil {
		t.Fatalf("expected nil after leave, got %+v", entry)
	}
}

func TestMatchmakingCrossModeUniqueness(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	userID := uuid.New()
	now := time.Now().UTC()
	t.Cleanup(func() {
		_ = client.Del(ctx, matchmakingPlayerKey(userID), matchmakingQueueKey("ranked_standard"), matchmakingQueueKey("other_mode")).Err()
	})
	_ = client.Del(ctx, matchmakingPlayerKey(userID)).Err()

	first, err := coord.Join(ctx, userID, "ranked_standard", now, 30*time.Second)
	if err != nil {
		t.Fatalf("join ranked_standard: %v", err)
	}

	// Joining a different mode while a valid entry exists for another mode should
	// replace only after cleanup in the script when modes differ.
	// Spec requires one player pointer across all modes — existing valid entry
	// with a different mode is treated as stale and replaced.
	second, err := coord.Join(ctx, userID, "other_mode", now.Add(time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("join other_mode: %v", err)
	}
	if second.Mode != "other_mode" {
		t.Fatalf("mode = %q", second.Mode)
	}
	if second.EntryID == first.EntryID {
		t.Fatalf("expected new entry after mode switch cleanup")
	}
}

func TestMatchmakingClaimPairExactTwoAndRace(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB, userC := uuid.New(), uuid.New(), uuid.New()

	cleanupKeys := []string{
		matchmakingPlayerKey(userA), matchmakingPlayerKey(userB), matchmakingPlayerKey(userC),
		matchmakingQueueKey(mode), matchmakingClaimsIndexKey(),
	}
	t.Cleanup(func() {
		for _, k := range cleanupKeys {
			_ = client.Del(ctx, k).Err()
		}
	})
	for _, k := range cleanupKeys {
		_ = client.Del(ctx, k).Err()
	}

	// One player alone cannot form a claim.
	if _, err := coord.Join(ctx, userA, mode, now, 30*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	claim, err := coord.ClaimPair(ctx, mode, now.Add(time.Second), 15*time.Second, 20)
	if err != nil {
		t.Fatalf("claim alone: %v", err)
	}
	if claim != nil {
		t.Fatalf("expected nil claim with single candidate, got %+v", claim)
	}

	// Two oldest players are claimed exactly once.
	if _, err := coord.Join(ctx, userB, mode, now.Add(2*time.Second), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}
	if _, err := coord.Join(ctx, userC, mode, now.Add(3*time.Second), 30*time.Second); err != nil {
		t.Fatalf("join C: %v", err)
	}

	firstClaim, err := coord.ClaimPair(ctx, mode, now.Add(4*time.Second), 15*time.Second, 20)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if firstClaim == nil {
		t.Fatal("expected claim for two players")
	}
	if firstClaim.UserIDA != userA && firstClaim.UserIDB != userA {
		t.Fatalf("oldest player A missing from claim: %+v", firstClaim)
	}
	if firstClaim.UserIDA != userB && firstClaim.UserIDB != userB {
		t.Fatalf("second-oldest player B missing from claim: %+v", firstClaim)
	}
	if firstClaim.FormationKey == "" || firstClaim.ClaimID == "" {
		t.Fatalf("missing claim identity: %+v", firstClaim)
	}

	// Concurrent second claim must not re-claim the same pair.
	secondClaim, err := coord.ClaimPair(ctx, mode, now.Add(5*time.Second), 15*time.Second, 20)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if secondClaim != nil {
		t.Fatalf("expected no second claim with only one remaining player, got %+v", secondClaim)
	}

	// Remaining player C still searching.
	entryC, err := coord.GetEntry(ctx, userC)
	if err != nil || entryC == nil || entryC.State != "searching" {
		t.Fatalf("player C state = %+v err=%v", entryC, err)
	}

	// Finalize clears claimed players.
	if err := coord.FinalizeClaim(ctx, firstClaim); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	for _, u := range []uuid.UUID{userA, userB} {
		entry, err := coord.GetEntry(ctx, u)
		if err != nil {
			t.Fatalf("get %s: %v", u, err)
		}
		if entry != nil {
			t.Fatalf("expected cleared player after finalize, got %+v", entry)
		}
	}
}

func TestMatchmakingClaimPrunesStaleAndRejectsExpired(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	staleUser, freshA, freshB := uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() {
		_ = client.Del(ctx,
			matchmakingPlayerKey(staleUser), matchmakingPlayerKey(freshA), matchmakingPlayerKey(freshB),
			matchmakingQueueKey(mode), matchmakingClaimsIndexKey(),
		).Err()
	})

	// Stale: join then expire lease by advancing claim time past lease.
	if _, err := coord.Join(ctx, staleUser, mode, now, 5*time.Second); err != nil {
		t.Fatalf("join stale: %v", err)
	}
	if _, err := coord.Join(ctx, freshA, mode, now.Add(time.Second), 60*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.Join(ctx, freshB, mode, now.Add(2*time.Second), 60*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}

	claim, err := coord.ClaimPair(ctx, mode, now.Add(10*time.Second), 15*time.Second, 20)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claim == nil {
		t.Fatal("expected claim of two fresh players")
	}
	if claim.UserIDA == staleUser || claim.UserIDB == staleUser {
		t.Fatalf("stale user claimed: %+v", claim)
	}
}

func TestMatchmakingReconcileExpiredClaims(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	t.Cleanup(func() {
		_ = client.Del(ctx, matchmakingPlayerKey(userA), matchmakingPlayerKey(userB), matchmakingQueueKey(mode), matchmakingClaimsIndexKey()).Err()
	})

	if _, err := coord.Join(ctx, userA, mode, now, 30*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.Join(ctx, userB, mode, now.Add(time.Second), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}
	// Claim with very short TTL so recover_after is in the past relative to later now.
	claim, err := coord.ClaimPair(ctx, mode, now.Add(2*time.Second), time.Millisecond, 20)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}
	ids, err := coord.ListExpiredClaims(ctx, now.Add(5*time.Second), 20)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(ids) < 1 {
		t.Fatalf("expired claims = %d, want >=1", len(ids))
	}
	// Low-level release requeues both; service layer adds durable-first checks.
	if err := coord.ReleaseClaim(ctx, claim, true, true, 30*time.Second); err != nil {
		t.Fatalf("release: %v", err)
	}
	entryA, err := coord.GetEntry(ctx, userA)
	if err != nil || entryA == nil || entryA.State != "searching" {
		t.Fatalf("A after reconcile: %+v err=%v", entryA, err)
	}
}

func TestMatchmakingReleaseClaimSelectiveRequeue(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	t.Cleanup(func() {
		_ = client.Del(ctx, matchmakingPlayerKey(userA), matchmakingPlayerKey(userB), matchmakingQueueKey(mode), matchmakingClaimsIndexKey()).Err()
	})

	if _, err := coord.Join(ctx, userA, mode, now, 30*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.Join(ctx, userB, mode, now.Add(time.Second), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}
	claim, err := coord.ClaimPair(ctx, mode, now.Add(2*time.Second), 15*time.Second, 20)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}
	// Requeue only A; discard B.
	if err := coord.ReleaseClaim(ctx, claim, true, false, 30*time.Second); err != nil {
		t.Fatalf("release: %v", err)
	}
	entryA, err := coord.GetEntry(ctx, userA)
	if err != nil || entryA == nil || entryA.State != "searching" {
		t.Fatalf("A should be searching: %+v err=%v", entryA, err)
	}
	entryB, err := coord.GetEntry(ctx, userB)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	if entryB != nil {
		t.Fatalf("B should be discarded, got %+v", entryB)
	}
}

func TestMatchmakingReleaseClaimRequeuesOriginalPriority(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	t.Cleanup(func() {
		_ = client.Del(ctx, matchmakingPlayerKey(userA), matchmakingPlayerKey(userB), matchmakingQueueKey(mode), matchmakingClaimsIndexKey()).Err()
	})

	entryA, err := coord.Join(ctx, userA, mode, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	entryB, err := coord.Join(ctx, userB, mode, now.Add(time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("join B: %v", err)
	}
	claim, err := coord.ClaimPair(ctx, mode, now.Add(2*time.Second), 15*time.Second, 20)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}
	if err := coord.ReleaseClaim(ctx, claim, true, true, 30*time.Second); err != nil {
		t.Fatalf("release: %v", err)
	}
	restoredA, err := coord.GetEntry(ctx, userA)
	if err != nil || restoredA == nil {
		t.Fatalf("restored A: %+v err=%v", restoredA, err)
	}
	if restoredA.State != "searching" || restoredA.EnqueuedAtMs != entryA.EnqueuedAtMs {
		t.Fatalf("priority A lost: got %+v want enqueued %d", restoredA, entryA.EnqueuedAtMs)
	}
	restoredB, err := coord.GetEntry(ctx, userB)
	if err != nil || restoredB == nil {
		t.Fatalf("restored B: %+v err=%v", restoredB, err)
	}
	if restoredB.EnqueuedAtMs != entryB.EnqueuedAtMs {
		t.Fatalf("priority B lost: got %d want %d", restoredB.EnqueuedAtMs, entryB.EnqueuedAtMs)
	}
}

func TestMatchmakingClaimSurvivesSearchingLeaseExpiry(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	var claimID string
	t.Cleanup(func() {
		keys := []string{
			matchmakingPlayerKey(userA), matchmakingPlayerKey(userB),
			matchmakingQueueKey(mode), matchmakingClaimsIndexKey(),
		}
		if claimID != "" {
			keys = append(keys, matchmakingClaimKey(claimID))
		}
		_ = client.Del(ctx, keys...).Err()
	})

	entryA, err := coord.Join(ctx, userA, mode, now, 2*time.Second)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	entryB, err := coord.Join(ctx, userB, mode, now.Add(time.Millisecond), 2*time.Second)
	if err != nil {
		t.Fatalf("join B: %v", err)
	}
	defer func() {
		_ = client.Del(ctx, "matchmaking:v1:entry:"+entryA.EntryID, "matchmaking:v1:entry:"+entryB.EntryID).Err()
	}()

	claim, err := coord.ClaimPair(ctx, mode, now.Add(1900*time.Millisecond), 15*time.Second, 20)
	if err != nil || claim == nil {
		t.Fatalf("claim near lease expiry: claim=%+v err=%v", claim, err)
	}
	claimID = claim.ClaimID

	for _, userID := range []uuid.UUID{userA, userB} {
		entry, renewErr := coord.RenewLease(ctx, userID, 30*time.Second, now.Add(3*time.Second))
		if renewErr != nil || entry == nil || entry.State != "claimed" || entry.ClaimID != claim.ClaimID {
			t.Fatalf("claimed player %s was lost after search lease: entry=%+v err=%v", userID, entry, renewErr)
		}
	}

	ttl, err := client.PTTL(ctx, matchmakingClaimKey(claim.ClaimID)).Result()
	if err != nil {
		t.Fatalf("claim TTL: %v", err)
	}
	if ttl != -1 {
		t.Fatalf("claim recovery record TTL = %v, want persistent until finalize/release", ttl)
	}
}

func TestMatchmakingDuplicateJoinPreservesActiveClaim(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	coord := NewMatchmakingCoordinator(client)
	mode := "ranked_standard"
	now := time.Now().UTC()
	userA, userB := uuid.New(), uuid.New()
	var entryA, entryB *QueueEntry
	var claim *PairClaim
	t.Cleanup(func() {
		keys := []string{
			matchmakingPlayerKey(userA), matchmakingPlayerKey(userB),
			matchmakingQueueKey(mode), matchmakingClaimsIndexKey(),
		}
		if entryA != nil {
			keys = append(keys, "matchmaking:v1:entry:"+entryA.EntryID)
		}
		if entryB != nil {
			keys = append(keys, "matchmaking:v1:entry:"+entryB.EntryID)
		}
		if claim != nil {
			keys = append(keys, matchmakingClaimKey(claim.ClaimID))
		}
		_ = client.Del(ctx, keys...).Err()
	})

	var err error
	entryA, err = coord.Join(ctx, userA, mode, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	entryB, err = coord.Join(ctx, userB, mode, now.Add(time.Millisecond), 30*time.Second)
	if err != nil {
		t.Fatalf("join B: %v", err)
	}
	claim, err = coord.ClaimPair(ctx, mode, now.Add(time.Second), 15*time.Second, 20)
	if err != nil || claim == nil {
		t.Fatalf("claim: claim=%+v err=%v", claim, err)
	}

	// Capture entry→user pointers before the duplicate join.
	entryPtrA, err := client.Get(ctx, "matchmaking:v1:entry:"+entryA.EntryID).Result()
	if err != nil {
		t.Fatalf("entry pointer A before: %v", err)
	}
	entryPtrB, err := client.Get(ctx, "matchmaking:v1:entry:"+entryB.EntryID).Result()
	if err != nil {
		t.Fatalf("entry pointer B before: %v", err)
	}
	scoreBefore, err := client.ZScore(ctx, matchmakingClaimsIndexKey(), claim.ClaimID).Result()
	if err != nil {
		t.Fatalf("claim index before: %v", err)
	}

	duplicate, err := coord.Join(ctx, userA, mode, now.Add(2*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("duplicate join while claimed: %v", err)
	}
	if duplicate.EntryID != entryA.EntryID || duplicate.State != "claimed" || duplicate.ClaimID != claim.ClaimID {
		t.Fatalf("duplicate join replaced claim state: got=%+v original=%+v claim=%+v", duplicate, entryA, claim)
	}

	for _, userID := range []uuid.UUID{userA, userB} {
		preserved, getErr := coord.GetEntry(ctx, userID)
		if getErr != nil || preserved == nil || preserved.State != "claimed" || preserved.ClaimID != claim.ClaimID {
			t.Fatalf("player %s claim not preserved: entry=%+v err=%v", userID, preserved, getErr)
		}
	}
	// Opponent claim pointer must remain; entry maps and recovery index must be untouched.
	entryPtrAAfter, err := client.Get(ctx, "matchmaking:v1:entry:"+entryA.EntryID).Result()
	if err != nil || entryPtrAAfter != entryPtrA {
		t.Fatalf("entry pointer A changed: before=%s after=%s err=%v", entryPtrA, entryPtrAAfter, err)
	}
	entryPtrBAfter, err := client.Get(ctx, "matchmaking:v1:entry:"+entryB.EntryID).Result()
	if err != nil || entryPtrBAfter != entryPtrB {
		t.Fatalf("entry pointer B changed: before=%s after=%s err=%v", entryPtrB, entryPtrBAfter, err)
	}
	scoreAfter, err := client.ZScore(ctx, matchmakingClaimsIndexKey(), claim.ClaimID).Result()
	if err != nil || scoreAfter != scoreBefore {
		t.Fatalf("claim index changed: before=%v after=%v err=%v", scoreBefore, scoreAfter, err)
	}
	storedClaim, err := coord.GetClaim(ctx, claim.ClaimID)
	if err != nil || storedClaim == nil {
		t.Fatalf("claim payload lost: claim=%+v err=%v", storedClaim, err)
	}
	if storedClaim.UserIDA != claim.UserIDA || storedClaim.UserIDB != claim.UserIDB {
		t.Fatalf("claim participants changed: got=%+v want=%+v", storedClaim, claim)
	}
	// Second player duplicate join must also be a no-op.
	duplicateB, err := coord.Join(ctx, userB, mode, now.Add(3*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("duplicate join B while claimed: %v", err)
	}
	if duplicateB.EntryID != entryB.EntryID || duplicateB.State != "claimed" || duplicateB.ClaimID != claim.ClaimID {
		t.Fatalf("duplicate join B replaced claim: got=%+v original=%+v", duplicateB, entryB)
	}
}
