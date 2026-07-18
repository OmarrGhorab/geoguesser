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

const uploadTTL = 30 * time.Minute

// allowedContentTypes is the allowlist for general uploads.
var allowedContentTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
}

// allowedExtensions is the allowlist for general file extensions.
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
}

// teamChatContentTypes is the allowlist for team-chat uploads (pre-sanitization).
var teamChatContentTypes = map[string]bool{
	MIMEJPEG:    true,
	MIMEPNG:     true,
	MIMEWebP:    true,
	"image/jpg": true,
}

// teamChatExtensions is the allowlist for team-chat file extensions.
var teamChatExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// MatchAccessChecker authorizes team-chat uploads against a match roster.
// Implementations live in matchplay wiring; uploads only depends on this surface.
// Returning nil means the user may create/complete team-chat uploads for matchID.
type MatchAccessChecker interface {
	AuthorizeTeamChatUpload(ctx context.Context, matchID, userID uuid.UUID) error
}

// Service implements uploads business logic.
type Service struct {
	repo      uploadRepository
	provider  storage.Provider
	cfg       config.Config
	sanitizer *ImageSanitizer
	matchAuth MatchAccessChecker
	metrics   MetricsRecorder
}

type uploadRepository interface {
	CreateUpload(ctx context.Context, upload *Upload) error
	GetUploadByID(ctx context.Context, id uuid.UUID) (*Upload, error)
	MarkUploadRejected(ctx context.Context, id uuid.UUID) error
	FinalizeUpload(ctx context.Context, uploadID uuid.UUID, file *File, rawStorageKey *string) (*File, bool, error)
	ClearRawStorageKey(ctx context.Context, uploadID uuid.UUID) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*File, error)
	GetFileByUploadID(ctx context.Context, uploadID uuid.UUID) (*File, error)
}

// NewService returns a new uploads service with default sanitizer limits from config.
func NewService(repo uploadRepository, provider storage.Provider, cfg config.Config) *Service {
	limits := SanitizeLimits{
		MaxInputBytes:  cfg.TeamChatImageMaxBytes,
		MaxPixels:      cfg.TeamChatImageMaxPixels,
		MaxDimension:   cfg.TeamChatImageMaxDimension,
		MaxOutputBytes: cfg.TeamChatImageMaxBytes,
		JPEGQuality:    85,
	}
	if limits.MaxInputBytes <= 0 {
		limits = DefaultSanitizeLimits()
	}
	s := &Service{
		repo:     repo,
		provider: provider,
		cfg:      cfg,
		metrics:  NoopMetrics{},
	}
	s.sanitizer = NewImageSanitizer(provider, limits, s.metrics)
	return s
}

// WithMatchAccess attaches same-match authorization for team-chat purpose.
func (s *Service) WithMatchAccess(checker MatchAccessChecker) *Service {
	s.matchAuth = checker
	return s
}

// WithMetrics attaches a metrics recorder and rebuilds the sanitizer to share it.
func (s *Service) WithMetrics(m MetricsRecorder) *Service {
	if m == nil {
		m = NoopMetrics{}
	}
	s.metrics = m
	limits := s.sanitizer.limits
	s.sanitizer = NewImageSanitizer(s.provider, limits, m)
	return s
}

