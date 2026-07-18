package matchplay_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchplay"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func chatTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping matchplay chat repository integration tests")
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
	if !db.Migrator().HasTable("team_messages") {
		t.Skip("team_messages table missing; run goose up through 00019")
	}
	return db
}

func seedChatUser(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'x', 'user', 'active', ?, ?)`,
		id, id.String()+"@chat.test", now, now,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO user_profiles (user_id, display_name, locale, created_at, updated_at) VALUES (?, ?, 'en', ?, ?)`,
		id, "Chat-"+id.String()[:8], now, now,
	).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

type chatFixture struct {
	matchID uuid.UUID
	gameID  uuid.UUID
	mapID   uuid.UUID
	team1   []uuid.UUID
	team2   []uuid.UUID
}

func seedChatMatch(t *testing.T, db *gorm.DB) chatFixture {
	t.Helper()
	now := time.Now().UTC()
	gameID := uuid.New()
	matchID := uuid.New()
	mapID := uuid.New()
	if err := db.Exec(`
		INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status, created_at, updated_at)
		VALUES (?, ?, ?, 'private', 'free', 'mixed', 'active', ?, ?)
	`, mapID, "chat-"+mapID.String(), "Chat fixture", now, now).Error; err != nil {
		t.Fatalf("seed map: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO games (id, mode, status, map_id, round_count, scoring_version, total_score, started_at, created_at, updated_at)
		VALUES (?, 'casual_duo', 'active', ?, 1, 1, 0, ?, ?, ?)
	`, gameID, mapID, now, now, now).Error; err != nil {
		t.Fatalf("seed game: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO matches (
			id, formation_key, game_id, mode, status, playlist, format, team_size,
			team_one_score, team_two_score, last_activity_at, matched_at, started_at, created_at, updated_at
		) VALUES (
			?, ?, ?, 'casual_duo', 'active', 'casual', 'duo', 2,
			0, 0, ?, ?, ?, ?, ?
		)`,
		matchID, "fk-"+matchID.String(), gameID, now, now, now, now, now,
	).Error; err != nil {
		// Fallback without newer columns if migration partial.
		t.Fatalf("seed match: %v", err)
	}

	users := []uuid.UUID{seedChatUser(t, db), seedChatUser(t, db), seedChatUser(t, db), seedChatUser(t, db)}
	for i, uid := range users {
		slot := 1
		if i >= 2 {
			slot = 2
		}
		gp := uuid.New()
		_ = db.Exec(`INSERT INTO game_players (id, game_id, user_id, display_name, status, total_score, team_slot) VALUES (?, ?, ?, ?, 'active', 0, ?)`,
			gp, gameID, uid, "P", slot)
		if err := db.Exec(`
			INSERT INTO match_players (match_id, user_id, game_player_id, status, team_slot, assigned_at)
			VALUES (?, ?, ?, 'assigned', ?, ?)`,
			matchID, uid, gp, slot, now,
		).Error; err != nil {
			t.Fatalf("seed participant: %v", err)
		}
	}
	return chatFixture{
		matchID: matchID,
		gameID:  gameID,
		mapID:   mapID,
		team1:   users[:2],
		team2:   users[2:],
	}
}

func cleanupChatFixture(t *testing.T, db *gorm.DB, fx chatFixture) {
	t.Helper()
	_ = db.Exec(`DELETE FROM team_message_reports WHERE match_id = ?`, fx.matchID)
	_ = db.Exec(`DELETE FROM team_messages WHERE match_id = ?`, fx.matchID)
	_ = db.Exec(`DELETE FROM match_mutes WHERE match_id = ?`, fx.matchID)
	_ = db.Exec(`DELETE FROM match_players WHERE match_id = ?`, fx.matchID)
	_ = db.Exec(`DELETE FROM matches WHERE id = ?`, fx.matchID)
	_ = db.Exec(`DELETE FROM game_players WHERE game_id = ?`, fx.gameID)
	_ = db.Exec(`DELETE FROM games WHERE id = ?`, fx.gameID)
	_ = db.Exec(`DELETE FROM maps WHERE id = ?`, fx.mapID)
	all := append(append([]uuid.UUID{}, fx.team1...), fx.team2...)
	for _, uid := range all {
		_ = db.Exec(`DELETE FROM user_profiles WHERE user_id = ?`, uid)
		_ = db.Exec(`DELETE FROM users WHERE id = ?`, uid)
	}
}

func TestTeamMessageTableName(t *testing.T) {
	t.Parallel()
	if (matchplay.TeamMessage{}).TableName() != "team_messages" {
		t.Fatal("team_messages table name")
	}
	if (matchplay.MatchMute{}).TableName() != "match_mutes" {
		t.Fatal("match_mutes table name")
	}
	if (matchplay.MessageReport{}).TableName() != "team_message_reports" {
		t.Fatal("team_message_reports table name")
	}
}

func TestChatRepositorySequencePaginationAndTeamFilter(t *testing.T) {
	db := chatTestDB(t)
	repo := matchplay.NewRepository(db)
	fx := seedChatMatch(t, db)
	t.Cleanup(func() { cleanupChatFixture(t, db, fx) })
	ctx := context.Background()
	now := time.Now().UTC()

	var seqs []int64
	for i := 0; i < 3; i++ {
		msg, created, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
			MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
			ClientMessageID: uuid.New(), Text: "m", Status: matchplay.MessageStatusActive,
			CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour),
		})
		if err != nil || !created {
			t.Fatalf("insert team1: %v created=%v", err, created)
		}
		seqs = append(seqs, msg.Sequence)
	}
	// Opponent team message
	if _, _, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 2, AuthorUserID: fx.team2[0],
		ClientMessageID: uuid.New(), Text: "opp", Status: matchplay.MessageStatusActive,
		CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("insert team2: %v", err)
	}

	page1, err := repo.ListTeamMessages(ctx, fx.matchID, 1, 0, 2, nil)
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 = %d err=%v", len(page1), err)
	}
	if page1[0].Sequence >= page1[1].Sequence {
		t.Fatalf("not ascending: %d %d", page1[0].Sequence, page1[1].Sequence)
	}
	page2, err := repo.ListTeamMessages(ctx, fx.matchID, 1, page1[1].Sequence, 10, nil)
	if err != nil || len(page2) != 1 || page2[0].Sequence != seqs[2] {
		t.Fatalf("page2 = %+v err=%v", page2, err)
	}
	// Team filter: team1 never sees team2
	for _, m := range append(page1, page2...) {
		if m.TeamSlot != 1 {
			t.Fatalf("leaked team slot %d", m.TeamSlot)
		}
	}
}

