package matchplay

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChatStore is the durable chat/moderation persistence surface.
// Implementations may be PostgreSQL-backed or in-memory fakes for unit tests.
type ChatStore interface {
	// LoadMatchForChat returns the match row or nil when missing.
	LoadMatchForChat(ctx context.Context, matchID uuid.UUID) (*Match, error)

	// FindParticipant returns the caller's participant row, or nil when not on the match.
	FindParticipant(ctx context.Context, matchID, userID uuid.UUID) (*MatchParticipant, error)

	// ListTeamParticipants returns all participants on a match (for teammate checks).
	ListTeamParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error)

	// FindMessageByClientID returns an existing message for idempotent replay.
	FindMessageByClientID(ctx context.Context, matchID, authorID, clientMessageID uuid.UUID) (*TeamMessage, error)

	// FindMessage returns a message by id within a match, or nil when missing.
	FindMessage(ctx context.Context, matchID, messageID uuid.UUID) (*TeamMessage, error)

	// InsertMessage persists a new message. Returns the stored row (with sequence),
	// whether it was newly created, and any error. Unique client_message_id conflicts
	// reload and return the existing row with created=false. Unique file_id conflicts
	// return ErrAttachmentAlreadyBound.
	InsertMessage(ctx context.Context, msg TeamMessage) (*TeamMessage, bool, error)

	// ListTeamMessages returns same-team messages with sequence > after, ascending.
	// excludeAuthorIDs filters muted authors. limit is applied after filtering.
	ListTeamMessages(ctx context.Context, matchID uuid.UUID, teamSlot int, afterSeq int64, limit int, excludeAuthorIDs []uuid.UUID) ([]TeamMessage, error)

	// GetChatFile returns a files row or nil when missing.
	GetChatFile(ctx context.Context, fileID uuid.UUID) (*ChatFile, error)

	// GetChatFiles returns files keyed by id for the provided ids.
	GetChatFiles(ctx context.Context, fileIDs []uuid.UUID) (map[uuid.UUID]ChatFile, error)

	// CountAuthorMessagesSince counts messages authored by user in the match since t.
	CountAuthorMessagesSince(ctx context.Context, matchID, authorID uuid.UUID, since time.Time) (int, error)

	// CountAuthorImageMessages counts messages with a file attachment by author in the match.
	CountAuthorImageMessages(ctx context.Context, matchID, authorID uuid.UUID) (int, error)

	// DisplayNames returns display names for user ids (missing keys get empty string).
	DisplayNames(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]string, error)

	// UpsertMute inserts a mute row idempotently.
	UpsertMute(ctx context.Context, mute MatchMute) error

	// DeleteMute removes a mute row (idempotent when absent).
	DeleteMute(ctx context.Context, matchID, muterID, mutedID uuid.UUID) error

	// ListMutedUserIDs returns user ids the muter has muted in the match.
	ListMutedUserIDs(ctx context.Context, matchID, muterID uuid.UUID) ([]uuid.UUID, error)

	// InsertReport creates a report. Unique reporter/message conflicts reload with created=false.
	// On first create, the message status is set to reported and expires_at is extended to retention.
	InsertReport(ctx context.Context, report MessageReport, extendExpiresTo time.Time) (*MessageReport, bool, error)

	// ListCleanupCandidates returns expired unreported messages eligible for deletion.
	// Skips messages protected by an active report retention window or legal hold.
	ListCleanupCandidates(ctx context.Context, now time.Time, limit int) ([]TeamMessage, error)

	// ListProtectedMessageIDs returns message ids that must not be deleted due to
	// legal hold or retention_until still in the future.
	ListProtectedMessageIDs(ctx context.Context, messageIDs []uuid.UUID, now time.Time) (map[uuid.UUID]struct{}, error)

	// DeleteMessages removes messages, their reports, and returns attached file ids.
	DeleteMessages(ctx context.Context, messageIDs []uuid.UUID) (fileIDs []uuid.UUID, err error)

	// ListRawUploadsForCleanup returns team-chat uploads eligible for raw cleanup
	// (pending/failed/rejected, older than cutoff).
	ListRawUploadsForCleanup(ctx context.Context, cutoff time.Time, limit int) ([]ChatUpload, error)

	// DeleteUploads removes upload rows by id.
	DeleteUploads(ctx context.Context, uploadIDs []uuid.UUID) error
	ClearUploadRawStorageKeys(ctx context.Context, uploadIDs []uuid.UUID) error

	// DeleteFiles removes file rows by id (after storage objects are deleted).
	DeleteFiles(ctx context.Context, fileIDs []uuid.UUID) error

	// DeleteMutesForMatches removes mute rows for the given matches (optional chat cleanup).
	DeleteMutesForMatches(ctx context.Context, matchIDs []uuid.UUID) error

	// TouchActivity updates last_activity_at for a match.
	TouchActivity(ctx context.Context, matchID uuid.UUID, at time.Time) error
}

