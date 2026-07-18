package uploads

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository owns upload/file persistence queries.
type Repository struct {
	db *gorm.DB
}

// NewRepository returns a new uploads repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateUpload inserts a pending upload.
func (r *Repository) CreateUpload(ctx context.Context, upload *Upload) error {
	if err := r.db.WithContext(ctx).Create(upload).Error; err != nil {
		return fmt.Errorf("failed to create upload: %w", err)
	}
	return nil
}

// GetUploadByID returns an upload by id.
func (r *Repository) GetUploadByID(ctx context.Context, id uuid.UUID) (*Upload, error) {
	var upload Upload
	if err := r.db.WithContext(ctx).First(&upload, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}
	return &upload, nil
}

// MarkUploadCompleted marks an upload as completed.
func (r *Repository) MarkUploadCompleted(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&Upload{}).Where("id = ?", id).Updates(map[string]any{
		"status":              uploadStatusCompleted,
		"sanitization_status": SanitizationNotRequired,
		"raw_storage_key":     nil,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to mark upload completed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

// MarkUploadCompletedSanitized marks a team-chat upload completed after sanitization.
func (r *Repository) MarkUploadCompletedSanitized(ctx context.Context, id uuid.UUID, derivativeKey string, sizeBytes int64) error {
	result := r.db.WithContext(ctx).Model(&Upload{}).Where("id = ?", id).Updates(map[string]any{
		"status":              uploadStatusCompleted,
		"sanitization_status": SanitizationReady,
		"storage_key":         derivativeKey,
		"size_bytes":          sizeBytes,
		"content_type":        SanitizedDerivativeMIME,
		"raw_storage_key":     nil,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to mark upload sanitized: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

// MarkUploadRejected marks an upload as rejected after failed sanitization.
func (r *Repository) MarkUploadRejected(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&Upload{}).Where("id = ?", id).Updates(map[string]any{
		"status":              uploadStatusRejected,
		"sanitization_status": SanitizationRejected,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to mark upload rejected: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUploadNotFound
	}
	return nil
}

// CreateFile inserts a completed file record.
func (r *Repository) CreateFile(ctx context.Context, file *File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// FinalizeUpload atomically inserts the file and completes its upload. Replays
// return the existing file, so a lost HTTP response is safe to retry.
func (r *Repository) FinalizeUpload(ctx context.Context, uploadID uuid.UUID, file *File, rawStorageKey *string) (*File, bool, error) {
	if r == nil || r.db == nil || file == nil {
		return nil, false, ErrUploadNotFound
	}
	var stored *File
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var upload Upload
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&upload, "id = ?", uploadID).Error; err != nil {
			return err
		}
		if upload.Status == uploadStatusCompleted {
			var existing File
			if err := tx.Where("upload_id = ?", uploadID).Take(&existing).Error; err != nil {
				return err
			}
			stored = &existing
			return nil
		}
		if upload.Status != uploadStatusPending {
			return ErrUploadAlreadyComplete
		}
		if err := tx.Create(file).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status": uploadStatusCompleted, "sanitization_status": file.SanitizationStatus,
			"storage_key": file.StorageKey, "size_bytes": file.SizeBytes,
			"content_type": file.ContentType, "raw_storage_key": rawStorageKey,
		}
		if err := tx.Model(&Upload{}).Where("id = ? AND status = ?", uploadID, uploadStatusPending).Updates(updates).Error; err != nil {
			return err
		}
		stored, created = file, true
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, ErrUploadNotFound
	}
	if err != nil {
		return nil, false, fmt.Errorf("finalize upload: %w", err)
	}
	return stored, created, nil
}

// ClearRawStorageKey records successful post-commit raw deletion.
func (r *Repository) ClearRawStorageKey(ctx context.Context, uploadID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&Upload{}).Where("id = ?", uploadID).Update("raw_storage_key", nil).Error; err != nil {
		return fmt.Errorf("clear raw storage key: %w", err)
	}
	return nil
}

// GetFileByUploadID returns the finalized file for idempotent completion replay.
func (r *Repository) GetFileByUploadID(ctx context.Context, uploadID uuid.UUID) (*File, error) {
	var file File
	if err := r.db.WithContext(ctx).Where("upload_id = ?", uploadID).Take(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get file by upload: %w", err)
	}
	return &file, nil
}

// GetFileByID returns a file by id.
func (r *Repository) GetFileByID(ctx context.Context, id uuid.UUID) (*File, error) {
	var file File
	if err := r.db.WithContext(ctx).First(&file, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return &file, nil
}

// CleanupExpiredUploads deletes uploads that expired before the given time.
func (r *Repository) CleanupExpiredUploads(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).Where("expires_at < ? AND status = 'pending'", before).Delete(&Upload{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired uploads: %w", err)
	}
	return nil
}
