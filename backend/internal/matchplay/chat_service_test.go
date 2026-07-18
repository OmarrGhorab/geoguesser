package matchplay_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/session"
)

// memoryChatStore is an in-memory ChatStore for unit tests.
type memoryChatStore struct {
	mu           sync.Mutex
	matches      map[uuid.UUID]*matchplay.Match
	participants map[uuid.UUID][]matchplay.MatchParticipant
	messages     map[uuid.UUID]*matchplay.TeamMessage // id -> msg
	byClient     map[string]uuid.UUID                 // match|author|client -> id
	byFile       map[uuid.UUID]uuid.UUID              // fileID -> messageID
	seq          int64
	files        map[uuid.UUID]*matchplay.ChatFile
	mutes        map[string]struct{} // match|muter|muted
	reports      map[uuid.UUID]*matchplay.MessageReport
	reportKey    map[string]uuid.UUID // message|reporter -> report id
	names        map[uuid.UUID]string
	uploads      map[uuid.UUID]*matchplay.ChatUpload
}

func newMemoryChatStore() *memoryChatStore {
	return &memoryChatStore{
		matches:      map[uuid.UUID]*matchplay.Match{},
		participants: map[uuid.UUID][]matchplay.MatchParticipant{},
		messages:     map[uuid.UUID]*matchplay.TeamMessage{},
		byClient:     map[string]uuid.UUID{},
		byFile:       map[uuid.UUID]uuid.UUID{},
		files:        map[uuid.UUID]*matchplay.ChatFile{},
		mutes:        map[string]struct{}{},
		reports:      map[uuid.UUID]*matchplay.MessageReport{},
		reportKey:    map[string]uuid.UUID{},
		names:        map[uuid.UUID]string{},
		uploads:      map[uuid.UUID]*matchplay.ChatUpload{},
	}
}

func clientKey(matchID, author, client uuid.UUID) string {
	return matchID.String() + "|" + author.String() + "|" + client.String()
}

func muteKey(matchID, muter, muted uuid.UUID) string {
	return matchID.String() + "|" + muter.String() + "|" + muted.String()
}

func reportKey(messageID, reporter uuid.UUID) string {
	return messageID.String() + "|" + reporter.String()
}