// ChatInsertLimits are enforced in the same transaction as the insert.
type ChatInsertLimits struct {
	BurstSince       time.Time
	BurstLimit       int
	MinuteSince      time.Time
	MinuteLimit      int
	MaxImagesInMatch int
}

type atomicChatStore interface {
	InsertMessageWithLimits(ctx context.Context, msg TeamMessage, limits ChatInsertLimits) (*TeamMessage, bool, error)
}

// LoadMatchForChat implements ChatStore.
func (r *Repository) LoadMatchForChat(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var match Match
	if err := r.db.WithContext(ctx).Where("id = ?", matchID).Take(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match for chat: %w", err)
	}
	return &match, nil
}

// ListTeamParticipants implements ChatStore.
func (r *Repository) ListTeamParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var parts []MatchParticipant
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).
		Order("team_slot ASC, assigned_at ASC, user_id ASC").
		Find(&parts).Error; err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return parts, nil
}

// FindMessageByClientID implements ChatStore.
func (r *Repository) FindMessageByClientID(ctx context.Context, matchID, authorID, clientMessageID uuid.UUID) (*TeamMessage, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var msg TeamMessage
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND author_user_id = ? AND client_message_id = ?", matchID, authorID, clientMessageID).
		Take(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find message by client id: %w", err)
	}
	return &msg, nil
}

// FindMessage implements ChatStore.
func (r *Repository) FindMessage(ctx context.Context, matchID, messageID uuid.UUID) (*TeamMessage, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var msg TeamMessage
	err := r.db.WithContext(ctx).
		Where("id = ? AND match_id = ?", messageID, matchID).
		Take(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find message: %w", err)
	}
	return &msg, nil
}

// InsertMessage implements ChatStore.
func (r *Repository) InsertMessage(ctx context.Context, msg TeamMessage) (*TeamMessage, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, ErrUnavailable
	}
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.Status == "" {
		msg.Status = MessageStatusActive
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}

	// Sequence is BIGSERIAL — omit so Postgres assigns it.
	err := r.db.WithContext(ctx).
		Omit("sequence").
		Create(&msg).Error
	if err != nil {
		if isChatUniqueViolation(err) {
			// Prefer client_message_id idempotent replay.
			existing, findErr := r.FindMessageByClientID(ctx, msg.MatchID, msg.AuthorUserID, msg.ClientMessageID)
			if findErr != nil {
				return nil, false, findErr
			}
			if existing != nil {
				return existing, false, nil
			}
			// File already bound to another message.
			if msg.FileID != nil {
				return nil, false, ErrAttachmentAlreadyBound
			}
			return nil, false, ErrIdempotencyConflict
		}
		return nil, false, fmt.Errorf("insert message: %w", err)
	}

	// Reload to capture database-assigned sequence / defaults.
	var stored TeamMessage
	if err := r.db.WithContext(ctx).Where("id = ?", msg.ID).Take(&stored).Error; err != nil {
		return nil, false, fmt.Errorf("reload message: %w", err)
	}
	return &stored, true, nil
}

