package uploads_test

import (
	"bytes"
	"context"
	"errors"
	"image/color"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/platform/storage"
	"github.com/raven/geoguess/backend/internal/uploads"
)

func TestCreateUploadValidation(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local provider failed: %v", err)
	}
	repo := &fakeUploadRepository{}
	service := uploads.NewService(repo, provider, config.Config{R2MaxFileSize: 10 * 1024 * 1024})

	cases := []struct {
		name string
		req  uploads.CreateUploadRequest
	}{
		{"empty file_name", uploads.CreateUploadRequest{FileName: "", ContentType: "image/png", SizeBytes: 1000}},
		{"empty content_type", uploads.CreateUploadRequest{FileName: "x.png", ContentType: "", SizeBytes: 1000}},
		{"zero size", uploads.CreateUploadRequest{FileName: "x.png", ContentType: "image/png", SizeBytes: 0}},
		{"too large", uploads.CreateUploadRequest{FileName: "x.png", ContentType: "image/png", SizeBytes: 20 * 1024 * 1024}},
		{"bad extension", uploads.CreateUploadRequest{FileName: "x.exe", ContentType: "image/png", SizeBytes: 1000}},
		{"bad content type", uploads.CreateUploadRequest{FileName: "x.png", ContentType: "application/json", SizeBytes: 1000}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.CreateUpload(context.Background(), "0197a1f0-0000-7000-8000-000000000001", tc.req)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestCreateUpload_GeneralPreserved(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local provider: %v", err)
	}
	repo := &fakeUploadRepository{}
	service := uploads.NewService(repo, provider, config.Config{R2MaxFileSize: 10 * 1024 * 1024})

	resp, err := service.CreateUpload(context.Background(), "0197a1f0-0000-7000-8000-000000000001", uploads.CreateUploadRequest{
		FileName:    "avatar.gif",
		ContentType: "image/gif",
		SizeBytes:   1024,
	})
	if err != nil {
		t.Fatalf("CreateUpload general: %v", err)
	}
	if resp.UploadID == "" || resp.UploadURL == "" {
		t.Fatal("expected upload id and url")
	}
	if repo.created == nil {
		t.Fatal("expected upload row")
	}
	if repo.created.Purpose != uploads.PurposeGeneral {
		t.Fatalf("purpose = %q", repo.created.Purpose)
	}
	if repo.created.SanitizationStatus != uploads.SanitizationNotRequired {
		t.Fatalf("sanitization = %q", repo.created.SanitizationStatus)
	}
}

func TestCreateUpload_TeamChatRequiresAuthAndFlags(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local provider: %v", err)
	}
	repo := &fakeUploadRepository{}
	matchID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000099")
	userID := "0197a1f0-0000-7000-8000-000000000001"

	// Disabled flag.
	svc := uploads.NewService(repo, provider, config.Config{
		R2MaxFileSize:         10 * 1024 * 1024,
		TeamChatImagesEnabled: false,
		TeamChatImageMaxBytes: 5 * 1024 * 1024,
	}).WithMatchAccess(allowAllMatchAuth{})
	_, err = svc.CreateUpload(context.Background(), userID, uploads.CreateUploadRequest{
		FileName:    "clue.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   1000,
		Purpose:     uploads.PurposeTeamChat,
		ContextID:   matchID.String(),
	})
	if !errors.Is(err, uploads.ErrImagesDisabled) {
		t.Fatalf("disabled = %v, want ErrImagesDisabled", err)
	}

	// Enabled but no match auth → fail closed.
	svc = uploads.NewService(repo, provider, config.Config{
		R2MaxFileSize:             10 * 1024 * 1024,
		TeamChatImagesEnabled:     true,
		TeamChatImageMaxBytes:     5 * 1024 * 1024,
		TeamChatImageMaxPixels:    20_000_000,
		TeamChatImageMaxDimension: 2048,
	})
	_, err = svc.CreateUpload(context.Background(), userID, uploads.CreateUploadRequest{
		FileName:    "clue.jpg",
		ContentType: "image/jpeg",
		SizeBytes:   1000,
		Purpose:     uploads.PurposeTeamChat,
		ContextID:   matchID.String(),
	})
	if !errors.Is(err, uploads.ErrMatchAccessDenied) {
		t.Fatalf("no auth = %v, want ErrMatchAccessDenied", err)
	}

	// SVG rejected for team chat.
	svc = uploads.NewService(repo, provider, config.Config{
		R2MaxFileSize:         10 * 1024 * 1024,
		TeamChatImagesEnabled: true,
		TeamChatImageMaxBytes: 5 * 1024 * 1024,
	}).WithMatchAccess(allowAllMatchAuth{})
	_, err = svc.CreateUpload(context.Background(), userID, uploads.CreateUploadRequest{
		FileName:    "x.svg",
		ContentType: "image/svg+xml",
		SizeBytes:   1000,
		Purpose:     uploads.PurposeTeamChat,
		ContextID:   matchID.String(),
	})
	if !errors.Is(err, uploads.ErrInvalidContentType) {
		t.Fatalf("svg = %v, want ErrInvalidContentType", err)
	}

	// Happy path create.
	_, err = svc.CreateUpload(context.Background(), userID, uploads.CreateUploadRequest{
		FileName:    "clue.webp",
		ContentType: "image/webp",
		SizeBytes:   2048,
		Purpose:     uploads.PurposeTeamChat,
		ContextID:   matchID.String(),
	})
	if err != nil {
		t.Fatalf("team_chat create: %v", err)
	}
	if repo.created.Purpose != uploads.PurposeTeamChat {
		t.Fatalf("purpose = %q", repo.created.Purpose)
	}
	if repo.created.SanitizationStatus != uploads.SanitizationPending {
		t.Fatalf("sanitization = %q", repo.created.SanitizationStatus)
	}
	if repo.created.ContextID == nil || *repo.created.ContextID != matchID {
		t.Fatalf("context_id mismatch")
	}
	if repo.created.RawStorageKey == nil || *repo.created.RawStorageKey != repo.created.StorageKey {
		t.Fatalf("raw_storage_key should equal pending storage key")
	}
}

