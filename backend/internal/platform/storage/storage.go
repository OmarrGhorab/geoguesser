package storage

import (
	"context"
	"time"
)

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

// Provider abstracts object storage operations.
type Provider interface {
	PresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration, maxSize int64) (string, error)
	PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	HeadObject(ctx context.Context, key string) (*ObjectInfo, error)
	DeleteObject(ctx context.Context, key string) error
}