// InsertMessageWithLimits serializes one author's quota check and insert. The
// advisory transaction lock makes limits authoritative across API instances.
func (r *Repository) InsertMessageWithLimits(ctx context.Context, msg TeamMessage, limits ChatInsertLimits) (*TeamMessage, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, ErrUnavailable
	}
	var stored *TeamMessage
	var created bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			lockKey := msg.MatchID.String() + "|" + msg.AuthorUserID.String()
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey).Error; err != nil {
				return fmt.Errorf("lock chat quota: %w", err)
			}
		}
		txRepo := &Repository{db: tx}
		existing, err := txRepo.FindMessageByClientID(ctx, msg.MatchID, msg.AuthorUserID, msg.ClientMessageID)
		if err != nil {
			return err
		}
		if existing != nil {
			stored, created = existing, false
			return nil
		}
		burst, err := txRepo.CountAuthorMessagesSince(ctx, msg.MatchID, msg.AuthorUserID, limits.BurstSince)
		if err != nil {
			return err
		}
		if burst >= limits.BurstLimit {
			return ErrRateLimited
		}
		minute, err := txRepo.CountAuthorMessagesSince(ctx, msg.MatchID, msg.AuthorUserID, limits.MinuteSince)
		if err != nil {
			return err
		}
		if minute >= limits.MinuteLimit {
			return ErrRateLimited
		}
		if msg.FileID != nil {
			images, err := txRepo.CountAuthorImageMessages(ctx, msg.MatchID, msg.AuthorUserID)
			if err != nil {
				return err
			}
			if images >= limits.MaxImagesInMatch {
				return ErrImageLimitExceeded
			}
		}
		stored, created, err = txRepo.InsertMessage(ctx, msg)
		return err
	})
	return stored, created, err
}

// ListTeamMessages implements ChatStore.
func (r *Repository) ListTeamMessages(
	ctx context.Context,
	matchID uuid.UUID,
	teamSlot int,
	afterSeq int64,
	limit int,
	excludeAuthorIDs []uuid.UUID,
) ([]TeamMessage, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit < 1 {
		limit = DefaultMessagePageLimit
	}
	q := r.db.WithContext(ctx).
		Where("match_id = ? AND team_slot = ? AND sequence > ?", matchID, teamSlot, afterSeq).
		Where("status IN ?", []string{MessageStatusActive, MessageStatusReported})
	if len(excludeAuthorIDs) > 0 {
		q = q.Where("author_user_id NOT IN ?", excludeAuthorIDs)
	}
	var rows []TeamMessage
	if err := q.Order("sequence ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list team messages: %w", err)
	}
	return rows, nil
}

// GetChatFile implements ChatStore.
func (r *Repository) GetChatFile(ctx context.Context, fileID uuid.UUID) (*ChatFile, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var f ChatFile
	if err := r.db.WithContext(ctx).Where("id = ?", fileID).Take(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get chat file: %w", err)
	}
	return &f, nil
}

// GetChatFiles implements ChatStore.
func (r *Repository) GetChatFiles(ctx context.Context, fileIDs []uuid.UUID) (map[uuid.UUID]ChatFile, error) {
	out := make(map[uuid.UUID]ChatFile, len(fileIDs))
	if len(fileIDs) == 0 {
		return out, nil
	}
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var rows []ChatFile
	if err := r.db.WithContext(ctx).Where("id IN ?", fileIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get chat files: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// CountAuthorMessagesSince implements ChatStore.
func (r *Repository) CountAuthorMessagesSince(ctx context.Context, matchID, authorID uuid.UUID, since time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, ErrUnavailable
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TeamMessage{}).
		Where("match_id = ? AND author_user_id = ? AND created_at >= ?", matchID, authorID, since).
		Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count messages since: %w", err)
	}
	return int(n), nil
}

// CountAuthorImageMessages implements ChatStore.
func (r *Repository) CountAuthorImageMessages(ctx context.Context, matchID, authorID uuid.UUID) (int, error) {
	if r == nil || r.db == nil {
		return 0, ErrUnavailable
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TeamMessage{}).
		Where("match_id = ? AND author_user_id = ? AND file_id IS NOT NULL", matchID, authorID).
		Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count image messages: %w", err)
	}
	return int(n), nil
}