// CreateUpload creates a pending upload and returns a presigned URL.
// Omitting purpose (or purpose=general) preserves existing general upload behavior.
func (s *Service) CreateUpload(ctx context.Context, ownerUserID string, req CreateUploadRequest) (*CreateUploadResponse, error) {
	start := nowUTC()
	purpose := normalizePurpose(req.Purpose)
	var outcome = "error"
	defer func() {
		s.metrics.ObserveUpload("create", purpose, outcome, since(start))
	}()

	userID, err := uuid.Parse(ownerUserID)
	if err != nil {
		outcome = "forbidden"
		return nil, ErrForbidden
	}

	if strings.TrimSpace(req.FileName) == "" {
		outcome = "validation"
		return nil, ErrFileNameRequired
	}
	if strings.TrimSpace(req.ContentType) == "" {
		outcome = "validation"
		return nil, ErrContentTypeRequired
	}
	if req.SizeBytes <= 0 {
		outcome = "validation"
		return nil, ErrInvalidSize
	}

	ext := strings.ToLower(filepath.Ext(req.FileName))
	contentType := NormalizeDeclaredMIME(req.ContentType)

	var (
		contextID     *uuid.UUID
		rawKey        *string
		sanitization  string
		maxSize       int64
		storagePrefix string
	)

	switch purpose {
	case PurposeGeneral:
		if req.SizeBytes > s.cfg.R2MaxFileSize {
			outcome = "validation"
			return nil, ErrFileTooLarge
		}
		if !allowedExtensions[ext] {
			outcome = "validation"
			return nil, ErrInvalidContentType
		}
		if !allowedContentTypes[contentType] {
			outcome = "validation"
			return nil, ErrInvalidContentType
		}
		sanitization = SanitizationNotRequired
		maxSize = req.SizeBytes
		storagePrefix = "uploads"

	case PurposeTeamChat:
		if !s.cfg.TeamChatImagesEnabled {
			outcome = "disabled"
			return nil, ErrImagesDisabled
		}
		if s.provider == nil {
			outcome = "storage_unavailable"
			return nil, ErrStorageUnavailable
		}
		maxBytes := s.cfg.TeamChatImageMaxBytes
		if maxBytes <= 0 {
			maxBytes = 5 * 1024 * 1024
		}
		if req.SizeBytes > maxBytes {
			outcome = "validation"
			return nil, ErrFileTooLarge
		}
		if !teamChatExtensions[ext] {
			outcome = "validation"
			return nil, ErrInvalidContentType
		}
		if !teamChatContentTypes[contentType] {
			outcome = "validation"
			return nil, ErrInvalidContentType
		}
		if strings.TrimSpace(req.ContextID) == "" {
			outcome = "validation"
			return nil, ErrContextIDRequired
		}
		matchID, err := uuid.Parse(req.ContextID)
		if err != nil {
			outcome = "validation"
			return nil, ErrInvalidContextID
		}
		if err := s.authorizeTeamChat(ctx, matchID, userID); err != nil {
			outcome = "forbidden"
			return nil, err
		}
		contextID = &matchID
		sanitization = SanitizationPending
		maxSize = req.SizeBytes
		storagePrefix = "uploads/raw/team_chat"

	default:
		outcome = "validation"
		return nil, ErrInvalidPurpose
	}

	uploadID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload id: %w", err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(uploadTTL)
	storageKey := fmt.Sprintf("%s/%s/%s%s", storagePrefix, userID.String(), uploadID.String(), ext)
	if purpose == PurposeTeamChat {
		rawKey = &storageKey
	}

	upload := &Upload{
		ID:                 uploadID,
		OwnerUserID:        userID,
		FileName:           sanitizeFileName(req.FileName),
		ContentType:        contentType,
		SizeBytes:          req.SizeBytes,
		StorageKey:         storageKey,
		Status:             uploadStatusPending,
		Purpose:            purpose,
		ContextID:          contextID,
		SanitizationStatus: sanitization,
		RawStorageKey:      rawKey,
		ExpiresAt:          expiresAt,
		CreatedAt:          now,
	}

	if err := s.repo.CreateUpload(ctx, upload); err != nil {
		return nil, err
	}

	uploadURL, err := s.provider.PresignedUploadURL(ctx, storageKey, contentType, uploadTTL, maxSize)
	if err != nil {
		s.metrics.ObserveStorageFailure("presign")
		outcome = "storage_unavailable"
		return nil, ErrStorageUnavailable
	}

	outcome = "ok"
	return &CreateUploadResponse{
		UploadID:   uploadID.String(),
		UploadURL:  uploadURL,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
		StorageKey: storageKey,
	}, nil
}

// CompleteUpload verifies the object exists and records the file.
// For team_chat purpose, performs synchronous sanitization and only returns
// ready sanitized file metadata. Raw objects are deleted on success or rejection.
func (s *Service) CompleteUpload(ctx context.Context, ownerUserID string, req CompleteUploadRequest) (*FileResponse, error) {
	start := nowUTC()
	purpose := PurposeGeneral
	var outcome = "error"
	defer func() {
		s.metrics.ObserveUpload("complete", purpose, outcome, since(start))
	}()

	userID, err := uuid.Parse(ownerUserID)
	if err != nil {
		outcome = "forbidden"
		return nil, ErrForbidden
	}
	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		outcome = "not_found"
		return nil, ErrUploadNotFound
	}

	upload, err := s.repo.GetUploadByID(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if upload == nil {
		outcome = "not_found"
		return nil, ErrUploadNotFound
	}
	purpose = normalizePurpose(upload.Purpose)
	if upload.OwnerUserID != userID {
		outcome = "forbidden"
		return nil, ErrForbidden
	}
	if upload.Status == uploadStatusCompleted {
		file, fileErr := s.repo.GetFileByUploadID(ctx, upload.ID)
		if fileErr != nil {
			return nil, fileErr
		}
		if file == nil {
			outcome = "conflict"
			return nil, ErrUploadAlreadyComplete
		}
		outcome = "replay"
		return &FileResponse{File: toFileDTO(file)}, nil
	}
	if upload.Status == uploadStatusRejected {
		outcome = "rejected"
		return nil, ErrUnsafeImage
	}
	if upload.ExpiresAt.Before(time.Now().UTC()) {
		outcome = "expired"
		return nil, ErrUploadExpired
	}

	if purpose == PurposeTeamChat {
		return s.completeTeamChatUpload(ctx, userID, upload, &outcome)
	}
	return s.completeGeneralUpload(ctx, userID, upload, &outcome)
}

