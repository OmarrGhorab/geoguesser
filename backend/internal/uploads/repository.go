package uploads

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	result := r.db.WithContext(ctx).Model(&Upload{}).Where("id = ?", id).Update("status", "completed")
	if result.Error != nil {
		return fmt.Errorf("failed to mark upload completed: %w", result.Error)
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
