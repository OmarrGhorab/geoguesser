package matchplay

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/session"
)

// AttachmentSigner creates short-lived private download URLs for sanitized derivatives.
// Implementations wrap platform/storage.Provider without importing it here.
type AttachmentSigner interface {
	PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

type chatEventSink interface {
	PublishChatMessageCreated(ctx context.Context, matchID uuid.UUID, teamSlot int, version int64, eligibleRecipients []uuid.UUID, payload any) error
}

// ChatServiceConfig holds chat-specific knobs layered on ServiceConfig.
type ChatServiceConfig struct {
	TeamChatImagesEnabled bool
	RetentionDays         int
	ReportRetentionDays   int
	AttachmentURLTTL      time.Duration
	MaxImagesPerMatch     int
	BurstLimit            int
	BurstWindow           time.Duration
	MinuteLimit           int
	MinuteWindow          time.Duration
}

// chatDeps are optional chat collaborators on Service.
type chatDeps struct {
	store  ChatStore
	signer AttachmentSigner
	cfg    ChatServiceConfig
}

// WithChatStore attaches durable chat persistence.
func (s *Service) WithChatStore(store ChatStore) *Service {
	if s == nil {
		return s
	}
	s.chat.store = store
	return s
}

// WithAttachmentSigner attaches signed URL generation for message attachments.
func (s *Service) WithAttachmentSigner(signer AttachmentSigner) *Service {
	if s == nil {
		return s
	}
	s.chat.signer = signer
	return s
}

// WithChatConfig applies chat retention/limit knobs.
func (s *Service) WithChatConfig(cfg ChatServiceConfig) *Service {
	if s == nil {
		return s
	}
	s.chat.cfg = normalizeChatConfig(cfg)
	return s
}

func normalizeChatConfig(cfg ChatServiceConfig) ChatServiceConfig {
	if cfg.RetentionDays < 1 {
		cfg.RetentionDays = DefaultChatRetentionDays
	}
	if cfg.ReportRetentionDays < 1 {
		cfg.ReportRetentionDays = DefaultReportRetentionDays
	}
	if cfg.AttachmentURLTTL <= 0 {
		cfg.AttachmentURLTTL = AttachmentSignedURLTTL
	}
	if cfg.MaxImagesPerMatch < 1 {
		cfg.MaxImagesPerMatch = MaxImagesPerUserPerMatch
	}
	if cfg.BurstLimit < 1 {
		cfg.BurstLimit = MessageBurstLimit
	}
	if cfg.BurstWindow <= 0 {
		cfg.BurstWindow = MessageBurstWindow
	}
	if cfg.MinuteLimit < 1 {
		cfg.MinuteLimit = MessageMinuteLimit
	}
	if cfg.MinuteWindow <= 0 {
		cfg.MinuteWindow = MessageMinuteWindow
	}
	return cfg
}

func (s *Service) chatStore() ChatStore {
	if s == nil {
		return nil
	}
	if s.chat.store != nil {
		return s.chat.store
	}
	// Repository implements both Store and ChatStore.
	if cs, ok := s.store.(ChatStore); ok {
		return cs
	}
	return nil
}

func (s *Service) chatCfg() ChatServiceConfig {
	if s == nil {
		return normalizeChatConfig(ChatServiceConfig{})
	}
	return normalizeChatConfig(s.chat.cfg)
}

// SendMessage creates a team chat message or returns the prior row for the same client_message_id.
// Created=false means idempotent replay (HTTP 200); Created=true means HTTP 201.
func (s *Service) SendMessage(ctx context.Context, sess *session.Context, matchID uuid.UUID, req SendMessageRequest) (*SendMessageResult, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveChatMessage("unauthorized")
		s.metrics.ObserveCommand("chat_send", "unauthorized", s.clock().Sub(start))
		return nil, err
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveChatMessage("unavailable")
		s.metrics.ObserveCommand("chat_send", "unavailable", s.clock().Sub(start))
		return nil, ErrUnavailable
	}
	if req.ClientMessageID == uuid.Nil {
		s.metrics.ObserveChatMessage("invalid")
		s.metrics.ObserveCommand("chat_send", "invalid", s.clock().Sub(start))
		return nil, ErrInvalidRequest
	}

	text := ""
	if req.Text != nil {
		text = strings.TrimSpace(*req.Text)
	}
	if utf8.RuneCountInString(text) > MaxMessageTextLength {
		s.metrics.ObserveChatMessage("too_long")
		s.metrics.ObserveCommand("chat_send", "too_long", s.clock().Sub(start))
		return nil, ErrMessageTooLong
	}
	if text == "" && req.FileID == nil {
		s.metrics.ObserveChatMessage("invalid")
		s.metrics.ObserveCommand("chat_send", "invalid", s.clock().Sub(start))
		return nil, ErrInvalidRequest
	}

	// Idempotent replay before rate limits so replays do not consume budget.
	if existing, err := cs.FindMessageByClientID(ctx, matchID, userID, req.ClientMessageID); err != nil {
		s.metrics.ObserveChatMessage("error")
		s.metrics.ObserveCommand("chat_send", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	} else if existing != nil {
		resp, err := s.projectMessage(ctx, cs, *existing)
		if err != nil {
			s.metrics.ObserveChatMessage("error")
			s.metrics.ObserveCommand("chat_send", "error", s.clock().Sub(start))
			return nil, err
		}
		s.metrics.ObserveChatMessage("replay")
		s.metrics.ObserveIdempotency("replay")
		s.metrics.ObserveCommand("chat_send", "replay", s.clock().Sub(start))
		return &SendMessageResult{Message: resp, Created: false}, nil
	}

	match, part, err := s.authorizeChatAccess(ctx, cs, matchID, userID)
	if err != nil {
		s.metrics.ObserveChatMessage(chatOutcome(err))
		s.metrics.ObserveCommand("chat_send", chatOutcome(err), s.clock().Sub(start))
		return nil, err
	}

	cfg := s.chatCfg()
	now := s.clock()

	var file *ChatFile
	if req.FileID != nil {
		if !cfg.TeamChatImagesEnabled {
			s.metrics.ObserveChatAttachment("images_disabled")
			s.metrics.ObserveChatMessage("image_unavailable")
			s.metrics.ObserveCommand("chat_send", "image_unavailable", s.clock().Sub(start))
			return nil, ErrImageUnavailable
		}
		f, err := cs.GetChatFile(ctx, *req.FileID)
		if err != nil {
			s.metrics.ObserveChatMessage("error")
			return nil, mapStoreErr(err)
		}
		if f == nil {
			s.metrics.ObserveChatAttachment("not_found")
			s.metrics.ObserveChatMessage("not_found")
			s.metrics.ObserveCommand("chat_send", "not_found", s.clock().Sub(start))
			return nil, ErrNotFound
		}
		if err := validateChatAttachment(f, matchID, userID); err != nil {
			s.metrics.ObserveChatAttachment(chatOutcome(err))
			s.metrics.ObserveChatMessage(chatOutcome(err))
			s.metrics.ObserveCommand("chat_send", chatOutcome(err), s.clock().Sub(start))
			return nil, err
		}
		file = f
	}

	msg := TeamMessage{
		ID:              uuid.New(),
		MatchID:         matchID,
		TeamSlot:        part.TeamSlot,
		AuthorUserID:    userID,
		ClientMessageID: req.ClientMessageID,
		Text:            text,
		FileID:          req.FileID,
		Status:          MessageStatusActive,
		CreatedAt:       now,
		ExpiresAt:       MessageExpiresAt(now, match.CompletedAt, cfg.RetentionDays),
	}
	limits := ChatInsertLimits{
		BurstSince: now.Add(-cfg.BurstWindow), BurstLimit: cfg.BurstLimit,
		MinuteSince: now.Add(-cfg.MinuteWindow), MinuteLimit: cfg.MinuteLimit,
		MaxImagesInMatch: cfg.MaxImagesPerMatch,
	}
	var stored *TeamMessage
	var created bool
	if atomicStore, ok := cs.(atomicChatStore); ok {
		stored, created, err = atomicStore.InsertMessageWithLimits(ctx, msg, limits)
	} else {
		// Test/in-memory adapters retain the legacy seam; production PostgreSQL
		// always implements the atomic extension above.
		if err = s.checkChatLimits(ctx, cs, matchID, userID, req.FileID != nil, limits); err == nil {
			stored, created, err = cs.InsertMessage(ctx, msg)
		}
	}
	if err != nil {
		s.metrics.ObserveChatMessage(chatOutcome(err))
		s.metrics.ObserveCommand("chat_send", chatOutcome(err), s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if created {
		_ = cs.TouchActivity(ctx, matchID, now)
		if file != nil {
			s.metrics.ObserveChatAttachment("ok")
		}
	} else {
		s.metrics.ObserveIdempotency("replay")
	}

	resp, err := s.projectMessage(ctx, cs, *stored)
	if err != nil {
		s.metrics.ObserveChatMessage("error")
		s.metrics.ObserveCommand("chat_send", "error", s.clock().Sub(start))
		return nil, err
	}
	if created {
		s.publishChatMessage(ctx, cs, matchID, part.TeamSlot, userID, resp)
	}
	outcome := "ok"
	if !created {
		outcome = "replay"
	}
	s.metrics.ObserveChatMessage(outcome)
	s.metrics.ObserveCommand("chat_send", outcome, s.clock().Sub(start))
	return &SendMessageResult{Message: resp, Created: created}, nil
}

func (s *Service) checkChatLimits(ctx context.Context, cs ChatStore, matchID, userID uuid.UUID, hasImage bool, limits ChatInsertLimits) error {
	if n, err := cs.CountAuthorMessagesSince(ctx, matchID, userID, limits.BurstSince); err != nil {
		return err
	} else if n >= limits.BurstLimit {
		return ErrRateLimited
	}
	if n, err := cs.CountAuthorMessagesSince(ctx, matchID, userID, limits.MinuteSince); err != nil {
		return err
	} else if n >= limits.MinuteLimit {
		return ErrRateLimited
	}
	if hasImage {
		if n, err := cs.CountAuthorImageMessages(ctx, matchID, userID); err != nil {
			return err
		} else if n >= limits.MaxImagesInMatch {
			return ErrImageLimitExceeded
		}
	}
	return nil
}

func (s *Service) publishChatMessage(ctx context.Context, cs ChatStore, matchID uuid.UUID, teamSlot int, authorID uuid.UUID, payload TeamMessageResponse) {
	publisher, ok := s.events.(chatEventSink)
	if !ok {
		return
	}
	parts, err := cs.ListTeamParticipants(ctx, matchID)
	if err != nil {
		return
	}
	recipients := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		if part.TeamSlot != teamSlot || !IsActiveParticipant(part.Status) {
			continue
		}
		muted, muteErr := cs.ListMutedUserIDs(ctx, matchID, part.UserID)
		if muteErr != nil {
			continue
		}
		blocked := false
		for _, mutedID := range muted {
			if mutedID == authorID {
				blocked = true
				break
			}
		}
		if !blocked {
			recipients = append(recipients, part.UserID)
		}
	}
	version := int64(0)
	if s.versions != nil {
		version, _ = s.versions.NextVersion(ctx, matchID)
	}
	if err := publisher.PublishChatMessageCreated(ctx, matchID, teamSlot, version, recipients, payload); err != nil {
		s.logger.WarnContext(ctx, "chat event publish failed", slog.Any("error", err))
	}
}

// ListMessages returns viewer-team messages after the given sequence, excluding muted authors.
func (s *Service) ListMessages(ctx context.Context, sess *session.Context, matchID uuid.UUID, afterSeq int64, limit int) (*MessageListResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", "unauthorized", s.clock().Sub(start))
		return nil, err
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveCommand("chat_list", "unavailable", s.clock().Sub(start))
		return nil, ErrUnavailable
	}
	if afterSeq < 0 {
		afterSeq = 0
	}
	if limit < 1 {
		limit = DefaultMessagePageLimit
	}
	if limit > MaxMessagePageLimit {
		limit = MaxMessagePageLimit
	}

	_, part, err := s.authorizeChatAccess(ctx, cs, matchID, userID)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", chatOutcome(err), s.clock().Sub(start))
		return nil, err
	}

	muted, err := cs.ListMutedUserIDs(ctx, matchID, userID)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}

	// Fetch one extra to detect a next page.
	rows, err := cs.ListTeamMessages(ctx, matchID, part.TeamSlot, afterSeq, limit+1, muted)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	fileIDs := make([]uuid.UUID, 0)
	authorIDs := make([]uuid.UUID, 0, len(rows))
	for _, m := range rows {
		authorIDs = append(authorIDs, m.AuthorUserID)
		if m.FileID != nil {
			fileIDs = append(fileIDs, *m.FileID)
		}
	}
	names, err := cs.DisplayNames(ctx, authorIDs)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	files, err := cs.GetChatFiles(ctx, fileIDs)
	if err != nil {
		s.metrics.ObserveCommand("chat_list", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}

	data := make([]TeamMessageResponse, 0, len(rows))
	for _, m := range rows {
		var f *ChatFile
		if m.FileID != nil {
			if ff, ok := files[*m.FileID]; ok {
				cp := ff
				f = &cp
			}
		}
		data = append(data, ToTeamMessageResponse(m, names[m.AuthorUserID], f))
	}

	page := MessagePageInfo{Limit: limit, NextCursor: nil}
	if hasMore && len(rows) > 0 {
		cur := strconv.FormatInt(rows[len(rows)-1].Sequence, 10)
		page.NextCursor = &cur
	}
	s.metrics.ObserveCommand("chat_list", "ok", s.clock().Sub(start))
	return &MessageListResponse{Data: data, Page: page}, nil
}

