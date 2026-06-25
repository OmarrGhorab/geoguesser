package uploads

import (
	"time"

	"github.com/google/uuid"
)

// Upload represents a pending file upload.
type Upload struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OwnerUserID uuid.UUID `gorm:"type:uuid;not null;index:uploads_owner_status"`
	FileName    string    `gorm:"type:text;not null"`
	ContentType string    `gorm:"type:text;not null"`
	SizeBytes   int64     `gorm:"type:bigint;not null"`
	StorageKey  string    `gorm:"type:text;not null"`
	Status      string    `gorm:"type:text;not null;default:'pending'"`
	ExpiresAt   time.Time `gorm:"type:timestamptz;not null"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name.
func (Upload) TableName() string {
	return "uploads"
}

// File represents a completed file.
type File struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UploadID    *uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	OwnerUserID uuid.UUID `gorm:"type:uuid;not null;index:files_owner_created_at"`
	FileName    string    `gorm:"type:text;not null"`
	ContentType string    `gorm:"type:text;not null"`
	SizeBytes   int64     `gorm:"type:bigint;not null"`
	StorageKey  string    `gorm:"type:text;not null;uniqueIndex"`
	IsPublic    bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name.
func (File) TableName() string {
	return "files"
}
