package friends_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/friends"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping friends repository integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if !db.Migrator().HasTable("friendships") {
		t.Skip("friendships table missing; run goose up with 00015_friends_social_graph.sql")
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', ?, ?, ?)`,
		id, id.String()+"@example.test", status, now, now).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, ?, 'en', ?, ?)`,
		id, "User-"+id.String()[:8], now, now).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

func cleanupPair(t *testing.T, db *gorm.DB, a, b uuid.UUID) {
	t.Helper()
	_ = db.Exec(`DELETE FROM friendships WHERE user_a_id IN (?,?) OR user_b_id IN (?,?)`, a, b, a, b)
	_ = db.Exec(`DELETE FROM user_profiles WHERE user_id IN (?,?)`, a, b)
	_ = db.Exec(`DELETE FROM users WHERE id IN (?,?)`, a, b)
}

func TestFriendshipTableName(t *testing.T) {
	if (friends.Friendship{}).TableName() != "friendships" {
		t.Fatal("table name")
	}
}

func TestNormalizePairHelpers(t *testing.T) {
	a := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	b := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	ua, ub, err := friends.NormalizePair(b, a)
	if err != nil || ua != a || ub != b {
		t.Fatalf("sorted pair failed: %v %v %v", ua, ub, err)
	}
}

func TestRepositoryCreateAcceptDeclineFlow(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })

	req, _, err := repo.CreateRequest(t.Context(), a, b)
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if req.Status != friends.StatusPending {
		t.Fatalf("status = %s", req.Status)
	}

	// Duplicate
	if _, _, err := repo.CreateRequest(t.Context(), a, b); err != friends.ErrAlreadyPending {
		t.Fatalf("duplicate err = %v", err)
	}
	// Reciprocal
	if _, _, err := repo.CreateRequest(t.Context(), b, a); err != friends.ErrAlreadyPending {
		t.Fatalf("reciprocal err = %v", err)
	}

	// Incoming for B
	incoming, err := repo.ListIncomingRequests(t.Context(), b, 20, "")
	if err != nil || len(incoming.Items) != 1 {
		t.Fatalf("incoming = %+v err=%v", incoming, err)
	}
	// Outgoing for A
	outgoing, err := repo.ListOutgoingRequests(t.Context(), a, 20, "")
	if err != nil || len(outgoing.Items) != 1 {
		t.Fatalf("outgoing = %+v err=%v", outgoing, err)
	}

	// A cannot accept own request
	if _, _, err := repo.AcceptRequest(t.Context(), req.ID, a); err != friends.ErrNotFound {
		t.Fatalf("self accept err = %v", err)
	}

	accepted, other, err := repo.AcceptRequest(t.Context(), req.ID, b)
	if err != nil {
		t.Fatalf("AcceptRequest: %v", err)
	}
	if accepted.Status != friends.StatusAccepted || other.UserID != a {
		t.Fatalf("accepted=%+v other=%+v", accepted, other)
	}

	friendsA, err := repo.ListAcceptedFriends(t.Context(), a, 20, "")
	if err != nil || len(friendsA.Items) != 1 || friendsA.Items[0].Other.UserID != b {
		t.Fatalf("friendsA=%+v err=%v", friendsA, err)
	}
	friendsB, err := repo.ListAcceptedFriends(t.Context(), b, 20, "")
	if err != nil || len(friendsB.Items) != 1 || friendsB.Items[0].Other.UserID != a {
		t.Fatalf("friendsB=%+v err=%v", friendsB, err)
	}

	if err := repo.RemoveFriendship(t.Context(), a, b); err != nil {
		t.Fatalf("RemoveFriendship: %v", err)
	}
	// idempotent
	if err := repo.RemoveFriendship(t.Context(), a, b); err != nil {
		t.Fatalf("RemoveFriendship retry: %v", err)
	}
}

func TestRepositoryDeclineDeletesPending(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })

	req, _, err := repo.CreateRequest(t.Context(), a, b)
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if err := repo.DeclineRequest(t.Context(), req.ID, b); err != nil {
		t.Fatalf("DeclineRequest: %v", err)
	}
	got, err := repo.GetByID(t.Context(), req.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected deleted request, got %+v", got)
	}
	// either may re-request
	if _, _, err := repo.CreateRequest(t.Context(), b, a); err != nil {
		t.Fatalf("re-request after decline: %v", err)
	}
}

func TestRepositoryBlockReplacesAndUnblockOwnership(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })

	if _, _, err := repo.CreateRequest(t.Context(), a, b); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if err := repo.BlockUser(t.Context(), a, b); err != nil {
		t.Fatalf("BlockUser: %v", err)
	}
	// privacy: request looks like not found
	if _, _, err := repo.CreateRequest(t.Context(), b, a); err != friends.ErrTargetNotFound {
		t.Fatalf("request while blocked err = %v", err)
	}
	// same blocker idempotent
	if err := repo.BlockUser(t.Context(), a, b); err != nil {
		t.Fatalf("reblock: %v", err)
	}
	// other party unblock is no-op
	if err := repo.UnblockUser(t.Context(), b, a); err != nil {
		t.Fatalf("unauthorized unblock: %v", err)
	}
	ua, ub, _ := friends.NormalizePair(a, b)
	still, err := repo.GetByPair(t.Context(), ua, ub)
	if err != nil || still == nil || still.Status != friends.StatusBlocked {
		t.Fatalf("expected block preserved: %+v err=%v", still, err)
	}
	// owner unblocks
	if err := repo.UnblockUser(t.Context(), a, b); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	gone, err := repo.GetByPair(t.Context(), ua, ub)
	if err != nil || gone != nil {
		t.Fatalf("expected removed block, got %+v err=%v", gone, err)
	}
}

func TestRepositorySelfPairAndInactiveTarget(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	inactive := seedUser(t, db, "disabled")
	t.Cleanup(func() { cleanupPair(t, db, a, inactive) })

	if _, _, err := repo.CreateRequest(t.Context(), a, a); err != friends.ErrSelfPair {
		t.Fatalf("self err = %v", err)
	}
	if _, _, err := repo.CreateRequest(t.Context(), a, inactive); err != friends.ErrTargetNotFound {
		t.Fatalf("inactive err = %v", err)
	}
}

func TestRepositoryLifecycleConstraints(t *testing.T) {
	db := testDB(t)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })
	ua, ub, _ := friends.NormalizePair(a, b)
	now := time.Now().UTC()
	// invalid: accepted without accepted_at
	err := db.Exec(`INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'accepted', ?, ?)`, uuid.New(), ua, ub, a, now, now).Error
	if err == nil {
		t.Fatal("expected lifecycle constraint failure for accepted without accepted_at")
	}
}

func TestRepositoryPairUniqueness(t *testing.T) {
	db := testDB(t)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })
	ua, ub, _ := friends.NormalizePair(a, b)
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?)`, uuid.New(), ua, ub, a, now, now).Error; err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	err := db.Exec(`INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?)`, uuid.New(), ua, ub, b, now, now).Error
	if err == nil {
		t.Fatal("expected unique pair failure")
	}
}