// DisplayNames implements ChatStore.
func (r *Repository) DisplayNames(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	type row struct {
		UserID      uuid.UUID `gorm:"column:user_id"`
		DisplayName string    `gorm:"column:display_name"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("user_profiles").
		Select("user_id, display_name").
		Where("user_id IN ?", userIDs).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load display names: %w", err)
	}
	for _, rrow := range rows {
		out[rrow.UserID] = rrow.DisplayName
	}
	return out, nil
}

// UpsertMute implements ChatStore.
func (r *Repository) UpsertMute(ctx context.Context, mute MatchMute) error {
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if mute.CreatedAt.IsZero() {
		mute.CreatedAt = time.Now().UTC()
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "match_id"}, {Name: "muter_user_id"}, {Name: "muted_user_id"}},
		DoNothing: true,
	}).Create(&mute).Error
	if err != nil {
		return fmt.Errorf("upsert mute: %w", err)
	}
	return nil
}

// DeleteMute implements ChatStore.
func (r *Repository) DeleteMute(ctx context.Context, matchID, muterID, mutedID uuid.UUID) error {
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if err := r.db.WithContext(ctx).
		Where("match_id = ? AND muter_user_id = ? AND muted_user_id = ?", matchID, muterID, mutedID).
		Delete(&MatchMute{}).Error; err != nil {
		return fmt.Errorf("delete mute: %w", err)
	}
	return nil
}

// ListMutedUserIDs implements ChatStore.
func (r *Repository) ListMutedUserIDs(ctx context.Context, matchID, muterID uuid.UUID) ([]uuid.UUID, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&MatchMute{}).
		Where("match_id = ? AND muter_user_id = ?", matchID, muterID).
		Pluck("muted_user_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list mutes: %w", err)
	}
	return ids, nil
}

// InsertReport implements ChatStore.
func (r *Repository) InsertReport(ctx context.Context, report MessageReport, extendExpiresTo time.Time) (*MessageReport, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, ErrUnavailable
	}
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	if report.Status == "" {
		report.Status = ReportStatusPending
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}

	var stored MessageReport
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the message row so status/retention updates stay consistent.
		var msg TeamMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND match_id = ?", report.MessageID, report.MatchID).
			Take(&msg).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("lock message for report: %w", err)
		}

		err := tx.Create(&report).Error
		if err != nil {
			if isChatUniqueViolation(err) {
				var existing MessageReport
				if findErr := tx.Where("message_id = ? AND reporter_user_id = ?", report.MessageID, report.ReporterUserID).
					Take(&existing).Error; findErr != nil {
					return findErr
				}
				stored = existing
				created = false
				return nil
			}
			return fmt.Errorf("insert report: %w", err)
		}
		created = true
		stored = report

		updates := map[string]any{
			"status": MessageStatusReported,
		}
		if extendExpiresTo.After(msg.ExpiresAt) {
			updates["expires_at"] = extendExpiresTo
		}
		if err := tx.Model(&TeamMessage{}).Where("id = ?", msg.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("mark message reported: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &stored, created, nil
}

// ListCleanupCandidates implements ChatStore.
func (r *Repository) ListCleanupCandidates(ctx context.Context, now time.Time, limit int) ([]TeamMessage, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit < 1 {
		limit = DefaultCleanupBatchSize
	}
	// Expired messages not protected by legal hold or active report retention.
	var rows []TeamMessage
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.*
		FROM team_messages m
		WHERE m.expires_at < ?
		  AND NOT EXISTS (
		    SELECT 1 FROM team_message_reports r
		    WHERE r.message_id = m.id
		      AND (r.legal_hold = true OR r.retention_until > ?)
		  )
		ORDER BY m.expires_at ASC
		LIMIT ?
	`, now, now, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list cleanup candidates: %w", err)
	}
	return rows, nil
}

