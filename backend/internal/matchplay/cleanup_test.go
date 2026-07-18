package matchplay_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
)

type recordingDeleter struct {
	mu       sync.Mutex
	calls    []string
	failKeys map[string]int // key -> remaining failures before success
	err      error
}

func (d *recordingDeleter) DeleteObject(_ context.Context, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, key)
	if d.err != nil {
		return d.err
	}
	if n, ok := d.failKeys[key]; ok && n > 0 {
		d.failKeys[key] = n - 1
		return errors.New("transient storage error")
	}
	return nil
}

func (d *recordingDeleter) callCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.calls)
}

func TestCleanupUnreportedReportedLegalHoldAndRaw(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	matchID := uuid.New()
	a, b := uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{uuid.New(), uuid.New()}, matchplay.MatchStatusCompleted, nil)

	// 30d unreported expired
	expiredID := uuid.New()
	store.messages[expiredID] = &matchplay.TeamMessage{
		ID: expiredID, MatchID: matchID, TeamSlot: 1, AuthorUserID: a,
		ClientMessageID: uuid.New(), Sequence: 1, Text: "old",
		Status: matchplay.MessageStatusActive, CreatedAt: now.Add(-40 * 24 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	// Reported still in 180d window
	reportedID := uuid.New()
	store.messages[reportedID] = &matchplay.TeamMessage{
		ID: reportedID, MatchID: matchID, TeamSlot: 1, AuthorUserID: b,
		ClientMessageID: uuid.New(), Sequence: 2, Text: "reported",
		Status: matchplay.MessageStatusReported, CreatedAt: now.Add(-40 * 24 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	rid := uuid.New()
	store.reports[rid] = &matchplay.MessageReport{
		ID: rid, MessageID: reportedID, MatchID: matchID, ReporterUserID: a,
		Reason: matchplay.ReportReasonSpam, Status: matchplay.ReportStatusPending,
		CreatedAt: now.Add(-time.Hour), RetentionUntil: now.Add(100 * 24 * time.Hour),
	}
	// Legal hold even if retention elapsed
	heldID := uuid.New()
	store.messages[heldID] = &matchplay.TeamMessage{
		ID: heldID, MatchID: matchID, TeamSlot: 1, AuthorUserID: a,
		ClientMessageID: uuid.New(), Sequence: 3, Text: "held",
		Status: matchplay.MessageStatusReported, CreatedAt: now.Add(-200 * 24 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	hid := uuid.New()
	store.reports[hid] = &matchplay.MessageReport{
		ID: hid, MessageID: heldID, MatchID: matchID, ReporterUserID: b,
		Reason: matchplay.ReportReasonOther, Status: matchplay.ReportStatusPending,
		CreatedAt: now.Add(-200 * 24 * time.Hour), RetentionUntil: now.Add(-time.Hour), LegalHold: true,
	}

	// File attached to expired message
	fileID := uuid.New()
	store.messages[expiredID].FileID = &fileID
	store.files[fileID] = &matchplay.ChatFile{
		ID: fileID, OwnerUserID: a, StorageKey: "sanitized/a.jpg", Purpose: matchplay.FilePurposeTeamChat,
		SanitizationStatus: matchplay.SanitizationReady,
	}

	// Raw upload older than 24h
	uploadID := uuid.New()
	rawKey := "raw/pending.bin"
	store.uploads[uploadID] = &matchplay.ChatUpload{
		ID: uploadID, OwnerUserID: a, StorageKey: "uploads/pending", Status: matchplay.UploadStatusPending,
		Purpose: matchplay.FilePurposeTeamChat, SanitizationStatus: matchplay.SanitizationPending,
		RawStorageKey: &rawKey, CreatedAt: now.Add(-25 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}
	completedUploadID := uuid.New()
	completedRawKey := "raw/completed.jpg"
	completedDerivativeKey := "sanitized/completed.jpg"
	store.uploads[completedUploadID] = &matchplay.ChatUpload{
		ID: completedUploadID, OwnerUserID: a, StorageKey: completedDerivativeKey, Status: matchplay.UploadStatusCompleted,
		Purpose: matchplay.FilePurposeTeamChat, SanitizationStatus: matchplay.SanitizationReady,
		RawStorageKey: &completedRawKey, CreatedAt: now.Add(-25 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}

	deleter := &recordingDeleter{}
	cleaner := matchplay.NewCleaner(store, deleter, nil, nil, matchplay.CleanupConfig{
		BatchSize: 100, RawMaxAge: 24 * time.Hour, DeleteRetries: 2, RetryBackoff: time.Millisecond,
	}).WithClock(func() time.Time { return now })

	stats, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if stats.MessagesDeleted != 1 {
		t.Fatalf("messages deleted = %d, want 1", stats.MessagesDeleted)
	}
	// ListCleanupCandidates already excludes protected rows; defense-in-depth skip count may be 0.
	if _, ok := store.messages[expiredID]; ok {
		t.Fatal("expired message still present")
	}
	if _, ok := store.messages[reportedID]; !ok {
		t.Fatal("reported message deleted early")
	}
	if _, ok := store.messages[heldID]; !ok {
		t.Fatal("legal-hold message deleted")
	}
	if stats.UploadsDeleted != 1 {
		t.Fatalf("uploads deleted = %d", stats.UploadsDeleted)
	}
	if _, ok := store.uploads[uploadID]; ok {
		t.Fatal("raw upload still present")
	}
	if completed := store.uploads[completedUploadID]; completed == nil || completed.RawStorageKey != nil {
		t.Fatal("completed upload metadata should remain with raw key cleared")
	}
	deleter.mu.Lock()
	for _, key := range deleter.calls {
		if key == completedDerivativeKey {
			deleter.mu.Unlock()
			t.Fatal("raw cleanup deleted a completed sanitized derivative")
		}
	}
	deleter.mu.Unlock()
	if deleter.callCount() < 2 {
		t.Fatalf("expected storage deletes, got %d", deleter.callCount())
	}
}

func TestCleanupStorageDeleteRetry(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	now := time.Now().UTC()
	matchID := uuid.New()
	a := uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, uuid.New()}, []uuid.UUID{uuid.New(), uuid.New()}, matchplay.MatchStatusCompleted, nil)

	msgID := uuid.New()
	fileID := uuid.New()
	store.messages[msgID] = &matchplay.TeamMessage{
		ID: msgID, MatchID: matchID, TeamSlot: 1, AuthorUserID: a,
		ClientMessageID: uuid.New(), Sequence: 1, Text: "x", FileID: &fileID,
		Status: matchplay.MessageStatusActive, CreatedAt: now.Add(-48 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	store.files[fileID] = &matchplay.ChatFile{
		ID: fileID, OwnerUserID: a, StorageKey: "retry-key", Purpose: matchplay.FilePurposeTeamChat,
		SanitizationStatus: matchplay.SanitizationReady,
	}

	deleter := &recordingDeleter{failKeys: map[string]int{"retry-key": 1}}
	cleaner := matchplay.NewCleaner(store, deleter, nil, nil, matchplay.CleanupConfig{
		BatchSize: 10, DeleteRetries: 3, RetryBackoff: time.Millisecond,
	}).WithClock(func() time.Time { return now })

	stats, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.MessagesDeleted != 1 {
		t.Fatalf("messages deleted = %d", stats.MessagesDeleted)
	}
	if stats.ObjectFailures != 0 {
		t.Fatalf("object failures = %d after retry", stats.ObjectFailures)
	}
	if stats.FilesDeleted != 1 {
		t.Fatalf("files deleted = %d", stats.FilesDeleted)
	}
	// First attempt fails, second succeeds => at least 2 calls.
	if deleter.callCount() < 2 {
		t.Fatalf("retry calls = %d", deleter.callCount())
	}
}

func TestCleanupBoundedBatch(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	now := time.Now().UTC()
	matchID := uuid.New()
	a := uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, uuid.New()}, []uuid.UUID{uuid.New(), uuid.New()}, matchplay.MatchStatusCompleted, nil)
	for i := 0; i < 5; i++ {
		id := uuid.New()
		store.messages[id] = &matchplay.TeamMessage{
			ID: id, MatchID: matchID, TeamSlot: 1, AuthorUserID: a,
			ClientMessageID: uuid.New(), Sequence: int64(i + 1), Text: "x",
			Status: matchplay.MessageStatusActive, CreatedAt: now.Add(-48 * time.Hour),
			ExpiresAt: now.Add(-time.Hour),
		}
	}
	cleaner := matchplay.NewCleaner(store, &recordingDeleter{}, nil, nil, matchplay.CleanupConfig{
		BatchSize: 2, DeleteRetries: 1,
	}).WithClock(func() time.Time { return now })

	stats, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.MessagesSelected != 2 || stats.MessagesDeleted != 2 {
		t.Fatalf("batch stats = %+v", stats)
	}
	if len(store.messages) != 3 {
		t.Fatalf("remaining messages = %d", len(store.messages))
	}
}