func TestCompleteUploadRejectsObjectMetadataMismatch(t *testing.T) {
	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	uploadID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000002")
	repo := &fakeUploadRepository{
		upload: &uploads.Upload{
			ID:                 uploadID,
			OwnerUserID:        userID,
			FileName:           "avatar.png",
			ContentType:        "image/png",
			SizeBytes:          100,
			StorageKey:         "uploads/user/avatar.png",
			Status:             "pending",
			Purpose:            uploads.PurposeGeneral,
			SanitizationStatus: uploads.SanitizationNotRequired,
			ExpiresAt:          time.Now().UTC().Add(time.Minute),
		},
	}
	provider := &fakeStorageProvider{
		object: &storage.ObjectInfo{
			Key:         "uploads/user/avatar.png",
			Size:        200,
			ContentType: "image/png",
		},
	}
	service := uploads.NewService(repo, provider, config.Config{R2MaxFileSize: 10 * 1024 * 1024})

	_, err := service.CompleteUpload(context.Background(), userID.String(), uploads.CompleteUploadRequest{UploadID: uploadID.String()})
	if !errors.Is(err, uploads.ErrObjectMetadataMismatch) {
		t.Fatalf("error = %v, want ErrObjectMetadataMismatch", err)
	}
	if repo.createdFile != nil {
		t.Fatal("file should not be created for mismatched object metadata")
	}
}