// GetAttachmentURL returns a short-lived signed URL for a same-team message attachment.
func (s *Service) GetAttachmentURL(ctx context.Context, sess *session.Context, matchID, messageID uuid.UUID) (*AttachmentURLResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveChatAttachment("unauthorized")
		s.metrics.ObserveCommand("chat_attachment", "unauthorized", s.clock().Sub(start))
		return nil, err
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveChatAttachment("unavailable")
		s.metrics.ObserveCommand("chat_attachment", "unavailable", s.clock().Sub(start))
		return nil, ErrUnavailable
	}
	if s.chat.signer == nil {
		s.metrics.ObserveChatAttachment("unavailable")
		s.metrics.ObserveCommand("chat_attachment", "unavailable", s.clock().Sub(start))
		return nil, ErrImageUnavailable
	}

	_, part, err := s.authorizeChatAccess(ctx, cs, matchID, userID)
	if err != nil {
		s.metrics.ObserveChatAttachment(chatOutcome(err))
		s.metrics.ObserveCommand("chat_attachment", chatOutcome(err), s.clock().Sub(start))
		return nil, err
	}

	msg, err := cs.FindMessage(ctx, matchID, messageID)
	if err != nil {
		s.metrics.ObserveChatAttachment("error")
		s.metrics.ObserveCommand("chat_attachment", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if msg == nil || msg.TeamSlot != part.TeamSlot || msg.FileID == nil {
		// Privacy-safe: missing, wrong team, or no attachment all look like not found.
		s.metrics.ObserveChatAttachment("not_found")
		s.metrics.ObserveCommand("chat_attachment", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	// Mute filtering: viewers who muted the author cannot fetch the attachment either.
	muted, err := cs.ListMutedUserIDs(ctx, matchID, userID)
	if err != nil {
		s.metrics.ObserveChatAttachment("error")
		return nil, mapStoreErr(err)
	}
	for _, id := range muted {
		if id == msg.AuthorUserID {
			s.metrics.ObserveChatAttachment("not_found")
			s.metrics.ObserveCommand("chat_attachment", "not_found", s.clock().Sub(start))
			return nil, ErrNotFound
		}
	}

	file, err := cs.GetChatFile(ctx, *msg.FileID)
	if err != nil {
		s.metrics.ObserveChatAttachment("error")
		return nil, mapStoreErr(err)
	}
	if file == nil || file.SanitizationStatus != SanitizationReady || file.StorageKey == "" {
		s.metrics.ObserveChatAttachment("not_ready")
		s.metrics.ObserveCommand("chat_attachment", "not_ready", s.clock().Sub(start))
		return nil, ErrAttachmentNotReady
	}

	ttl := s.chatCfg().AttachmentURLTTL
	url, err := s.chat.signer.PresignedDownloadURL(ctx, file.StorageKey, ttl)
	if err != nil {
		s.logger.WarnContext(ctx, "chat attachment sign failed", slog.Any("error", err))
		s.metrics.ObserveDependencyFailure("storage")
		s.metrics.ObserveChatAttachment("error")
		s.metrics.ObserveCommand("chat_attachment", "error", s.clock().Sub(start))
		return nil, ErrImageUnavailable
	}
	expires := s.clock().Add(ttl)
	s.metrics.ObserveChatAttachment("ok")
	s.metrics.ObserveCommand("chat_attachment", "ok", s.clock().Sub(start))
	return &AttachmentURLResponse{URL: url, ExpiresAt: expires.UTC()}, nil
}

// MuteTeammate idempotently mutes a same-team participant for the viewer.
func (s *Service) MuteTeammate(ctx context.Context, sess *session.Context, matchID, targetUserID uuid.UUID) error {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveChatMute("unauthorized")
		s.metrics.ObserveCommand("chat_mute", "unauthorized", s.clock().Sub(start))
		return err
	}
	if targetUserID == uuid.Nil || targetUserID == userID {
		s.metrics.ObserveChatMute("invalid")
		s.metrics.ObserveCommand("chat_mute", "invalid", s.clock().Sub(start))
		return ErrInvalidRequest
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveChatMute("unavailable")
		return ErrUnavailable
	}
	_, part, err := s.authorizeChatAccess(ctx, cs, matchID, userID)
	if err != nil {
		s.metrics.ObserveChatMute(chatOutcome(err))
		s.metrics.ObserveCommand("chat_mute", chatOutcome(err), s.clock().Sub(start))
		return err
	}
	target, err := cs.FindParticipant(ctx, matchID, targetUserID)
	if err != nil {
		s.metrics.ObserveChatMute("error")
		return mapStoreErr(err)
	}
	if target == nil || target.TeamSlot != part.TeamSlot {
		s.metrics.ObserveChatMute("teammate_required")
		s.metrics.ObserveCommand("chat_mute", "teammate_required", s.clock().Sub(start))
		return ErrTeammateRequired
	}
	if err := cs.UpsertMute(ctx, MatchMute{
		MatchID:     matchID,
		MuterUserID: userID,
		MutedUserID: targetUserID,
		CreatedAt:   s.clock(),
	}); err != nil {
		s.metrics.ObserveChatMute("error")
		s.metrics.ObserveCommand("chat_mute", "error", s.clock().Sub(start))
		return mapStoreErr(err)
	}
	s.metrics.ObserveChatMute("ok")
	s.metrics.ObserveCommand("chat_mute", "ok", s.clock().Sub(start))
	return nil
}

// UnmuteTeammate idempotently removes a mute.
func (s *Service) UnmuteTeammate(ctx context.Context, sess *session.Context, matchID, targetUserID uuid.UUID) error {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveChatMute("unauthorized")
		s.metrics.ObserveCommand("chat_unmute", "unauthorized", s.clock().Sub(start))
		return err
	}
	if targetUserID == uuid.Nil {
		s.metrics.ObserveChatMute("invalid")
		return ErrInvalidRequest
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveChatMute("unavailable")
		return ErrUnavailable
	}
	// Chat access window still required to toggle mutes.
	if _, _, err := s.authorizeChatAccess(ctx, cs, matchID, userID); err != nil {
		s.metrics.ObserveChatMute(chatOutcome(err))
		s.metrics.ObserveCommand("chat_unmute", chatOutcome(err), s.clock().Sub(start))
		return err
	}
	if err := cs.DeleteMute(ctx, matchID, userID, targetUserID); err != nil {
		s.metrics.ObserveChatMute("error")
		s.metrics.ObserveCommand("chat_unmute", "error", s.clock().Sub(start))
		return mapStoreErr(err)
	}
	s.metrics.ObserveChatMute("ok")
	s.metrics.ObserveCommand("chat_unmute", "ok", s.clock().Sub(start))
	return nil
}

// ReportMessage records a moderation report. Idempotent per reporter/message.
// The target author is not notified; response is always privacy-safe 202 semantics.
func (s *Service) ReportMessage(ctx context.Context, sess *session.Context, matchID, messageID uuid.UUID, req ReportMessageRequest) (*ReportMessageResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveChatReport("unauthorized")
		s.metrics.ObserveCommand("chat_report", "unauthorized", s.clock().Sub(start))
		return nil, err
	}
	reason := strings.TrimSpace(strings.ToLower(req.Reason))
	if !ValidReportReason(reason) {
		s.metrics.ObserveChatReport("invalid")
		s.metrics.ObserveCommand("chat_report", "invalid", s.clock().Sub(start))
		return nil, ErrInvalidRequest
	}
	cs := s.chatStore()
	if cs == nil {
		s.metrics.ObserveChatReport("unavailable")
		return nil, ErrUnavailable
	}

	_, part, err := s.authorizeChatAccess(ctx, cs, matchID, userID)
	if err != nil {
		s.metrics.ObserveChatReport(chatOutcome(err))
		s.metrics.ObserveCommand("chat_report", chatOutcome(err), s.clock().Sub(start))
		return nil, err
	}

	msg, err := cs.FindMessage(ctx, matchID, messageID)
	if err != nil {
		s.metrics.ObserveChatReport("error")
		return nil, mapStoreErr(err)
	}
	// Privacy-safe: wrong team / missing look identical; self-report rejected as invalid.
	if msg == nil || msg.TeamSlot != part.TeamSlot {
		s.metrics.ObserveChatReport("not_found")
		s.metrics.ObserveCommand("chat_report", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}
	if msg.AuthorUserID == userID {
		s.metrics.ObserveChatReport("invalid")
		s.metrics.ObserveCommand("chat_report", "invalid", s.clock().Sub(start))
		return nil, ErrInvalidRequest
	}

	cfg := s.chatCfg()
	now := s.clock()
	retentionUntil := ReportRetentionUntil(now, cfg.ReportRetentionDays)
	report := MessageReport{
		ID:             uuid.New(),
		MessageID:      messageID,
		MatchID:        matchID,
		ReporterUserID: userID,
		Reason:         reason,
		Status:         ReportStatusPending,
		CreatedAt:      now,
		RetentionUntil: retentionUntil,
		LegalHold:      false,
	}
	_, created, err := cs.InsertReport(ctx, report, retentionUntil)
	if err != nil {
		s.metrics.ObserveChatReport(chatOutcome(err))
		s.metrics.ObserveCommand("chat_report", chatOutcome(err), s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	outcome := "ok"
	if !created {
		outcome = "replay"
	}
	s.metrics.ObserveChatReport(outcome)
	s.metrics.ObserveCommand("chat_report", outcome, s.clock().Sub(start))
	// Always return accepted status without leaking prior report state details.
	return &ReportMessageResponse{Status: "accepted"}, nil
}

// authorizeChatAccess ensures the caller is a match participant on a Duo/Squad match
// during the active match or 15-minute post-terminal chat window.
// Non-participants receive privacy-safe ErrNotFound.
func (s *Service) authorizeChatAccess(ctx context.Context, cs ChatStore, matchID, userID uuid.UUID) (*Match, *MatchParticipant, error) {
	match, err := cs.LoadMatchForChat(ctx, matchID)
	if err != nil {
		return nil, nil, mapStoreErr(err)
	}
	if match == nil {
		return nil, nil, ErrNotFound
	}
	part, err := cs.FindParticipant(ctx, matchID, userID)
	if err != nil {
		return nil, nil, mapStoreErr(err)
	}
	if part == nil {
		return nil, nil, ErrNotFound
	}
	if !canChat(*match, s.clock()) {
		return nil, nil, ErrChatUnavailable
	}
	return match, part, nil
}

func validateChatAttachment(f *ChatFile, matchID, ownerID uuid.UUID) error {
	if f == nil {
		return ErrNotFound
	}
	if f.Purpose != FilePurposeTeamChat {
		return ErrNotFound
	}
	if f.ContextID == nil || *f.ContextID != matchID {
		return ErrNotFound
	}
	if f.OwnerUserID != ownerID {
		// Privacy-safe: do not reveal other users' files.
		return ErrNotFound
	}
	if f.SanitizationStatus != SanitizationReady {
		return ErrAttachmentNotReady
	}
	return nil
}

func (s *Service) projectMessage(ctx context.Context, cs ChatStore, msg TeamMessage) (TeamMessageResponse, error) {
	names, err := cs.DisplayNames(ctx, []uuid.UUID{msg.AuthorUserID})
	if err != nil {
		return TeamMessageResponse{}, mapStoreErr(err)
	}
	var file *ChatFile
	if msg.FileID != nil {
		f, err := cs.GetChatFile(ctx, *msg.FileID)
		if err != nil {
			return TeamMessageResponse{}, mapStoreErr(err)
		}
		file = f
	}
	return ToTeamMessageResponse(msg, names[msg.AuthorUserID], file), nil
}

func chatOutcome(err error) string {
	switch err {
	case nil:
		return "ok"
	case ErrUnauthorized:
		return "unauthorized"
	case ErrNotFound, ErrForbiddenOpponent:
		return "not_found"
	case ErrChatUnavailable:
		return "chat_unavailable"
	case ErrMessageTooLong:
		return "too_long"
	case ErrAttachmentNotReady:
		return "not_ready"
	case ErrAttachmentAlreadyBound:
		return "already_bound"
	case ErrImageUnavailable:
		return "image_unavailable"
	case ErrImageLimitExceeded:
		return "image_limit"
	case ErrRateLimited:
		return "rate_limited"
	case ErrTeammateRequired:
		return "teammate_required"
	case ErrInvalidRequest:
		return "invalid"
	case ErrUnavailable:
		return "unavailable"
	default:
		return "error"
	}
}
