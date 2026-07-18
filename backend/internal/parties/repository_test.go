package parties_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/parties"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping parties repository integration tests")
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
	if !db.Migrator().HasTable("parties") {
		t.Skip("parties table missing; run goose up with 00019_casual_ranked_team_modes.sql")
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', ?, ?, ?)`,
		id, id.String()+"@party.test", status, now, now).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec(`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, ?, 'en', ?, ?)`,
		id, "PartyUser-"+id.String()[:8], now, now).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

func cleanupUsers(t *testing.T, db *gorm.DB, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		_ = db.Exec(`DELETE FROM party_invites WHERE inviter_user_id = ? OR invitee_user_id = ?`, id, id)
		_ = db.Exec(`DELETE FROM party_members WHERE user_id = ?`, id)
	}
	// Delete parties led by these users (and any orphaned members/invites via cascade).
	for _, id := range ids {
		_ = db.Exec(`DELETE FROM parties WHERE leader_user_id = ?`, id)
	}
	for _, id := range ids {
		_ = db.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, id)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, id)
	}
}

func TestPartyTableNames(t *testing.T) {
	if (parties.Party{}).TableName() != "parties" {
		t.Fatal("parties table name")
	}
	if (parties.PartyMember{}).TableName() != "party_members" {
		t.Fatal("party_members table name")
	}
	if (parties.PartyInvite{}).TableName() != "party_invites" {
		t.Fatal("party_invites table name")
	}
}

func TestCapacityForFormat(t *testing.T) {
	if c, ok := parties.CapacityForFormat("duo"); !ok || c != 2 {
		t.Fatalf("duo = %d %v", c, ok)
	}
	if c, ok := parties.CapacityForFormat("squad"); !ok || c != 4 {
		t.Fatalf("squad = %d %v", c, ok)
	}
	if _, ok := parties.CapacityForFormat("solo"); ok {
		t.Fatal("solo should not be durable party format")
	}
}

func TestRepositoryCreateOneActivePartyUniqueness(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "duo", now)
	if err != nil {
		t.Fatalf("CreateParty: %v", err)
	}
	if snap.Party.Format != "duo" || snap.Party.Capacity != 2 {
		t.Fatalf("party = %+v", snap.Party)
	}
	if len(snap.Members) != 1 || snap.Members[0].Member.UserID != leader {
		t.Fatalf("members = %+v", snap.Members)
	}

	if _, err := repo.CreateParty(t.Context(), leader, "squad", now); err != parties.ErrActivePartyConflict {
		t.Fatalf("second create err = %v, want active party conflict", err)
	}

	// Version starts at 0
	if snap.Party.Version != 0 {
		t.Fatalf("version = %d", snap.Party.Version)
	}
}

func TestRepositoryInviteAcceptCapacityAndVersion(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	friend := seedUser(t, db, "active")
	extra := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader, friend, extra) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "duo", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := snap.Party.ID

	inv, err := repo.CreateInvite(t.Context(), pid, leader, friend, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	// Duplicate pending invite
	if _, err := repo.CreateInvite(t.Context(), pid, leader, friend, now.Add(15*time.Minute), now); err != parties.ErrAlreadyInvited {
		t.Fatalf("dup invite err = %v", err)
	}

	joined, err := repo.AcceptInvite(t.Context(), inv.Invite.ID, friend, now.Add(time.Second))
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if joined.ActiveMemberCount() != 2 {
		t.Fatalf("members = %d", joined.ActiveMemberCount())
	}
	if joined.Party.Version <= snap.Party.Version {
		t.Fatalf("version did not increment: before=%d after=%d", snap.Party.Version, joined.Party.Version)
	}
	// Readiness cleared
	for _, m := range joined.Members {
		if m.Member.Ready {
			t.Fatalf("readiness not cleared")
		}
	}

	// Full party rejects more invites
	if _, err := repo.CreateInvite(t.Context(), pid, leader, extra, now.Add(15*time.Minute), now); err != parties.ErrPartyFull {
		t.Fatalf("full err = %v", err)
	}
}