func TestCompleteUpload_TeamChatSanitizesAndCleansRaw(t *testing.T) {
	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	uploadID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000002")
	matchID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000099")

	local, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local: %v", err)
	}
	rawKey := "uploads/raw/team_chat/" + userID.String() + "/" + uploadID.String() + ".jpg"
	jpegBytes := mustEncodeJPEG(t, solidImage(80, 40, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	if err := local.PutObject(context.Background(), rawKey, uploads.MIMEJPEG, bytes.NewReader(jpegBytes), int64(len(jpegBytes))); err != nil {
		t.Fatalf("put raw: %v", err)
	}

	rawKeyCopy := rawKey
	repo := &fakeUploadRepository{
		upload: &uploads.Upload{
			ID:                 uploadID,
			OwnerUserID:        userID,
			FileName:           "clue.jpg",
			ContentType:        uploads.MIMEJPEG,
			SizeBytes:          int64(len(jpegBytes)),
			StorageKey:         rawKey,
			Status:             "pending",
			Purpose:            uploads.PurposeTeamChat,
			ContextID:          &matchID,
			SanitizationStatus: uploads.SanitizationPending,
			RawStorageKey:      &rawKeyCopy,
			ExpiresAt:          time.Now().UTC().Add(time.Minute),
		},
	}

	svc := uploads.NewService(repo, local, config.Config{
		R2MaxFileSize:             10 * 1024 * 1024,
		TeamChatImagesEnabled:     true,
		TeamChatImageMaxBytes:     5 * 1024 * 1024,
		TeamChatImageMaxPixels:    20_000_000,
		TeamChatImageMaxDimension: 2048,
		R2SignedURLTTL:            time.Minute,
	}).WithMatchAccess(allowAllMatchAuth{})

	resp, err := svc.CompleteUpload(context.Background(), userID.String(), uploads.CompleteUploadRequest{UploadID: uploadID.String()})
	if err != nil {
		t.Fatalf("complete team_chat: %v", err)
	}
	if resp.File.ContentType != uploads.MIMEJPEG {
		t.Fatalf("file content type = %q", resp.File.ContentType)
	}
	if resp.File.SanitizationStatus != uploads.SanitizationReady {
		t.Fatalf("sanitization = %q", resp.File.SanitizationStatus)
	}
	if resp.File.Purpose != uploads.PurposeTeamChat {
		t.Fatalf("purpose = %q", resp.File.Purpose)
	}
	if repo.createdFile == nil || !repo.createdFile.IsReadyTeamChat() {
		t.Fatal("expected ready team chat file")
	}
	// Raw cleaned.
	if _, err := local.HeadObject(context.Background(), rawKey); !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("raw still present: %v", err)
	}
	// Derivative present.
	if _, err := local.HeadObject(context.Background(), repo.createdFile.StorageKey); err != nil {
		t.Fatalf("derivative missing: %v", err)
	}

	// Attachment integration API.
	file, err := svc.GetReadyTeamChatFile(context.Background(), repo.createdFile.ID, matchID, userID)
	if err != nil {
		t.Fatalf("GetReadyTeamChatFile: %v", err)
	}
	signed, err := svc.PresignTeamChatDownload(context.Background(), file, 2*time.Minute)
	if err != nil {
		t.Fatalf("PresignTeamChatDownload: %v", err)
	}
	if signed.URL == "" {
		t.Fatal("expected signed url")
	}
}

func TestCompleteUpload_TeamChatKeepsRawWhenFinalizationFails(t *testing.T) {
	userID, uploadID, matchID := uuid.New(), uuid.New(), uuid.New()
	local, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local: %v", err)
	}
	rawKey := "uploads/raw/team_chat/" + userID.String() + "/" + uploadID.String() + ".jpg"
	jpegBytes := mustEncodeJPEG(t, solidImage(32, 32, color.White))
	if err := local.PutObject(context.Background(), rawKey, uploads.MIMEJPEG, bytes.NewReader(jpegBytes), int64(len(jpegBytes))); err != nil {
		t.Fatalf("put raw: %v", err)
	}
	rawCopy := rawKey
	repo := &fakeUploadRepository{
		finalizeErr: errors.New("database unavailable"),
		upload: &uploads.Upload{
			ID: uploadID, OwnerUserID: userID, FileName: "clue.jpg", ContentType: uploads.MIMEJPEG,
			SizeBytes: int64(len(jpegBytes)), StorageKey: rawKey, Status: "pending", Purpose: uploads.PurposeTeamChat,
			ContextID: &matchID, SanitizationStatus: uploads.SanitizationPending, RawStorageKey: &rawCopy,
			ExpiresAt: time.Now().UTC().Add(time.Minute),
		},
	}
	svc := uploads.NewService(repo, local, config.Config{
		TeamChatImagesEnabled: true, TeamChatImageMaxBytes: 5 * 1024 * 1024,
		TeamChatImageMaxPixels: 20_000_000, TeamChatImageMaxDimension: 2048,
	}).WithMatchAccess(allowAllMatchAuth{})

	if _, err := svc.CompleteUpload(context.Background(), userID.String(), uploads.CompleteUploadRequest{UploadID: uploadID.String()}); err == nil {
		t.Fatal("expected finalization failure")
	}
	if _, err := local.HeadObject(context.Background(), rawKey); err != nil {
		t.Fatalf("raw must remain retryable after DB failure: %v", err)
	}
}

