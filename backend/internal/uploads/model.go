package uploads

import (
	"time"

	"github.com/google/uuid"
)

// Purpose and sanitization status constants (match migration 00019).
const (
	PurposeGeneral  = "general"
	PurposeTeamChat = "team_chat"

	SanitizationNotRequired = "not_required"
	SanitizationPending     = "pending"
	SanitizationReady       = "ready"
	SanitizationRejected    = "rejected"

	uploadStatusPending   = "pending"
	uploadStatusCompleted = "completed"
	uploadStatusExpired   = "expired"
	uploadStatusRejected  = "rejected"
)

// Upload represents a pending (or completed/rejected) file upload.
type Upload struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OwnerUserID        uuid.UUID  `gorm:"type:uuid;not null;index:uploads_owner_status"`
	FileName           string     `gorm:"type:text;not null"`
	ContentType        string     `gorm:"type:text;not null"`
	SizeBytes          int64      `gorm:"type:bigint;not null"`
	StorageKey         string     `gorm:"type:text;not null"`
	Status             string     `gorm:"type:text;not null;default:'pending'"`
	Purpose            string     `gorm:"type:text;not null;default:'general'"`
	ContextID          *uuid.UUID `gorm:"type:uuid"`
	SanitizationStatus string     `gorm:"type:text;not null;default:'not_required'"`
	RawStorageKey      *string    `gorm:"type:text"`
	ExpiresAt          time.Time  `gorm:"type:timestamptz;not null"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name.
func (Upload) TableName() string {
	return "uploads"
}

// File represents a completed file.
// For team_chat purpose, StorageKey always points at the sanitized derivative
// and SanitizationStatus must be "ready" before the file can be attached.
// RawStorageKey is cleared after successful sanitization cleanup.
type File struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UploadID           *uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	OwnerUserID        uuid.UUID  `gorm:"type:uuid;not null;index:files_owner_created_at"`
	FileName           string     `gorm:"type:text;not null"`
	ContentType        string     `gorm:"type:text;not null"`
	SizeBytes          int64      `gorm:"type:bigint;not null"`
	StorageKey         string     `gorm:"type:text;not null;uniqueIndex"`
	IsPublic           bool       `gorm:"type:boolean;not null;default:false"`
	Purpose            string     `gorm:"type:text;not null;default:'general'"`
	ContextID          *uuid.UUID `gorm:"type:uuid"`
	SanitizationStatus string     `gorm:"type:text;not null;default:'not_required'"`
	RawStorageKey      *string    `gorm:"type:text"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name.
func (File) TableName() string {
	return "files"
}

// IsReadyTeamChat reports whether the file may be attached to a team-chat message.
func (f *File) IsReadyTeamChat() bool {
	if f == nil {
		return false
	}
	return f.Purpose == PurposeTeamChat && f.SanitizationStatus == SanitizationReady
}