func (s *Service) completeGeneralUpload(ctx context.Context, userID uuid.UUID, upload *Upload, outcome *string) (*FileResponse, error) {
	info, err := s.provider.HeadObject(ctx, upload.StorageKey)
	if err != nil {
		s.metrics.ObserveStorageFailure("head")
		*outcome = "not_found"
		return nil, ErrObjectNotFound
	}
	// Local storage may not preserve Content-Type precisely; allow size match and
	// treat empty/octet-stream as acceptable for general uploads.
	if info.Size != upload.SizeBytes {
		*outcome = "validation"
		return nil, ErrObjectMetadataMismatch
	}
	if info.ContentType != "" &&
		info.ContentType != "application/octet-stream" &&
		NormalizeDeclaredMIME(info.ContentType) != NormalizeDeclaredMIME(upload.ContentType) {
		*outcome = "validation"
		return nil, ErrObjectMetadataMismatch
	}

	fileID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate file id: %w", err)
	}
	now := time.Now().UTC()
	file := &File{
		ID:                 fileID,
		UploadID:           &upload.ID,
		OwnerUserID:        userID,
		FileName:           upload.FileName,
		ContentType:        upload.ContentType,
		SizeBytes:          info.Size,
		StorageKey:         upload.StorageKey,
		IsPublic:           false,
		Purpose:            PurposeGeneral,
		ContextID:          nil,
		SanitizationStatus: SanitizationNotRequired,
		RawStorageKey:      nil,
		CreatedAt:          now,
	}

	stored, _, err := s.repo.FinalizeUpload(ctx, upload.ID, file, nil)
	if err != nil {
		return nil, err
	}

	*outcome = "ok"
	return &FileResponse{File: toFileDTO(stored)}, nil
}

func (s *Service) completeTeamChatUpload(ctx context.Context, userID uuid.UUID, upload *Upload, outcome *string) (*FileResponse, error) {
	if !s.cfg.TeamChatImagesEnabled {
		*outcome = "disabled"
		return nil, ErrImagesDisabled
	}
	if upload.ContextID == nil {
		*outcome = "validation"
		return nil, ErrContextIDRequired
	}
	if err := s.authorizeTeamChat(ctx, *upload.ContextID, userID); err != nil {
		*outcome = "forbidden"
		return nil, err
	}

	// Verify raw object exists and size is within declared bounds.
	info, err := s.provider.HeadObject(ctx, upload.StorageKey)
	if err != nil {
		s.metrics.ObserveStorageFailure("head")
		if errorsIs(err, storage.ErrObjectNotFound) {
			*outcome = "not_found"
			return nil, ErrObjectNotFound
		}
		*outcome = "storage_unavailable"
		return nil, ErrStorageUnavailable
	}
	maxBytes := s.cfg.TeamChatImageMaxBytes
	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024
	}
	if info.Size <= 0 || info.Size > maxBytes || info.Size > upload.SizeBytes {
		// Oversized or empty raw: reject and cleanup.
		_ = s.provider.DeleteObject(ctx, upload.StorageKey)
		_ = s.repo.MarkUploadRejected(ctx, upload.ID)
		*outcome = "rejected"
		return nil, ErrFileTooLarge
	}

	derivativeKey := fmt.Sprintf("uploads/sanitized/team_chat/%s/%s.jpg", userID.String(), upload.ID.String())
	result, err := s.sanitizer.SanitizeStoredObjectDeferredCleanup(ctx, upload.StorageKey, derivativeKey, upload.ContentType, true)
	if err != nil {
		_ = s.repo.MarkUploadRejected(ctx, upload.ID)
		*outcome = "rejected"
		return nil, err
	}

	fileID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate file id: %w", err)
	}
	now := time.Now().UTC()
	// Derivative file name uses .jpg; original base name is preserved without raw extension risk.
	fileName := stripExtension(upload.FileName) + ".jpg"
	file := &File{
		ID:                 fileID,
		UploadID:           &upload.ID,
		OwnerUserID:        userID,
		FileName:           fileName,
		ContentType:        SanitizedDerivativeMIME,
		SizeBytes:          result.SizeBytes,
		StorageKey:         derivativeKey,
		IsPublic:           false,
		Purpose:            PurposeTeamChat,
		ContextID:          upload.ContextID,
		SanitizationStatus: SanitizationReady,
		RawStorageKey:      nil,
		CreatedAt:          now,
	}

	rawKey := upload.StorageKey
	stored, _, err := s.repo.FinalizeUpload(ctx, upload.ID, file, &rawKey)
	if err != nil {
		return nil, err
	}
	if err := s.provider.DeleteObject(ctx, rawKey); err != nil && !errorsIs(err, storage.ErrObjectNotFound) {
		s.metrics.ObserveRawCleanup("retry")
	} else {
		s.metrics.ObserveRawCleanup("deleted")
		_ = s.repo.ClearRawStorageKey(ctx, upload.ID)
	}

	*outcome = "ok"
	return &FileResponse{File: toFileDTO(stored)}, nil
}

