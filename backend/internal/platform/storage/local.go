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
//
// Objects are always private: there is no public URL surface for derivatives.
// PresignedDownloadURL returns a file:// path for local tooling only.
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
// support direct browser uploads; tests should write files via PutObject.
func (l *LocalProvider) PresignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration, maxSize int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", MapContextError(ctx, err)
	}
	return fmt.Sprintf("local://%s", filepath.Join(l.basePath, key)), nil
}

// PresignedDownloadURL returns the local file path as a file:// URL.
// This is not a public CDN/derivative URL; access still requires filesystem rights.
func (l *LocalProvider) PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", MapContextError(ctx, err)
	}
	return fmt.Sprintf("file://%s", filepath.Join(l.basePath, key)), nil
}

// HeadObject returns metadata for a local file.
func (l *LocalProvider) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, MapContextError(ctx, err)
	}
	path := filepath.Join(l.basePath, key)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	ct := readContentTypeMeta(path)
	return &ObjectInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  ct,
		LastModified: stat.ModTime(),
	}, nil
}

// GetObject returns a bounded reader for a local file.
func (l *LocalProvider) GetObject(ctx context.Context, key string, maxBytes int64) (io.ReadCloser, *ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, MapContextError(ctx, err)
	}
	if maxBytes <= 0 {
		return nil, nil, ErrObjectTooLarge
	}
	path := filepath.Join(l.basePath, key)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to stat local file: %w", err)
	}
	if stat.Size() > maxBytes {
		return nil, nil, ErrObjectTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to open local file: %w", err)
	}
	info := &ObjectInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  readContentTypeMeta(path),
		LastModified: stat.ModTime(),
	}
	return &boundedReadCloser{rc: f, r: io.LimitReader(f, maxBytes)}, info, nil
}

// PutObject writes data privately to a local file. No public URL is minted.
func (l *LocalProvider) PutObject(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	if err := ctx.Err(); err != nil {
		return MapContextError(ctx, err)
	}
	path := filepath.Join(l.basePath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	// Write to a temp file then rename for crash-safe private storage.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".put-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	var written int64
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			_ = tmp.Close()
			return MapContextError(ctx, err)
		}
		n, readErr := body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				_ = tmp.Close()
				return fmt.Errorf("failed to write file: %w", werr)
			}
			written += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = tmp.Close()
			return fmt.Errorf("failed to read body: %w", readErr)
		}
	}
	if size >= 0 && written != size {
		_ = tmp.Close()
		return fmt.Errorf("put object size mismatch: wrote %d, expected %d", written, size)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to finalize object: %w", err)
	}
	cleanup = false
	if contentType != "" {
		_ = writeContentTypeMeta(path, contentType)
	}
	return nil
}

// DeleteObject removes a local file (and optional content-type sidecar).
func (l *LocalProvider) DeleteObject(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return MapContextError(ctx, err)
	}
	path := filepath.Join(l.basePath, key)
	_ = os.Remove(contentTypeMetaPath(path))
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to delete local file: %w", err)
	}
	return nil
}

// WriteObject writes data directly to a local file. Deprecated for new code:
// prefer PutObject. Kept for existing tests that call it via type assertion.
func (l *LocalProvider) WriteObject(ctx context.Context, key string, r io.Reader) error {
	return l.PutObject(ctx, key, "application/octet-stream", r, -1)
}

func contentTypeMetaPath(objectPath string) string {
	return objectPath + ".content-type"
}

func writeContentTypeMeta(objectPath, contentType string) error {
	return os.WriteFile(contentTypeMetaPath(objectPath), []byte(contentType), 0o644)
}

func readContentTypeMeta(objectPath string) string {
	data, err := os.ReadFile(contentTypeMetaPath(objectPath))
	if err != nil || len(data) == 0 {
		return "application/octet-stream"
	}
	return string(data)
}

// boundedReadCloser limits reads and closes the underlying closer.
type boundedReadCloser struct {
	rc io.ReadCloser
	r  io.Reader
}

func (b *boundedReadCloser) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func (b *boundedReadCloser) Close() error {
	return b.rc.Close()
}