func TestRepositoryForeignKeyRejectsUnknownUsers(t *testing.T) {
	db := testDB(t)
	a := seedUser(t, db, "active")
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM friendships WHERE user_a_id = ? OR user_b_id = ?`, a, a)
		_ = db.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, a)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, a)
	})
	missing := uuid.New()
	ua, ub, _ := friends.NormalizePair(a, missing)
	now := time.Now().UTC()
	err := db.Exec(`INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'pending', ?, ?)`, uuid.New(), ua, ub, a, now, now).Error
	if err == nil {
		t.Fatal("expected foreign key failure for missing user")
	}
}

func TestRepositoryPaginationCursorForOutgoingRequests(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	viewer := seedUser(t, db, "active")
	targets := make([]uuid.UUID, 0, 5)
	for i := 0; i < 5; i++ {
		targets = append(targets, seedUser(t, db, "active"))
	}
	t.Cleanup(func() {
		for _, id := range targets {
			cleanupPair(t, db, viewer, id)
		}
	})
	for _, target := range targets {
		if _, _, err := repo.CreateRequest(t.Context(), viewer, target); err != nil {
			t.Fatalf("CreateRequest: %v", err)
		}
	}
	page1, err := repo.ListOutgoingRequests(t.Context(), viewer, 2, "")
	if err != nil || len(page1.Items) != 2 || page1.NextCursor == nil {
		t.Fatalf("page1=%+v err=%v", page1, err)
	}
	page2, err := repo.ListOutgoingRequests(t.Context(), viewer, 2, *page1.NextCursor)
	if err != nil || len(page2.Items) != 2 {
		t.Fatalf("page2=%+v err=%v", page2, err)
	}
	// No overlap between pages.
	seen := map[uuid.UUID]bool{}
	for _, item := range page1.Items {
		seen[item.Friendship.ID] = true
	}
	for _, item := range page2.Items {
		if seen[item.Friendship.ID] {
			t.Fatalf("cursor page overlap on %s", item.Friendship.ID)
		}
	}
}

func TestRepositoryBlockedListOrdersByUpdatedAt(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	blocker := seedUser(t, db, "active")
	oldTarget := seedUser(t, db, "active")
	newTarget := seedUser(t, db, "active")
	t.Cleanup(func() {
		cleanupPair(t, db, blocker, oldTarget)
		cleanupPair(t, db, blocker, newTarget)
	})

	// Insert blocked rows with deterministic timestamps. Disable the updated_at
	// trigger so the test can control block time independently of created_at.
	if err := db.Exec(`ALTER TABLE friendships DISABLE TRIGGER friendships_updated_at`).Error; err != nil {
		t.Fatalf("disable trigger: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec(`ALTER TABLE friendships ENABLE TRIGGER friendships_updated_at`)
	})

	insertBlocked := func(target uuid.UUID, created, updated time.Time) {
		t.Helper()
		ua, ub, _ := friends.NormalizePair(blocker, target)
		if err := db.Exec(`
			INSERT INTO friendships (id, user_a_id, user_b_id, requested_by_user_id, status, blocked_by_user_id, accepted_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'blocked', ?, NULL, ?, ?)
		`, uuid.New(), ua, ub, blocker, blocker, created, updated).Error; err != nil {
			t.Fatalf("insert blocked: %v", err)
		}
	}
	// Old friendship created long ago but blocked recently-less; new block is more recent.
	insertBlocked(oldTarget, time.Now().UTC().Add(-72*time.Hour), time.Now().UTC().Add(-2*time.Hour))
	insertBlocked(newTarget, time.Now().UTC().Add(-1*time.Hour), time.Now().UTC().Add(-1*time.Minute))

	page, err := repo.ListBlockedUsers(t.Context(), blocker, 10, "")
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("blocked page=%+v err=%v", page, err)
	}
	// Must order by updated_at (block time), not original created_at.
	if page.Items[0].Other.UserID != newTarget {
		t.Fatalf("expected newest block first, got %s want %s (created_at order would favor wrong row)", page.Items[0].Other.UserID, newTarget)
	}
	if page.Items[1].Other.UserID != oldTarget {
		t.Fatalf("expected older block second, got %s", page.Items[1].Other.UserID)
	}
}

func TestRepositoryConcurrentCreateAcceptNoDeadlock(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })

	req, _, err := repo.CreateRequest(t.Context(), a, b)
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}

	errCh := make(chan error, 2)
	go func() {
		_, _, err := repo.AcceptRequest(t.Context(), req.ID, b)
		errCh <- err
	}()
	go func() {
		// Concurrent reverse create should conflict or serialize, not deadlock.
		_, _, err := repo.CreateRequest(t.Context(), b, a)
		errCh <- err
	}()

	// Fail if both hang (deadlock).
	timeout := time.After(10 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			if err == nil {
				continue
			}
			// Domain conflicts/not-found are expected under races.
			if errors.Is(err, friends.ErrAlreadyPending) ||
				errors.Is(err, friends.ErrAlreadyFriends) ||
				errors.Is(err, friends.ErrNotFound) ||
				errors.Is(err, friends.ErrTargetNotFound) {
				continue
			}
			t.Fatalf("unexpected concurrent error: %v", err)
		case <-timeout:
			t.Fatal("concurrent create/accept timed out — possible deadlock")
		}
	}

	// Relationship must be either accepted or still pending (never duplicated).
	ua, ub, _ := friends.NormalizePair(a, b)
	got, err := repo.GetByPair(t.Context(), ua, ub)
	if err != nil {
		t.Fatalf("GetByPair: %v", err)
	}
	if got == nil {
		t.Fatal("expected durable friendship edge after concurrent operations")
	}
	if got.Status != friends.StatusAccepted && got.Status != friends.StatusPending {
		t.Fatalf("unexpected status %s", got.Status)
	}
}

func TestRepositoryDisabledCallerCannotAccept(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })
	req, _, err := repo.CreateRequest(t.Context(), a, b)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Exec(`UPDATE users SET status = 'disabled' WHERE id = ?`, b).Error; err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, _, err := repo.AcceptRequest(t.Context(), req.ID, b); err != friends.ErrUnauthorized {
		t.Fatalf("accept as disabled = %v, want unauthorized", err)
	}
}

func BenchmarkListAcceptedFriends(b *testing.B) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		b.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		b.Skip(err)
	}
	if !db.Migrator().HasTable("friendships") {
		b.Skip("friendships missing")
	}
	repo := friends.NewRepository(db)
	viewer := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		viewer, viewer.String()+"@bench.test", now, now).Error; err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = db.Exec(`DELETE FROM friendships WHERE user_a_id = ? OR user_b_id = ?`, viewer, viewer)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, viewer)
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.ListAcceptedFriends(b.Context(), viewer, 20, ""); err != nil {
			b.Fatal(err)
		}
	}
}