// GetSignedURL returns a signed download URL for a file owned by the user.
// Owner-only. Teammates must use the authorized match attachment endpoint.
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
	// Team-chat files: only ready sanitized derivatives may be signed.
	if file.Purpose == PurposeTeamChat && file.SanitizationStatus != SanitizationReady {
		return nil, ErrAttachmentNotReady
	}

	return s.presignFile(ctx, file.StorageKey)
}

// ---------------------------------------------------------------------------
// Chat attachment integration API
// ---------------------------------------------------------------------------

// GetReadyTeamChatFile loads a team-chat file that is ready for attachment.
//
// Callers (matchplay chat) MUST still enforce same-team / chat-access policy.
// This method only enforces:
//   - file exists
//   - purpose == team_chat
//   - context_id == matchID
//   - owner_user_id == authorUserID
//   - sanitization_status == ready
//
// Storage keys are available on the returned File for PresignTeamChatDownload;
// never serialize StorageKey or RawStorageKey to clients.
func (s *Service) GetReadyTeamChatFile(ctx context.Context, fileID, matchID, authorUserID uuid.UUID) (*File, error) {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	if file.Purpose != PurposeTeamChat {
		return nil, ErrPurposeContextMismatch
	}
	if file.ContextID == nil || *file.ContextID != matchID {
		return nil, ErrPurposeContextMismatch
	}
	if file.OwnerUserID != authorUserID {
		return nil, ErrForbidden
	}
	if file.SanitizationStatus != SanitizationReady || !file.IsReadyTeamChat() {
		return nil, ErrAttachmentNotReady
	}
	return file, nil
}

// PresignTeamChatDownload returns a short-lived private signed URL for a ready
// team-chat derivative. Never accepts or returns raw storage keys for clients.
// ttl is capped at 5 minutes for chat attachment use.
func (s *Service) PresignTeamChatDownload(ctx context.Context, file *File, ttl time.Duration) (*SignedURLResponse, error) {
	if file == nil || !file.IsReadyTeamChat() {
		return nil, ErrAttachmentNotReady
	}
	if ttl <= 0 || ttl > 5*time.Minute {
		ttl = 5 * time.Minute
	}
	expiresAt := time.Now().UTC().Add(ttl)
	url, err := s.provider.PresignedDownloadURL(ctx, file.StorageKey, ttl)
	if err != nil {
		s.metrics.ObserveStorageFailure("presign")
		return nil, ErrStorageUnavailable
	}
	return &SignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// LookupFile returns a file by ID without ownership checks.
// Intended for authorized matchplay attachment flows after policy checks.
func (s *Service) LookupFile(ctx context.Context, fileID uuid.UUID) (*File, error) {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	return file, nil
}

func (s *Service) presignFile(ctx context.Context, storageKey string) (*SignedURLResponse, error) {
	expiresAt := time.Now().UTC().Add(s.cfg.R2SignedURLTTL)
	url, err := s.provider.PresignedDownloadURL(ctx, storageKey, s.cfg.R2SignedURLTTL)
	if err != nil {
		s.metrics.ObserveStorageFailure("presign")
		return nil, fmt.Errorf("failed to create signed url: %w", err)
	}
	return &SignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) authorizeTeamChat(ctx context.Context, matchID, userID uuid.UUID) error {
	if s.matchAuth == nil {
		// Fail closed when team-chat auth is not wired (prevents open upload).
		return ErrMatchAccessDenied
	}
	if err := s.matchAuth.AuthorizeTeamChatUpload(ctx, matchID, userID); err != nil {
		return ErrMatchAccessDenied
	}
	return nil
}

func normalizePurpose(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	if p == "" {
		return PurposeGeneral
	}
	return p
}

func toFileDTO(file *File) FileDTO {
	dto := FileDTO{
		ID:          file.ID.String(),
		FileName:    file.FileName,
		ContentType: file.ContentType,
		SizeBytes:   file.SizeBytes,
		CreatedAt:   file.CreatedAt.Format(time.RFC3339),
	}
	if file.Purpose != "" {
		dto.Purpose = file.Purpose
	}
	if file.SanitizationStatus != "" {
		dto.SanitizationStatus = file.SanitizationStatus
	}
	if file.ContextID != nil {
		dto.ContextID = file.ContextID.String()
	}
	return dto
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return name
}

func stripExtension(name string) string {
	name = sanitizeFileName(name)
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
