package uploads_test

import (
	"context"
	"errors"
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
	repo := uploads.NewRepository(nil) // repo not used for validation errors
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

func TestCompleteUploadRejectsObjectMetadataMismatch(t *testing.T) {
	userID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000001")
	uploadID := uuid.MustParse("0197a1f0-0000-7000-8000-000000000002")
	repo := &fakeUploadRepository{
		upload: &uploads.Upload{
			ID:          uploadID,
			OwnerUserID: userID,
			FileName:    "avatar.png",
			ContentType: "image/png",
			SizeBytes:   100,
			StorageKey:  "uploads/user/avatar.png",
			Status:      "pending",
			ExpiresAt:   time.Now().UTC().Add(time.Minute),
		},
	}
	provider := fakeStorageProvider{
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

type fakeUploadRepository struct {
	upload      *uploads.Upload
	createdFile *uploads.File
}

func (f *fakeUploadRepository) CreateUpload(context.Context, *uploads.Upload) error {
	return nil
}

func (f *fakeUploadRepository) GetUploadByID(context.Context, uuid.UUID) (*uploads.Upload, error) {
	return f.upload, nil
}

func (f *fakeUploadRepository) MarkUploadCompleted(context.Context, uuid.UUID) error {
	return nil
}

func (f *fakeUploadRepository) CreateFile(_ context.Context, file *uploads.File) error {
	f.createdFile = file
	return nil
}

func (f *fakeUploadRepository) GetFileByID(context.Context, uuid.UUID) (*uploads.File, error) {
	return nil, nil
}

func (f *fakeUploadRepository) CleanupExpiredUploads(context.Context, time.Time) error {
	return nil
}

type fakeStorageProvider struct {
	object *storage.ObjectInfo
}

func (f fakeStorageProvider) PresignedUploadURL(context.Context, string, string, time.Duration, int64) (string, error) {
	return "https://upload.example.test", nil
}

func (f fakeStorageProvider) PresignedDownloadURL(context.Context, string, time.Duration) (string, error) {
	return "https://download.example.test", nil
}

func (f fakeStorageProvider) HeadObject(context.Context, string) (*storage.ObjectInfo, error) {
	return f.object, nil
}

func (f fakeStorageProvider) DeleteObject(context.Context, string) error {
	return nil
}
