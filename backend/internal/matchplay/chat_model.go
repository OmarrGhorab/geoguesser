package matchplay

import (
	"time"

	"github.com/google/uuid"
)

// Team message status values (team_messages.status).
const (
	MessageStatusActive   = "active"
	MessageStatusReported = "reported"
	MessageStatusRemoved  = "removed"
	MessageStatusExpired  = "expired"
)

// Report reason codes (team_message_reports.reason).
const (
	ReportReasonHarassment = "harassment"
	ReportReasonHate       = "hate"
	ReportReasonSexual     = "sexual"
	ReportReasonViolent    = "violent"
	ReportReasonSpam       = "spam"
	ReportReasonOther      = "other"
)

// Report status values (team_message_reports.status).
const (
	ReportStatusPending   = "pending"
	ReportStatusReviewed  = "reviewed"
	ReportStatusDismissed = "dismissed"
	ReportStatusActioned  = "actioned"
)

// Team-chat file purpose / sanitization values (files / uploads extensions).
const (
	FilePurposeTeamChat     = "team_chat"
	FilePurposeGeneral      = "general"
	SanitizationNotRequired = "not_required"
	SanitizationPending     = "pending"
	SanitizationReady       = "ready"
	SanitizationRejected    = "rejected"
	UploadStatusPending     = "pending"
	UploadStatusCompleted   = "completed"
	UploadStatusFailed      = "failed"
	UploadStatusExpired     = "expired"
)

// Chat validation and retention defaults.
const (
	MaxMessageTextLength        = 500
	DefaultMessagePageLimit     = 50
	MaxMessagePageLimit         = 100
	MaxImagesPerUserPerMatch    = 5
	MessageBurstLimit           = 10
	MessageBurstWindow          = 10 * time.Second
	MessageMinuteLimit          = 60
	MessageMinuteWindow         = time.Minute
	AttachmentSignedURLTTL      = 5 * time.Minute
	DefaultChatRetentionDays    = 30
	DefaultReportRetentionDays  = 180
	RawUploadRetention          = 24 * time.Hour
	DefaultCleanupBatchSize     = 100
	DefaultCleanupDeleteRetries = 3
)

// TeamMessage is a durable team-chat message (team_messages).
type TeamMessage struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MatchID         uuid.UUID  `gorm:"type:uuid;not null"`
	TeamSlot        int        `gorm:"type:smallint;not null"`
	AuthorUserID    uuid.UUID  `gorm:"type:uuid;not null"`
	ClientMessageID uuid.UUID  `gorm:"type:uuid;not null"`
	Sequence        int64      `gorm:"type:bigint;not null"`
	Text            string     `gorm:"type:text;not null;default:''"`
	FileID          *uuid.UUID `gorm:"type:uuid"`
	Status          string     `gorm:"type:text;not null;default:'active'"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	ExpiresAt       time.Time  `gorm:"type:timestamptz;not null"`
}

// TableName returns the database table name.
func (TeamMessage) TableName() string { return "team_messages" }

// MatchMute is a per-viewer mute of a teammate for one match (match_mutes).
type MatchMute struct {
	MatchID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	MuterUserID uuid.UUID `gorm:"type:uuid;primaryKey"`
	MutedUserID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (MatchMute) TableName() string { return "match_mutes" }

// MessageReport is a moderation report against a team message (team_message_reports).
type MessageReport struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MessageID      uuid.UUID  `gorm:"type:uuid;not null"`
	MatchID        uuid.UUID  `gorm:"type:uuid;not null"`
	ReporterUserID uuid.UUID  `gorm:"type:uuid;not null"`
	Reason         string     `gorm:"type:text;not null"`
	Status         string     `gorm:"type:text;not null;default:'pending'"`
	CreatedAt      time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	ResolvedAt     *time.Time `gorm:"type:timestamptz"`
	RetentionUntil time.Time  `gorm:"type:timestamptz;not null"`
	LegalHold      bool       `gorm:"type:boolean;not null;default:false"`
}

// TableName returns the database table name.
func (MessageReport) TableName() string { return "team_message_reports" }

// ChatFile is the files projection needed for attachment validation and signed URLs.
// Never expose StorageKey or RawStorageKey in API responses.
type ChatFile struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID        uuid.UUID  `gorm:"type:uuid;not null"`
	FileName           string     `gorm:"type:text;not null"`
	ContentType        string     `gorm:"type:text;not null"`
	SizeBytes          int64      `gorm:"type:bigint;not null"`
	StorageKey         string     `gorm:"type:text;not null"`
	Purpose            string     `gorm:"type:text;not null"`
	ContextID          *uuid.UUID `gorm:"type:uuid"`
	SanitizationStatus string     `gorm:"type:text;not null"`
	RawStorageKey      *string    `gorm:"type:text"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null"`
}

// TableName returns the database table name.
func (ChatFile) TableName() string { return "files" }

// ChatUpload is the uploads projection for raw-object retention cleanup.
type ChatUpload struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key"`
	OwnerUserID        uuid.UUID  `gorm:"type:uuid;not null"`
	StorageKey         string     `gorm:"type:text;not null"`
	Status             string     `gorm:"type:text;not null"`
	Purpose            string     `gorm:"type:text;not null"`
	ContextID          *uuid.UUID `gorm:"type:uuid"`
	SanitizationStatus string     `gorm:"type:text;not null"`
	RawStorageKey      *string    `gorm:"type:text"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null"`
	ExpiresAt          time.Time  `gorm:"type:timestamptz;not null"`
}

// TableName returns the database table name.
func (ChatUpload) TableName() string { return "uploads" }

// ValidReportReason reports whether reason is an allowed report code.
func ValidReportReason(reason string) bool {
	switch reason {
	case ReportReasonHarassment, ReportReasonHate, ReportReasonSexual,
		ReportReasonViolent, ReportReasonSpam, ReportReasonOther:
		return true
	default:
		return false
	}
}

// MessageExpiresAt computes the normal unreported retention deadline.
// Prefer match completion + retention when known; otherwise now + retention.
func MessageExpiresAt(now time.Time, completedAt *time.Time, retentionDays int) time.Time {
	if retentionDays < 1 {
		retentionDays = DefaultChatRetentionDays
	}
	d := time.Duration(retentionDays) * 24 * time.Hour
	if completedAt != nil && !completedAt.IsZero() {
		return completedAt.UTC().Add(d)
	}
	return now.UTC().Add(d)
}

// ReportRetentionUntil computes the minimum retention deadline from a report time.
func ReportRetentionUntil(reportedAt time.Time, retentionDays int) time.Time {
	if retentionDays < 1 {
		retentionDays = DefaultReportRetentionDays
	}
	return reportedAt.UTC().Add(time.Duration(retentionDays) * 24 * time.Hour)
}