func (m *memoryChatStore) seedDuo(matchID uuid.UUID, team1, team2 []uuid.UUID, status string, chatUntil *time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	match := &matchplay.Match{
		ID:              matchID,
		GameID:          uuid.New(),
		Mode:            "casual_duo",
		Status:          status,
		Playlist:        matchplay.PlaylistCasual,
		Format:          matchplay.FormatDuo,
		TeamSize:        2,
		LastActivityAt:  now,
		ChatAccessUntil: chatUntil,
		MatchedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if status != matchplay.MatchStatusActive && status != matchplay.MatchStatusMatched {
		completed := now.Add(-time.Minute)
		match.CompletedAt = &completed
	}
	m.matches[matchID] = match
	var parts []matchplay.MatchParticipant
	for _, uid := range team1 {
		parts = append(parts, matchplay.MatchParticipant{
			MatchID: matchID, UserID: uid, GamePlayerID: uuid.New(),
			Status: matchplay.ParticipantStatusActive, TeamSlot: 1, AssignedAt: now,
		})
		m.names[uid] = "T1-" + uid.String()[:8]
	}
	for _, uid := range team2 {
		parts = append(parts, matchplay.MatchParticipant{
			MatchID: matchID, UserID: uid, GamePlayerID: uuid.New(),
			Status: matchplay.ParticipantStatusActive, TeamSlot: 2, AssignedAt: now,
		})
		m.names[uid] = "T2-" + uid.String()[:8]
	}
	m.participants[matchID] = parts
}

func (m *memoryChatStore) seedSolo(matchID, userA, userB uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	m.matches[matchID] = &matchplay.Match{
		ID: matchID, GameID: uuid.New(), Mode: "casual_solo", Status: matchplay.MatchStatusActive,
		Playlist: matchplay.PlaylistCasual, Format: matchplay.FormatSolo, TeamSize: 1,
		LastActivityAt: now, MatchedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	m.participants[matchID] = []matchplay.MatchParticipant{
		{MatchID: matchID, UserID: userA, GamePlayerID: uuid.New(), Status: matchplay.ParticipantStatusActive, TeamSlot: 1, AssignedAt: now},
		{MatchID: matchID, UserID: userB, GamePlayerID: uuid.New(), Status: matchplay.ParticipantStatusActive, TeamSlot: 2, AssignedAt: now},
	}
}

func (m *memoryChatStore) LoadMatchForChat(_ context.Context, matchID uuid.UUID) (*matchplay.Match, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	match, ok := m.matches[matchID]
	if !ok {
		return nil, nil
	}
	cp := *match
	return &cp, nil
}

func (m *memoryChatStore) FindParticipant(_ context.Context, matchID, userID uuid.UUID) (*matchplay.MatchParticipant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.participants[matchID] {
		if p.UserID == userID {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memoryChatStore) ListTeamParticipants(_ context.Context, matchID uuid.UUID) ([]matchplay.MatchParticipant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]matchplay.MatchParticipant{}, m.participants[matchID]...), nil
}

func (m *memoryChatStore) FindMessageByClientID(_ context.Context, matchID, authorID, clientMessageID uuid.UUID) (*matchplay.TeamMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byClient[clientKey(matchID, authorID, clientMessageID)]
	if !ok {
		return nil, nil
	}
	cp := *m.messages[id]
	return &cp, nil
}

func (m *memoryChatStore) FindMessage(_ context.Context, matchID, messageID uuid.UUID) (*matchplay.TeamMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[messageID]
	if !ok || msg.MatchID != matchID {
		return nil, nil
	}
	cp := *msg
	return &cp, nil
}

func (m *memoryChatStore) InsertMessage(_ context.Context, msg matchplay.TeamMessage) (*matchplay.TeamMessage, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ck := clientKey(msg.MatchID, msg.AuthorUserID, msg.ClientMessageID)
	if id, ok := m.byClient[ck]; ok {
		cp := *m.messages[id]
		return &cp, false, nil
	}
	if msg.FileID != nil {
		if _, ok := m.byFile[*msg.FileID]; ok {
			return nil, false, matchplay.ErrAttachmentAlreadyBound
		}
	}
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	m.seq++
	msg.Sequence = m.seq
	cp := msg
	m.messages[msg.ID] = &cp
	m.byClient[ck] = msg.ID
	if msg.FileID != nil {
		m.byFile[*msg.FileID] = msg.ID
	}
	out := cp
	return &out, true, nil
}

func (m *memoryChatStore) ListTeamMessages(_ context.Context, matchID uuid.UUID, teamSlot int, afterSeq int64, limit int, excludeAuthorIDs []uuid.UUID) ([]matchplay.TeamMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	excl := map[uuid.UUID]struct{}{}
	for _, id := range excludeAuthorIDs {
		excl[id] = struct{}{}
	}
	var rows []matchplay.TeamMessage
	for _, msg := range m.messages {
		if msg.MatchID != matchID || msg.TeamSlot != teamSlot || msg.Sequence <= afterSeq {
			continue
		}
		if msg.Status != matchplay.MessageStatusActive && msg.Status != matchplay.MessageStatusReported {
			continue
		}
		if _, skip := excl[msg.AuthorUserID]; skip {
			continue
		}
		rows = append(rows, *msg)
	}
	// Sort by sequence ascending (simple insertion for tests).
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].Sequence < rows[i].Sequence {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (m *memoryChatStore) GetChatFile(_ context.Context, fileID uuid.UUID) (*matchplay.ChatFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[fileID]
	if !ok {
		return nil, nil
	}
	cp := *f
	return &cp, nil
}

func (m *memoryChatStore) GetChatFiles(_ context.Context, fileIDs []uuid.UUID) (map[uuid.UUID]matchplay.ChatFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]matchplay.ChatFile{}
	for _, id := range fileIDs {
		if f, ok := m.files[id]; ok {
			out[id] = *f
		}
	}
	return out, nil
}

func (m *memoryChatStore) CountAuthorMessagesSince(_ context.Context, matchID, authorID uuid.UUID, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, msg := range m.messages {
		if msg.MatchID == matchID && msg.AuthorUserID == authorID && !msg.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (m *memoryChatStore) CountAuthorImageMessages(_ context.Context, matchID, authorID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, msg := range m.messages {
		if msg.MatchID == matchID && msg.AuthorUserID == authorID && msg.FileID != nil {
			n++
		}
	}
	return n, nil
}

func (m *memoryChatStore) DisplayNames(_ context.Context, userIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]string{}
	for _, id := range userIDs {
		out[id] = m.names[id]
	}
	return out, nil
}

func (m *memoryChatStore) UpsertMute(_ context.Context, mute matchplay.MatchMute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mutes[muteKey(mute.MatchID, mute.MuterUserID, mute.MutedUserID)] = struct{}{}
	return nil
}

func (m *memoryChatStore) DeleteMute(_ context.Context, matchID, muterID, mutedID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mutes, muteKey(matchID, muterID, mutedID))
	return nil
}

func (m *memoryChatStore) ListMutedUserIDs(_ context.Context, matchID, muterID uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []uuid.UUID
	prefix := matchID.String() + "|" + muterID.String() + "|"
	for k := range m.mutes {
		if strings.HasPrefix(k, prefix) {
			raw := strings.TrimPrefix(k, prefix)
			if id, err := uuid.Parse(raw); err == nil {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func (m *memoryChatStore) InsertReport(_ context.Context, report matchplay.MessageReport, extendExpiresTo time.Time) (*matchplay.MessageReport, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[report.MessageID]
	if !ok || msg.MatchID != report.MatchID {
		return nil, false, matchplay.ErrNotFound
	}
	rk := reportKey(report.MessageID, report.ReporterUserID)
	if id, exists := m.reportKey[rk]; exists {
		cp := *m.reports[id]
		return &cp, false, nil
	}
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	cp := report
	m.reports[report.ID] = &cp
	m.reportKey[rk] = report.ID
	msg.Status = matchplay.MessageStatusReported
	if extendExpiresTo.After(msg.ExpiresAt) {
		msg.ExpiresAt = extendExpiresTo
	}
	out := cp
	return &out, true, nil
}

func (m *memoryChatStore) ListCleanupCandidates(_ context.Context, now time.Time, limit int) ([]matchplay.TeamMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	protected := map[uuid.UUID]struct{}{}
	for _, r := range m.reports {
		if r.LegalHold || r.RetentionUntil.After(now) {
			protected[r.MessageID] = struct{}{}
		}
	}
	var rows []matchplay.TeamMessage
	for _, msg := range m.messages {
		if !msg.ExpiresAt.Before(now) {
			continue
		}
		if _, ok := protected[msg.ID]; ok {
			continue
		}
		rows = append(rows, *msg)
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (m *memoryChatStore) ListProtectedMessageIDs(_ context.Context, messageIDs []uuid.UUID, now time.Time) (map[uuid.UUID]struct{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]struct{}{}
	want := map[uuid.UUID]struct{}{}
	for _, id := range messageIDs {
		want[id] = struct{}{}
	}
	for _, r := range m.reports {
		if _, ok := want[r.MessageID]; !ok {
			continue
		}
		if r.LegalHold || r.RetentionUntil.After(now) {
			out[r.MessageID] = struct{}{}
		}
	}
	return out, nil
}

func (m *memoryChatStore) DeleteMessages(_ context.Context, messageIDs []uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var fileIDs []uuid.UUID
	for _, id := range messageIDs {
		msg, ok := m.messages[id]
		if !ok {
			continue
		}
		if msg.FileID != nil {
			fileIDs = append(fileIDs, *msg.FileID)
			delete(m.byFile, *msg.FileID)
		}
		delete(m.byClient, clientKey(msg.MatchID, msg.AuthorUserID, msg.ClientMessageID))
		delete(m.messages, id)
	}
	for rid, r := range m.reports {
		for _, mid := range messageIDs {
			if r.MessageID == mid {
				delete(m.reportKey, reportKey(r.MessageID, r.ReporterUserID))
				delete(m.reports, rid)
			}
		}
	}
	return fileIDs, nil
}

func (m *memoryChatStore) ListRawUploadsForCleanup(_ context.Context, cutoff time.Time, limit int) ([]matchplay.ChatUpload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rows []matchplay.ChatUpload
	for _, u := range m.uploads {
		if u.Purpose != matchplay.FilePurposeTeamChat || !u.CreatedAt.Before(cutoff) {
			continue
		}
		rows = append(rows, *u)
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (m *memoryChatStore) DeleteUploads(_ context.Context, uploadIDs []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range uploadIDs {
		delete(m.uploads, id)
	}
	return nil
}

func (m *memoryChatStore) ClearUploadRawStorageKeys(_ context.Context, uploadIDs []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	wanted := make(map[uuid.UUID]struct{}, len(uploadIDs))
	for _, id := range uploadIDs {
		wanted[id] = struct{}{}
	}
	for i := range m.uploads {
		if _, ok := wanted[m.uploads[i].ID]; ok {
			m.uploads[i].RawStorageKey = nil
		}
	}
	return nil
}

func (m *memoryChatStore) DeleteFiles(_ context.Context, fileIDs []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range fileIDs {
		delete(m.files, id)
	}
	return nil
}

func (m *memoryChatStore) DeleteMutesForMatches(_ context.Context, matchIDs []uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	set := map[uuid.UUID]struct{}{}
	for _, id := range matchIDs {
		set[id] = struct{}{}
	}
	for k := range m.mutes {
		parts := strings.SplitN(k, "|", 2)
		if len(parts) == 0 {
			continue
		}
		if mid, err := uuid.Parse(parts[0]); err == nil {
			if _, ok := set[mid]; ok {
				delete(m.mutes, k)
			}
		}
	}
	return nil
}

func (m *memoryChatStore) TouchActivity(_ context.Context, matchID uuid.UUID, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if match, ok := m.matches[matchID]; ok {
		match.LastActivityAt = at
	}
	return nil
}

type stubSigner struct {
	url string
	err error
}

func (s stubSigner) PresignedDownloadURL(context.Context, string, time.Duration) (string, error) {
	return s.url, s.err
}

func chatSession(userID uuid.UUID) *session.Context {
	id := userID.String()
	return &session.Context{Kind: session.KindUser, UserID: &id, Role: "user"}
}

func newChatService(t *testing.T, store *memoryChatStore, now func() time.Time) *matchplay.Service {
	t.Helper()
	svc := matchplay.NewService(nil, nil, nil, matchplay.ServiceConfig{}).
		WithChatStore(store).
		WithChatConfig(matchplay.ChatServiceConfig{
			TeamChatImagesEnabled: true,
			RetentionDays:         30,
			ReportRetentionDays:   180,
			MaxImagesPerMatch:     5,
			BurstLimit:            10,
			BurstWindow:           10 * time.Second,
			MinuteLimit:           60,
			MinuteWindow:          time.Minute,
		}).
		WithAttachmentSigner(stubSigner{url: "https://example.test/signed"})
	if now != nil {
		svc.WithClock(now)
	}
	return svc
}

type chatEventCapture struct {
	recipients []uuid.UUID
	calls      int
}

func (c *chatEventCapture) PublishMatchEvent(context.Context, uuid.UUID, string, int64, *uuid.UUID, *uuid.UUID, any) error {
	return nil
}

func (c *chatEventCapture) PublishChatMessageCreated(_ context.Context, _ uuid.UUID, _ int, _ int64, recipients []uuid.UUID, _ any) error {
	c.calls++
	c.recipients = append([]uuid.UUID{}, recipients...)
	return nil
}

func TestSendMessagePublishesOnlyToUnmutedTeammates(t *testing.T) {
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	if err := store.UpsertMute(context.Background(), matchplay.MatchMute{MatchID: matchID, MuterUserID: b, MutedUserID: a}); err != nil {
		t.Fatalf("mute: %v", err)
	}
	capture := &chatEventCapture{}
	svc := newChatService(t, store, nil).WithEvents(capture)
	if _, err := svc.SendMessage(context.Background(), chatSession(a), matchID, matchplay.SendMessageRequest{ClientMessageID: uuid.New(), Text: strPtr("clue")}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if capture.calls != 1 || len(capture.recipients) != 1 || capture.recipients[0] != a {
		t.Fatalf("recipients = %v calls=%d", capture.recipients, capture.calls)
	}
}

func TestSendMessageAuthorizationAndValidation(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	// Non-participant -> not found
	if _, err := svc.SendMessage(ctx, chatSession(uuid.New()), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("hi"),
	}); err != matchplay.ErrNotFound {
		t.Fatalf("non-participant err = %v", err)
	}

	// Solo format unavailable
	soloID := uuid.New()
	store.seedSolo(soloID, a, c)
	if _, err := svc.SendMessage(ctx, chatSession(a), soloID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("hi"),
	}); err != matchplay.ErrChatUnavailable {
		t.Fatalf("solo err = %v", err)
	}

	// Empty text and no file
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(),
	}); err != matchplay.ErrInvalidRequest {
		t.Fatalf("empty err = %v", err)
	}

	// Too long
	long := strings.Repeat("x", matchplay.MaxMessageTextLength+1)
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: &long,
	}); err != matchplay.ErrMessageTooLong {
		t.Fatalf("too long err = %v", err)
	}

	// Happy path
	clientID := uuid.New()
	res, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: clientID, Text: strPtr("northern Spain?"),
	})
	if err != nil || !res.Created || res.Message.Text != "northern Spain?" {
		t.Fatalf("send = %+v err=%v", res, err)
	}

	// Idempotent replay
	res2, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: clientID, Text: strPtr("northern Spain?"),
	})
	if err != nil || res2.Created || res2.Message.ID != res.Message.ID {
		t.Fatalf("replay = %+v err=%v", res2, err)
	}
}

func TestSendMessageAttachmentRules(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	// Not ready
	fileID := uuid.New()
	ctxID := matchID
	store.files[fileID] = &matchplay.ChatFile{
		ID: fileID, OwnerUserID: a, ContentType: "image/jpeg", SizeBytes: 100,
		StorageKey: "k", Purpose: matchplay.FilePurposeTeamChat, ContextID: &ctxID,
		SanitizationStatus: matchplay.SanitizationPending,
	}
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), FileID: &fileID,
	}); err != matchplay.ErrAttachmentNotReady {
		t.Fatalf("pending err = %v", err)
	}

	// Ready attachment
	store.files[fileID].SanitizationStatus = matchplay.SanitizationReady
	res, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), FileID: &fileID,
	})
	if err != nil || res.Message.Attachment == nil || res.Message.Attachment.FileID != fileID {
		t.Fatalf("attach = %+v err=%v", res, err)
	}

	// One attachment binding
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), FileID: &fileID,
	}); err != matchplay.ErrAttachmentAlreadyBound {
		t.Fatalf("double bind err = %v", err)
	}

	// Image limit (5)
	for i := 0; i < 4; i++ {
		fid := uuid.New()
		store.files[fid] = &matchplay.ChatFile{
			ID: fid, OwnerUserID: a, ContentType: "image/jpeg", SizeBytes: 10,
			StorageKey: "k", Purpose: matchplay.FilePurposeTeamChat, ContextID: &ctxID,
			SanitizationStatus: matchplay.SanitizationReady,
		}
		if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
			ClientMessageID: uuid.New(), FileID: &fid,
		}); err != nil {
			t.Fatalf("image %d err = %v", i, err)
		}
	}
	fid := uuid.New()
	store.files[fid] = &matchplay.ChatFile{
		ID: fid, OwnerUserID: a, ContentType: "image/jpeg", SizeBytes: 10,
		StorageKey: "k", Purpose: matchplay.FilePurposeTeamChat, ContextID: &ctxID,
		SanitizationStatus: matchplay.SanitizationReady,
	}
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), FileID: &fid,
	}); err != matchplay.ErrImageLimitExceeded {
		t.Fatalf("image limit err = %v", err)
	}
}