func TestRepositoryConcurrentAcceptRespectsCapacity(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	a := seedUser(t, db, "active")
	b := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader, a, b) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "duo", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := snap.Party.ID

	invA, err := repo.CreateInvite(t.Context(), pid, leader, a, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("invite a: %v", err)
	}
	// For concurrent race, create second invite while only 1 slot remains after first accept.
	// Start with squad to have room for two invites then reduce — better: duo with two invites
	// created while party only has leader; both try accept for the one remaining slot.
	invB, err := repo.CreateInvite(t.Context(), pid, leader, b, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("invite b: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := repo.AcceptInvite(t.Context(), invA.Invite.ID, a, now.Add(time.Second))
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := repo.AcceptInvite(t.Context(), invB.Invite.ID, b, now.Add(time.Second))
		errs <- err
	}()
	wg.Wait()
	close(errs)

	var success, failures int
	for err := range errs {
		if err == nil {
			success++
		} else {
			failures++
			if err != parties.ErrPartyFull && err != parties.ErrTargetBusy {
				// Either full or busy is acceptable for the loser.
				t.Logf("concurrent accept err = %v", err)
			}
		}
	}
	if success != 1 {
		t.Fatalf("expected exactly one successful accept, got success=%d failures=%d", success, failures)
	}

	final, err := repo.LoadSnapshot(t.Context(), pid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if final.ActiveMemberCount() != 2 {
		t.Fatalf("active members = %d, want 2", final.ActiveMemberCount())
	}
}

func TestRepositoryLeaveLeaderTransferAndVersion(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	// Deterministic UUIDs for leadership tie-break.
	leader := seedUser(t, db, "active")
	// seed early/late with natural UUIDs; joined_at order drives transfer.
	early := seedUser(t, db, "active")
	late := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader, early, late) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "squad", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := snap.Party.ID

	inv1, err := repo.CreateInvite(t.Context(), pid, leader, early, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("invite early: %v", err)
	}
	if _, err := repo.AcceptInvite(t.Context(), inv1.Invite.ID, early, now.Add(time.Second)); err != nil {
		t.Fatalf("accept early: %v", err)
	}
	inv2, err := repo.CreateInvite(t.Context(), pid, leader, late, now.Add(15*time.Minute), now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("invite late: %v", err)
	}
	if _, err := repo.AcceptInvite(t.Context(), inv2.Invite.ID, late, now.Add(3*time.Second)); err != nil {
		t.Fatalf("accept late: %v", err)
	}

	// Mark both ready then leave leader — readiness clears + leadership transfers.
	if _, err := repo.SetReadiness(t.Context(), pid, early, true, now.Add(4*time.Second)); err != nil {
		t.Fatalf("ready early: %v", err)
	}
	before, _ := repo.LoadSnapshot(t.Context(), pid)
	if err := repo.LeaveParty(t.Context(), pid, leader, now.Add(5*time.Second)); err != nil {
		t.Fatalf("leave: %v", err)
	}
	after, err := repo.LoadSnapshot(t.Context(), pid)
	if err != nil {
		t.Fatalf("load after leave: %v", err)
	}
	if after.Party.LeaderUserID != early {
		t.Fatalf("leader = %s, want early %s", after.Party.LeaderUserID, early)
	}
	if after.Party.Version <= before.Party.Version {
		t.Fatalf("version not bumped: %d -> %d", before.Party.Version, after.Party.Version)
	}
	for _, m := range after.Members {
		if m.Member.Ready {
			t.Fatal("readiness should clear on roster change")
		}
	}
}

func TestRepositoryDisbandIdempotentAndRevokesInvites(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	friend := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader, friend) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "duo", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.CreateInvite(t.Context(), snap.Party.ID, leader, friend, now.Add(15*time.Minute), now); err != nil {
		t.Fatalf("invite: %v", err)
	}

	if err := repo.DisbandParty(t.Context(), snap.Party.ID, leader, now.Add(time.Second)); err != nil {
		t.Fatalf("disband: %v", err)
	}
	if err := repo.DisbandParty(t.Context(), snap.Party.ID, leader, now.Add(2*time.Second)); err != nil {
		t.Fatalf("disband retry: %v", err)
	}

	closed, err := repo.LoadSnapshot(t.Context(), snap.Party.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if closed.Party.Status != parties.StatusClosed {
		t.Fatalf("status = %s", closed.Party.Status)
	}
	if closed.ActiveMemberCount() != 0 {
		t.Fatalf("members still active: %d", closed.ActiveMemberCount())
	}

	// Invitee should not see pending invite
	pending, err := repo.ListPendingInvitesForUser(t.Context(), friend, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected revoked invites omitted, got %d", len(pending))
	}
}

func TestRepositoryQueuedBlocksLeave(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader) })

	now := time.Now().UTC()
	snap, err := repo.CreateParty(t.Context(), leader, "duo", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Exec(`UPDATE parties SET status = 'queued' WHERE id = ?`, snap.Party.ID).Error; err != nil {
		t.Fatalf("set queued: %v", err)
	}
	if err := repo.LeaveParty(t.Context(), snap.Party.ID, leader, now.Add(time.Second)); err != parties.ErrPartyLocked {
		t.Fatalf("leave err = %v, want party locked", err)
	}
}

func TestRepositoryDeclineIdempotent(t *testing.T) {
	db := testDB(t)
	repo := parties.NewRepository(db)
	leader := seedUser(t, db, "active")
	friend := seedUser(t, db, "active")
	t.Cleanup(func() { cleanupUsers(t, db, leader, friend) })

	now := time.Now().UTC()
	snap, _ := repo.CreateParty(t.Context(), leader, "duo", now)
	inv, err := repo.CreateInvite(t.Context(), snap.Party.ID, leader, friend, now.Add(15*time.Minute), now)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := repo.DeclineInvite(t.Context(), inv.Invite.ID, friend, now.Add(time.Second)); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if err := repo.DeclineInvite(t.Context(), inv.Invite.ID, friend, now.Add(2*time.Second)); err != nil {
		t.Fatalf("decline retry: %v", err)
	}
}
