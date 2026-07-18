package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// R2Provider implements storage.Provider for Cloudflare R2.
//
// PutObject always stores private objects: no public ACL is set and no public
// derivative URL is returned. Clients must use PresignedDownloadURL for access.
type R2Provider struct {
	client    *s3.Client
	bucket    string
	publicURL string // retained for config compatibility; never used for chat derivatives
}

// NewR2Provider returns an R2 storage provider.
func NewR2Provider(accountID, accessKeyID, secretAccessKey, bucket, endpoint, publicURL string) (*R2Provider, error) {
	if accountID == "" || accessKeyID == "" || secretAccessKey == "" || bucket == "" {
		return nil, fmt.Errorf("R2 account id, access key, secret key, and bucket are required")
	}

	endpointURL := endpoint
	if endpointURL == "" {
		endpointURL = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpointURL)
	})

	return &R2Provider{
		client:    client,
		bucket:    bucket,
		publicURL: publicURL,
	}, nil
}

// PresignedUploadURL returns a presigned URL for uploading an object.
func (r *R2Provider) PresignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration, maxSize int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", MapContextError(ctx, err)
	}
	presigner := s3.NewPresignClient(r.client)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}
	if maxSize > 0 {
		input.ContentLength = aws.Int64(maxSize)
	}

	req, err := presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("failed to presign upload: %w", MapContextError(ctx, err))
	}
	return req.URL, nil
}

// PresignedDownloadURL returns a short-lived private signed URL (not a public derivative URL).
func (r *R2Provider) PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", MapContextError(ctx, err)
	}
	presigner := s3.NewPresignClient(r.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("failed to presign download: %w", MapContextError(ctx, err))
	}
	return req.URL, nil
}

// HeadObject returns metadata for an object if it exists.
func (r *R2Provider) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, MapContextError(ctx, err)
	}
	out, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to head object: %w", MapContextError(ctx, err))
	}

	info := &ObjectInfo{
		Key:         key,
		Size:        aws.ToInt64(out.ContentLength),
		ContentType: aws.ToString(out.ContentType),
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}
	return info, nil
}

// GetObject returns a bounded private object reader. No public URL is produced.
func (r *R2Provider) GetObject(ctx context.Context, key string, maxBytes int64) (io.ReadCloser, *ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, MapContextError(ctx, err)
	}
	if maxBytes <= 0 {
		return nil, nil, ErrObjectTooLarge
	}

	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to get object: %w", MapContextError(ctx, err))
	}

	size := aws.ToInt64(out.ContentLength)
	if size > maxBytes {
		_ = out.Body.Close()
		return nil, nil, ErrObjectTooLarge
	}

	info := &ObjectInfo{
		Key:         key,
		Size:        size,
		ContentType: aws.ToString(out.ContentType),
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}

	return &boundedReadCloser{rc: out.Body, r: io.LimitReader(out.Body, maxBytes)}, info, nil
}

// PutObject stores an object privately. ACL is never set to public-read and no
// public derivative URL is returned or constructed from publicURL.
func (r *R2Provider) PutObject(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	if err := ctx.Err(); err != nil {
		return MapContextError(ctx, err)
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
		Body:   body,
		// Intentionally no ACL: objects remain private. publicURL is never used.
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", MapContextError(ctx, err))
	}
	return nil
}

// DeleteObject removes an object from R2.
func (r *R2Provider) DeleteObject(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return MapContextError(ctx, err)
	}
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", MapContextError(ctx, err))
	}
	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nsk *types.NoSuchKey
	var nf *types.NotFound
	return errors.As(err, &nsk) || errors.As(err, &nf)
}
