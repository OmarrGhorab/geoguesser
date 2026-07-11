package friends_test

import (
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

	req, err := repo.CreateRequest(t.Context(), a, b)
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if req.Status != friends.StatusPending {
		t.Fatalf("status = %s", req.Status)
	}

	// Duplicate
	if _, err := repo.CreateRequest(t.Context(), a, b); err != friends.ErrAlreadyPending {
		t.Fatalf("duplicate err = %v", err)
	}
	// Reciprocal
	if _, err := repo.CreateRequest(t.Context(), b, a); err != friends.ErrAlreadyPending {
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

	req, err := repo.CreateRequest(t.Context(), a, b)
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
	if _, err := repo.CreateRequest(t.Context(), b, a); err != nil {
		t.Fatalf("re-request after decline: %v", err)
	}
}

func TestRepositoryBlockReplacesAndUnblockOwnership(t *testing.T) {
	db := testDB(t)
	repo := friends.NewRepository(db)
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupPair(t, db, a, b) })

	if _, err := repo.CreateRequest(t.Context(), a, b); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if err := repo.BlockUser(t.Context(), a, b); err != nil {
		t.Fatalf("BlockUser: %v", err)
	}
	// privacy: request looks like not found
	if _, err := repo.CreateRequest(t.Context(), b, a); err != friends.ErrTargetNotFound {
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

	if _, err := repo.CreateRequest(t.Context(), a, a); err != friends.ErrSelfPair {
		t.Fatalf("self err = %v", err)
	}
	if _, err := repo.CreateRequest(t.Context(), a, inactive); err != friends.ErrTargetNotFound {
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