// ListProtectedMessageIDs implements ChatStore.
func (r *Repository) ListProtectedMessageIDs(ctx context.Context, messageIDs []uuid.UUID, now time.Time) (map[uuid.UUID]struct{}, error) {
	out := make(map[uuid.UUID]struct{})
	if len(messageIDs) == 0 {
		return out, nil
	}
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&MessageReport{}).
		Where("message_id IN ? AND (legal_hold = true OR retention_until > ?)", messageIDs, now).
		Distinct("message_id").
		Pluck("message_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list protected messages: %w", err)
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

// DeleteMessages implements ChatStore.
func (r *Repository) DeleteMessages(ctx context.Context, messageIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	var fileIDs []uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&TeamMessage{}).
			Where("id IN ? AND file_id IS NOT NULL", messageIDs).
			Pluck("file_id", &fileIDs).Error; err != nil {
			return fmt.Errorf("pluck file ids: %w", err)
		}
		if err := tx.Where("message_id IN ?", messageIDs).Delete(&MessageReport{}).Error; err != nil {
			return fmt.Errorf("delete reports: %w", err)
		}
		if err := tx.Where("id IN ?", messageIDs).Delete(&TeamMessage{}).Error; err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileIDs, nil
}

// ListRawUploadsForCleanup implements ChatStore.
func (r *Repository) ListRawUploadsForCleanup(ctx context.Context, cutoff time.Time, limit int) ([]ChatUpload, error) {
	if r == nil || r.db == nil {
		return nil, ErrUnavailable
	}
	if limit < 1 {
		limit = DefaultCleanupBatchSize
	}
	var rows []ChatUpload
	// Pending/failed/rejected team-chat uploads past the raw retention window.
	// Also catch any row still holding a raw_storage_key past cutoff.
	err := r.db.WithContext(ctx).
		Where("purpose = ? AND created_at < ?", FilePurposeTeamChat, cutoff).
		Where(
			"status IN ? OR sanitization_status IN ? OR raw_storage_key IS NOT NULL",
			[]string{UploadStatusPending, UploadStatusFailed, UploadStatusExpired},
			[]string{SanitizationPending, SanitizationRejected},
		).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list raw uploads: %w", err)
	}
	return rows, nil
}

// DeleteUploads implements ChatStore.
func (r *Repository) DeleteUploads(ctx context.Context, uploadIDs []uuid.UUID) error {
	if len(uploadIDs) == 0 {
		return nil
	}
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", uploadIDs).Delete(&ChatUpload{}).Error; err != nil {
		return fmt.Errorf("delete uploads: %w", err)
	}
	return nil
}

// ClearUploadRawStorageKeys retains completed upload/file metadata after raw cleanup.
func (r *Repository) ClearUploadRawStorageKeys(ctx context.Context, uploadIDs []uuid.UUID) error {
	if len(uploadIDs) == 0 {
		return nil
	}
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if err := r.db.WithContext(ctx).Model(&ChatUpload{}).Where("id IN ?", uploadIDs).Update("raw_storage_key", nil).Error; err != nil {
		return fmt.Errorf("clear raw upload keys: %w", err)
	}
	return nil
}

// DeleteFiles implements ChatStore.
func (r *Repository) DeleteFiles(ctx context.Context, fileIDs []uuid.UUID) error {
	if len(fileIDs) == 0 {
		return nil
	}
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", fileIDs).Delete(&ChatFile{}).Error; err != nil {
		return fmt.Errorf("delete files: %w", err)
	}
	return nil
}

// DeleteMutesForMatches implements ChatStore.
func (r *Repository) DeleteMutesForMatches(ctx context.Context, matchIDs []uuid.UUID) error {
	if len(matchIDs) == 0 {
		return nil
	}
	if r == nil || r.db == nil {
		return ErrUnavailable
	}
	if err := r.db.WithContext(ctx).Where("match_id IN ?", matchIDs).Delete(&MatchMute{}).Error; err != nil {
		return fmt.Errorf("delete mutes: %w", err)
	}
	return nil
}

func isChatUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique constraint")
}
