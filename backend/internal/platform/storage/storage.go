package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// Common storage errors.
var (
	// ErrObjectNotFound indicates the object key does not exist.
	ErrObjectNotFound = errors.New("storage object not found")
	// ErrObjectTooLarge indicates the stored object exceeds the requested read bound.
	ErrObjectTooLarge = errors.New("storage object exceeds max read size")
	// ErrCanceled indicates the operation was canceled via context.
	ErrCanceled = errors.New("storage operation canceled")
)

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

// Provider abstracts object storage operations.
//
// PutObject ALWAYS stores private objects (no public ACL / public derivative URL).
// Callers that need temporary access MUST use PresignedDownloadURL.
type Provider interface {
	PresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration, maxSize int64) (string, error)
	PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	HeadObject(ctx context.Context, key string) (*ObjectInfo, error)
	// GetObject returns a bounded reader for the object body. At most maxBytes
	// may be read. If the object ContentLength is known and exceeds maxBytes,
	// ErrObjectTooLarge is returned without streaming the body. The caller must
	// close the returned reader. maxBytes must be > 0.
	GetObject(ctx context.Context, key string, maxBytes int64) (io.ReadCloser, *ObjectInfo, error)
	// PutObject stores an object privately (no public ACL). size may be -1 when
	// the body length is unknown; prefer an exact size when available.
	PutObject(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	DeleteObject(ctx context.Context, key string) error
}

// ReadAllBounded reads at most maxBytes from r and returns the bytes.
// If more than maxBytes are available, it returns ErrObjectTooLarge.
func ReadAllBounded(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, ErrObjectTooLarge
	}
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrObjectTooLarge
	}
	return data, nil
}

// MapContextError maps context cancellation/deadline errors to ErrCanceled.
func MapContextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return ErrCanceled
	}
	return err
}