func TestChatRepositoryUniqueClientIDAndFile(t *testing.T) {
	db := chatTestDB(t)
	repo := matchplay.NewRepository(db)
	fx := seedChatMatch(t, db)
	t.Cleanup(func() { cleanupChatFixture(t, db, fx) })
	ctx := context.Background()
	now := time.Now().UTC()

	clientID := uuid.New()
	msg1, created, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: clientID, Text: "once", Status: matchplay.MessageStatusActive,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil || !created {
		t.Fatalf("first insert: %v created=%v", err, created)
	}
	msg2, created, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: clientID, Text: "once", Status: matchplay.MessageStatusActive,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil || created || msg2.ID != msg1.ID {
		t.Fatalf("idempotent insert: created=%v id=%s want=%s err=%v", created, msg2.ID, msg1.ID, err)
	}

	// Unique file binding requires a real files row.
	fileID := uuid.New()
	if err := db.Exec(`
		INSERT INTO files (id, owner_user_id, file_name, content_type, size_bytes, storage_key, is_public, purpose, sanitization_status, context_id, created_at)
		VALUES (?, ?, 'a.jpg', 'image/jpeg', 10, ?, false, 'team_chat', 'ready', ?, ?)
	`, fileID, fx.team1[0], "chat-file-"+fileID.String(), fx.matchID, now).Error; err != nil {
		t.Skipf("files purpose columns unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Exec(`DELETE FROM files WHERE id = ?`, fileID) })

	if _, _, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: uuid.New(), Text: "", FileID: &fileID, Status: matchplay.MessageStatusActive,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("file message: %v", err)
	}
	_, _, err = repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: uuid.New(), Text: "x", FileID: &fileID, Status: matchplay.MessageStatusActive,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != matchplay.ErrAttachmentAlreadyBound {
		t.Fatalf("duplicate file err = %v", err)
	}
}

func TestChatRepositoryConcurrentSends(t *testing.T) {
	db := chatTestDB(t)
	repo := matchplay.NewRepository(db)
	fx := seedChatMatch(t, db)
	t.Cleanup(func() { cleanupChatFixture(t, db, fx) })
	ctx := context.Background()
	now := time.Now().UTC()

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	seqs := make(chan int64, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			msg, created, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
				MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
				ClientMessageID: uuid.New(), Text: "c", Status: matchplay.MessageStatusActive,
				CreatedAt: now, ExpiresAt: now.Add(time.Hour),
			})
			if err != nil {
				errs <- err
				return
			}
			if !created {
				errs <- err
				return
			}
			seqs <- msg.Sequence
		}()
	}
	wg.Wait()
	close(errs)
	close(seqs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent insert: %v", err)
		}
	}
	seen := map[int64]struct{}{}
	for s := range seqs {
		if _, ok := seen[s]; ok {
			t.Fatalf("duplicate sequence %d", s)
		}
		seen[s] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("got %d unique sequences, want %d", len(seen), n)
	}
}