func TestCompleteUpload_TeamChatRejectsSpoofAndCleansRaw(t *testing.T) {
	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	uploadID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000003")
	matchID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000099")

	local, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local: %v", err)
	}
	rawKey := "uploads/raw/team_chat/spoof.png"
	pngBytes := mustEncodePNG(t, solidImage(16, 16, color.White))
	if err := local.PutObject(context.Background(), rawKey, uploads.MIMEPNG, bytes.NewReader(pngBytes), int64(len(pngBytes))); err != nil {
		t.Fatalf("put: %v", err)
	}
	rawCopy := rawKey
	repo := &fakeUploadRepository{
		upload: &uploads.Upload{
			ID:                 uploadID,
			OwnerUserID:        userID,
			FileName:           "spoof.jpg",
			ContentType:        uploads.MIMEJPEG, // declared JPEG, body PNG
			SizeBytes:          int64(len(pngBytes)),
			StorageKey:         rawKey,
			Status:             "pending",
			Purpose:            uploads.PurposeTeamChat,
			ContextID:          &matchID,
			SanitizationStatus: uploads.SanitizationPending,
			RawStorageKey:      &rawCopy,
			ExpiresAt:          time.Now().UTC().Add(time.Minute),
		},
	}
	svc := uploads.NewService(repo, local, config.Config{
		TeamChatImagesEnabled:     true,
		TeamChatImageMaxBytes:     5 * 1024 * 1024,
		TeamChatImageMaxPixels:    20_000_000,
		TeamChatImageMaxDimension: 2048,
	}).WithMatchAccess(allowAllMatchAuth{})

	_, err = svc.CompleteUpload(context.Background(), userID.String(), uploads.CompleteUploadRequest{UploadID: uploadID.String()})
	if !errors.Is(err, uploads.ErrUnsafeImage) {
		t.Fatalf("spoof complete = %v, want ErrUnsafeImage", err)
	}
	if !repo.rejected {
		t.Fatal("expected upload marked rejected")
	}
	if _, err := local.HeadObject(context.Background(), rawKey); !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("raw should be cleaned: %v", err)
	}
}

func TestGetReadyTeamChatFile_Guards(t *testing.T) {
	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	other := uuid.MustParse("0197a1f0-0000-7000-8000-000000000002")
	matchID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000099")
	fileID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000010")
	provider, _ := storage.NewLocalProvider(t.TempDir())

	ready := &uploads.File{
		ID:                 fileID,
		OwnerUserID:        userID,
		FileName:           "a.jpg",
		ContentType:        uploads.MIMEJPEG,
		SizeBytes:          10,
		StorageKey:         "sanitized/a.jpg",
		Purpose:            uploads.PurposeTeamChat,
		ContextID:          &matchID,
		SanitizationStatus: uploads.SanitizationReady,
	}
	repo := &fakeUploadRepository{file: ready}
	svc := uploads.NewService(repo, provider, config.Config{})

	if _, err := svc.GetReadyTeamChatFile(context.Background(), fileID, matchID, other); !errors.Is(err, uploads.ErrForbidden) {
		t.Fatalf("wrong owner = %v", err)
	}
	wrongMatch := uuid.MustParse("0197a1f0-0000-7000-8000-000000000088")
	if _, err := svc.GetReadyTeamChatFile(context.Background(), fileID, wrongMatch, userID); !errors.Is(err, uploads.ErrPurposeContextMismatch) {
		t.Fatalf("wrong match = %v", err)
	}
	ready.SanitizationStatus = uploads.SanitizationPending
	if _, err := svc.GetReadyTeamChatFile(context.Background(), fileID, matchID, userID); !errors.Is(err, uploads.ErrAttachmentNotReady) {
		t.Fatalf("pending = %v", err)
	}
}

// --- fakes ---

type allowAllMatchAuth struct{}

