package uploads

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/platform/storage"
)

const (
	uploadStatusPending   = "pending"
	uploadStatusCompleted = "completed"
	uploadStatusExpired   = "expired"
	uploadTTL             = 30 * time.Minute
)

// allowedContentTypes is the allowlist for uploads.
var allowedContentTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// allowedExtensions is the allowlist for file extensions.
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
}

// Service implements uploads business logic.
type Service struct {
	repo     uploadRepository
	provider storage.Provider
	cfg      config.Config
}

type uploadRepository interface {
	CreateUpload(ctx context.Context, upload *Upload) error
	GetUploadByID(ctx context.Context, id uuid.UUID) (*Upload, error)
	MarkUploadCompleted(ctx context.Context, id uuid.UUID) error
	CreateFile(ctx context.Context, file *File) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*File, error)
}

// NewService returns a new uploads service.
func NewService(repo uploadRepository, provider storage.Provider, cfg config.Config) *Service {
	return &Service{repo: repo, provider: provider, cfg: cfg}
}

// CreateUpload creates a pending upload and returns a presigned URL.
func (s *Service) CreateUpload(ctx context.Context, ownerUserID string, req CreateUploadRequest) (*CreateUploadResponse, error) {
	userID, err := uuid.Parse(ownerUserID)
	if err != nil {
		return nil, ErrForbidden
	}

	if strings.TrimSpace(req.FileName) == "" {
		return nil, ErrFileNameRequired
	}
	if strings.TrimSpace(req.ContentType) == "" {
		return nil, ErrContentTypeRequired
	}
	if req.SizeBytes <= 0 {
		return nil, ErrInvalidSize
	}
	if req.SizeBytes > s.cfg.R2MaxFileSize {
		return nil, ErrFileTooLarge
	}
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if !allowedExtensions[ext] {
		return nil, ErrInvalidContentType
	}
	if !allowedContentTypes[req.ContentType] {
		return nil, ErrInvalidContentType
	}

	uploadID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload id: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(uploadTTL)
	storageKey := fmt.Sprintf("uploads/%s/%s%s", userID.String(), uploadID.String(), ext)

	upload := &Upload{
		ID:          uploadID,
		OwnerUserID: userID,
		FileName:    sanitizeFileName(req.FileName),
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		StorageKey:  storageKey,
		Status:      uploadStatusPending,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	if err := s.repo.CreateUpload(ctx, upload); err != nil {
		return nil, err
	}

	uploadURL, err := s.provider.PresignedUploadURL(ctx, storageKey, req.ContentType, uploadTTL, req.SizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create presigned upload url: %w", err)
	}

	return &CreateUploadResponse{
		UploadID:   uploadID.String(),
		UploadURL:  uploadURL,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
		StorageKey: storageKey,
	}, nil
}

// CompleteUpload verifies the object exists and records the file.
func (s *Service) CompleteUpload(ctx context.Context, ownerUserID string, req CompleteUploadRequest) (*FileResponse, error) {
	userID, err := uuid.Parse(ownerUserID)
	if err != nil {
		return nil, ErrForbidden
	}
	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		return nil, ErrUploadNotFound
	}

	upload, err := s.repo.GetUploadByID(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if upload == nil {
		return nil, ErrUploadNotFound
	}
	if upload.OwnerUserID != userID {
		return nil, ErrForbidden
	}
	if upload.Status == uploadStatusCompleted {
		return nil, ErrUploadAlreadyComplete
	}
	if upload.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUploadExpired
	}

	info, err := s.provider.HeadObject(ctx, upload.StorageKey)
	if err != nil {
		return nil, ErrObjectNotFound
	}
	if info.Size != upload.SizeBytes || info.ContentType != upload.ContentType {
		return nil, ErrObjectMetadataMismatch
	}

	fileID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate file id: %w", err)
	}
	now := time.Now().UTC()
	file := &File{
		ID:          fileID,
		UploadID:    &uploadID,
		OwnerUserID: userID,
		FileName:    upload.FileName,
		ContentType: upload.ContentType,
		SizeBytes:   info.Size,
		StorageKey:  upload.StorageKey,
		IsPublic:    false,
		CreatedAt:   now,
	}

	if err := s.repo.CreateFile(ctx, file); err != nil {
		return nil, err
	}
	if err := s.repo.MarkUploadCompleted(ctx, uploadID); err != nil {
		return nil, err
	}

	return &FileResponse{File: toFileDTO(file)}, nil
}

// GetSignedURL returns a signed download URL for a file owned by the user.
func (s *Service) GetSignedURL(ctx context.Context, ownerUserID string, fileID string) (*SignedURLResponse, error) {
	userID, err := uuid.Parse(ownerUserID)
	if err != nil {
		return nil, ErrForbidden
	}
	id, err := uuid.Parse(fileID)
	if err != nil {
		return nil, ErrFileNotFound
	}

	file, err := s.repo.GetFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	if file.OwnerUserID != userID {
		return nil, ErrForbidden
	}

	expiresAt := time.Now().UTC().Add(s.cfg.R2SignedURLTTL)
	url, err := s.provider.PresignedDownloadURL(ctx, file.StorageKey, s.cfg.R2SignedURLTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create signed url: %w", err)
	}

	return &SignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func toFileDTO(file *File) FileDTO {
	return FileDTO{
		ID:          file.ID.String(),
		FileName:    file.FileName,
		ContentType: file.ContentType,
		SizeBytes:   file.SizeBytes,
		CreatedAt:   file.CreatedAt.Format(time.RFC3339),
	}
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return name
}