func TestChatRepositoryReportRetentionAndLegalHold(t *testing.T) {
	db := chatTestDB(t)
	repo := matchplay.NewRepository(db)
	fx := seedChatMatch(t, db)
	t.Cleanup(func() { cleanupChatFixture(t, db, fx) })
	ctx := context.Background()
	now := time.Now().UTC()

	// Expired unreported message is cleanup-eligible.
	expired, _, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: uuid.New(), Text: "old", Status: matchplay.MessageStatusActive,
		CreatedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Reported with active retention.
	reported, _, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[1],
		ClientMessageID: uuid.New(), Text: "bad", Status: matchplay.MessageStatusActive,
		CreatedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	retention := now.Add(180 * 24 * time.Hour)
	if _, created, err := repo.InsertReport(ctx, matchplay.MessageReport{
		MessageID: reported.ID, MatchID: fx.matchID, ReporterUserID: fx.team1[0],
		Reason: matchplay.ReportReasonSpam, Status: matchplay.ReportStatusPending,
		CreatedAt: now, RetentionUntil: retention,
	}, retention); err != nil || !created {
		t.Fatalf("report: created=%v err=%v", created, err)
	}
	// Idempotent report
	if _, created, err := repo.InsertReport(ctx, matchplay.MessageReport{
		MessageID: reported.ID, MatchID: fx.matchID, ReporterUserID: fx.team1[0],
		Reason: matchplay.ReportReasonSpam, Status: matchplay.ReportStatusPending,
		CreatedAt: now, RetentionUntil: retention,
	}, retention); err != nil || created {
		t.Fatalf("report replay: created=%v err=%v", created, err)
	}

	// Legal hold message
	held, _, err := repo.InsertMessage(ctx, matchplay.TeamMessage{
		MatchID: fx.matchID, TeamSlot: 1, AuthorUserID: fx.team1[0],
		ClientMessageID: uuid.New(), Text: "hold", Status: matchplay.MessageStatusActive,
		CreatedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.InsertReport(ctx, matchplay.MessageReport{
		MessageID: held.ID, MatchID: fx.matchID, ReporterUserID: fx.team1[1],
		Reason: matchplay.ReportReasonOther, Status: matchplay.ReportStatusPending,
		CreatedAt: now, RetentionUntil: now.Add(-time.Hour), LegalHold: true,
	}, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	cands, err := repo.ListCleanupCandidates(ctx, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundExpired, foundReported, foundHeld := false, false, false
	for _, c := range cands {
		if c.ID == expired.ID {
			foundExpired = true
		}
		if c.ID == reported.ID {
			foundReported = true
		}
		if c.ID == held.ID {
			foundHeld = true
		}
	}
	if !foundExpired {
		t.Fatal("expected expired unreported candidate")
	}
	if foundReported {
		t.Fatal("reported message within retention must not be selected")
	}
	if foundHeld {
		t.Fatal("legal-hold message must not be selected")
	}

	// Unique reports constraint covered by idempotent insert above.
	// Mute unique key
	if err := repo.UpsertMute(ctx, matchplay.MatchMute{
		MatchID: fx.matchID, MuterUserID: fx.team1[0], MutedUserID: fx.team1[1], CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertMute(ctx, matchplay.MatchMute{
		MatchID: fx.matchID, MuterUserID: fx.team1[0], MutedUserID: fx.team1[1], CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}