func TestSendMessageRateLimits(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	now := time.Now().UTC()
	svc := newChatService(t, store, func() time.Time { return now })
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
			ClientMessageID: uuid.New(), Text: strPtr("m"),
		}); err != nil {
			t.Fatalf("msg %d err = %v", i, err)
		}
	}
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("over"),
	}); err != matchplay.ErrRateLimited {
		t.Fatalf("burst err = %v", err)
	}
}

func TestListMessagesMuteFilteringAndTerminalWindow(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	// Opponent messages never listed
	if _, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("team1"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendMessage(ctx, chatSession(c), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("team2"),
	}); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListMessages(ctx, chatSession(a), matchID, 0, 50)
	if err != nil || len(list.Data) != 1 || list.Data[0].Text != "team1" {
		t.Fatalf("list = %+v err=%v", list, err)
	}

	// Mute teammate filters
	if _, err := svc.SendMessage(ctx, chatSession(b), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("from-b"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MuteTeammate(ctx, chatSession(a), matchID, b); err != nil {
		t.Fatal(err)
	}
	list, err = svc.ListMessages(ctx, chatSession(a), matchID, 0, 50)
	if err != nil || len(list.Data) != 1 || list.Data[0].Text != "team1" {
		t.Fatalf("muted list = %+v err=%v", list, err)
	}

	// Terminal window expired
	past := time.Now().UTC().Add(-time.Hour)
	store.mu.Lock()
	store.matches[matchID].Status = matchplay.MatchStatusCompleted
	store.matches[matchID].ChatAccessUntil = &past
	completed := past.Add(-time.Minute)
	store.matches[matchID].CompletedAt = &completed
	store.mu.Unlock()
	if _, err := svc.ListMessages(ctx, chatSession(a), matchID, 0, 50); err != matchplay.ErrChatUnavailable {
		t.Fatalf("expired window err = %v", err)
	}

	// Within 15-minute window still allowed
	until := time.Now().UTC().Add(10 * time.Minute)
	store.mu.Lock()
	store.matches[matchID].ChatAccessUntil = &until
	store.mu.Unlock()
	if _, err := svc.ListMessages(ctx, chatSession(a), matchID, 0, 50); err != nil {
		t.Fatalf("result window err = %v", err)
	}
}

func TestReportPrivacyAndIdempotency(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	res, err := svc.SendMessage(ctx, chatSession(b), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), Text: strPtr("report me"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Opponent cannot report (looks like not found)
	if _, err := svc.ReportMessage(ctx, chatSession(c), matchID, res.Message.ID, matchplay.ReportMessageRequest{
		Reason: matchplay.ReportReasonSpam,
	}); err != matchplay.ErrNotFound {
		t.Fatalf("opponent report err = %v", err)
	}

	// Self-report invalid
	if _, err := svc.ReportMessage(ctx, chatSession(b), matchID, res.Message.ID, matchplay.ReportMessageRequest{
		Reason: matchplay.ReportReasonSpam,
	}); err != matchplay.ErrInvalidRequest {
		t.Fatalf("self report err = %v", err)
	}

	ack, err := svc.ReportMessage(ctx, chatSession(a), matchID, res.Message.ID, matchplay.ReportMessageRequest{
		Reason: matchplay.ReportReasonHarassment,
	})
	if err != nil || ack.Status != "accepted" {
		t.Fatalf("report = %+v err=%v", ack, err)
	}
	// Idempotent
	ack2, err := svc.ReportMessage(ctx, chatSession(a), matchID, res.Message.ID, matchplay.ReportMessageRequest{
		Reason: matchplay.ReportReasonHarassment,
	})
	if err != nil || ack2.Status != "accepted" {
		t.Fatalf("report replay = %+v err=%v", ack2, err)
	}

	// Message still listable for team (status reported, not removed)
	list, err := svc.ListMessages(ctx, chatSession(a), matchID, 0, 50)
	if err != nil || len(list.Data) != 1 || list.Data[0].Status != matchplay.MessageStatusReported {
		t.Fatalf("listed reported = %+v err=%v", list, err)
	}
}

func TestMuteTeammateOnly(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	if err := svc.MuteTeammate(ctx, chatSession(a), matchID, a); err != matchplay.ErrInvalidRequest {
		t.Fatalf("self mute err = %v", err)
	}
	if err := svc.MuteTeammate(ctx, chatSession(a), matchID, c); err != matchplay.ErrTeammateRequired {
		t.Fatalf("opponent mute err = %v", err)
	}
	if err := svc.MuteTeammate(ctx, chatSession(a), matchID, b); err != nil {
		t.Fatal(err)
	}
	// Idempotent
	if err := svc.MuteTeammate(ctx, chatSession(a), matchID, b); err != nil {
		t.Fatal(err)
	}
	if err := svc.UnmuteTeammate(ctx, chatSession(a), matchID, b); err != nil {
		t.Fatal(err)
	}
}

func TestGetAttachmentURL(t *testing.T) {
	t.Parallel()
	store := newMemoryChatStore()
	matchID := uuid.New()
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	store.seedDuo(matchID, []uuid.UUID{a, b}, []uuid.UUID{c, d}, matchplay.MatchStatusActive, nil)
	svc := newChatService(t, store, nil)
	ctx := context.Background()

	fileID := uuid.New()
	ctxID := matchID
	store.files[fileID] = &matchplay.ChatFile{
		ID: fileID, OwnerUserID: a, ContentType: "image/jpeg", SizeBytes: 12,
		StorageKey: "private/key", Purpose: matchplay.FilePurposeTeamChat, ContextID: &ctxID,
		SanitizationStatus: matchplay.SanitizationReady,
	}
	res, err := svc.SendMessage(ctx, chatSession(a), matchID, matchplay.SendMessageRequest{
		ClientMessageID: uuid.New(), FileID: &fileID,
	})
	if err != nil {
		t.Fatal(err)
	}

	url, err := svc.GetAttachmentURL(ctx, chatSession(b), matchID, res.Message.ID)
	if err != nil || url.URL == "" {
		t.Fatalf("teammate attachment = %+v err=%v", url, err)
	}
	if _, err := svc.GetAttachmentURL(ctx, chatSession(c), matchID, res.Message.ID); err != matchplay.ErrNotFound {
		t.Fatalf("opponent attachment err = %v", err)
	}
}

func strPtr(s string) *string { return &s }
