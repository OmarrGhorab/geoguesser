package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalProvider stores objects on the local filesystem. It is intended for
// development and tests, not production.
type LocalProvider struct {
	basePath string
}

// NewLocalProvider returns a local filesystem storage provider.
func NewLocalProvider(basePath string) (*LocalProvider, error) {
	if basePath == "" {
		return nil, fmt.Errorf("local storage base path is required")
	}
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create local storage directory: %w", err)
	}
	return &LocalProvider{basePath: basePath}, nil
}

// PresignedUploadURL returns a dummy presigned URL. LocalProvider does not
// support direct browser uploads; tests should write files directly.
func (l *LocalProvider) PresignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration, maxSize int64) (string, error) {
	return fmt.Sprintf("local://%s", filepath.Join(l.basePath, key)), nil
}

// PresignedDownloadURL returns the local file path as the URL.
func (l *LocalProvider) PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	return fmt.Sprintf("file://%s", filepath.Join(l.basePath, key)), nil
}

// HeadObject returns metadata for a local file.
func (l *LocalProvider) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	path := filepath.Join(l.basePath, key)
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	return &ObjectInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  "application/octet-stream",
		LastModified: stat.ModTime(),
	}, nil
}

// DeleteObject removes a local file.
func (l *LocalProvider) DeleteObject(ctx context.Context, key string) error {
	path := filepath.Join(l.basePath, key)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete local file: %w", err)
	}
	return nil
}

// WriteObject writes data directly to a local file. This helper is only
// available for the local provider and is useful in tests.
func (l *LocalProvider) WriteObject(ctx context.Context, key string, r io.Reader) error {
	path := filepath.Join(l.basePath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