func (allowAllMatchAuth) AuthorizeTeamChatUpload(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type fakeUploadRepository struct {
	upload      *uploads.Upload
	created     *uploads.Upload
	createdFile *uploads.File
	file        *uploads.File
	rejected    bool
	finalizeErr error
}

func (f *fakeUploadRepository) CreateUpload(_ context.Context, upload *uploads.Upload) error {
	cpy := *upload
	f.created = &cpy
	f.upload = &cpy
	return nil
}

func (f *fakeUploadRepository) GetUploadByID(context.Context, uuid.UUID) (*uploads.Upload, error) {
	return f.upload, nil
}

func (f *fakeUploadRepository) MarkUploadCompleted(context.Context, uuid.UUID) error {
	return nil
}

func (f *fakeUploadRepository) MarkUploadCompletedSanitized(_ context.Context, _ uuid.UUID, derivativeKey string, sizeBytes int64) error {
	if f.upload != nil {
		f.upload.StorageKey = derivativeKey
		f.upload.SizeBytes = sizeBytes
		f.upload.Status = "completed"
		f.upload.SanitizationStatus = uploads.SanitizationReady
		f.upload.RawStorageKey = nil
	}
	return nil
}

func (f *fakeUploadRepository) MarkUploadRejected(context.Context, uuid.UUID) error {
	f.rejected = true
	if f.upload != nil {
		f.upload.Status = "rejected"
		f.upload.SanitizationStatus = uploads.SanitizationRejected
	}
	return nil
}

func (f *fakeUploadRepository) CreateFile(_ context.Context, file *uploads.File) error {
	cpy := *file
	f.createdFile = &cpy
	f.file = &cpy
	return nil
}

func (f *fakeUploadRepository) FinalizeUpload(_ context.Context, _ uuid.UUID, file *uploads.File, rawStorageKey *string) (*uploads.File, bool, error) {
	if f.finalizeErr != nil {
		return nil, false, f.finalizeErr
	}
	if f.file != nil && f.file.UploadID != nil && file.UploadID != nil && *f.file.UploadID == *file.UploadID {
		return f.file, false, nil
	}
	cpy := *file
	f.createdFile = &cpy
	f.file = &cpy
	if f.upload != nil {
		f.upload.Status = "completed"
		f.upload.StorageKey = file.StorageKey
		f.upload.SizeBytes = file.SizeBytes
		f.upload.ContentType = file.ContentType
		f.upload.SanitizationStatus = file.SanitizationStatus
		f.upload.RawStorageKey = rawStorageKey
	}
	return &cpy, true, nil
}

func (f *fakeUploadRepository) ClearRawStorageKey(context.Context, uuid.UUID) error {
	if f.upload != nil {
		f.upload.RawStorageKey = nil
	}
	return nil
}

func (f *fakeUploadRepository) GetFileByUploadID(_ context.Context, uploadID uuid.UUID) (*uploads.File, error) {
	if f.file != nil && f.file.UploadID != nil && *f.file.UploadID == uploadID {
		return f.file, nil
	}
	return nil, nil
}

func (f *fakeUploadRepository) GetFileByID(context.Context, uuid.UUID) (*uploads.File, error) {
	return f.file, nil
}

type fakeStorageProvider struct {
	object  *storage.ObjectInfo
	objects map[string][]byte
}

func (f *fakeStorageProvider) PresignedUploadURL(context.Context, string, string, time.Duration, int64) (string, error) {
	return "https://upload.example.test", nil
}

func (f *fakeStorageProvider) PresignedDownloadURL(context.Context, string, time.Duration) (string, error) {
	return "https://download.example.test", nil
}

func (f *fakeStorageProvider) HeadObject(context.Context, string) (*storage.ObjectInfo, error) {
	return f.object, nil
}

func (f *fakeStorageProvider) GetObject(ctx context.Context, key string, maxBytes int64) (io.ReadCloser, *storage.ObjectInfo, error) {
	if f.objects != nil {
		if data, ok := f.objects[key]; ok {
			if int64(len(data)) > maxBytes {
				return nil, nil, storage.ErrObjectTooLarge
			}
			return io.NopCloser(bytes.NewReader(data)), &storage.ObjectInfo{Key: key, Size: int64(len(data))}, nil
		}
	}
	return nil, nil, storage.ErrObjectNotFound
}

func (f *fakeStorageProvider) PutObject(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	f.objects[key] = data
	return nil
}

func (f *fakeStorageProvider) DeleteObject(context.Context, string) error {
	return nil
}
