package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Provider implements storage.Provider for Cloudflare R2.
type R2Provider struct {
	client    *s3.Client
	bucket    string
	publicURL string
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
		return "", fmt.Errorf("failed to presign upload: %w", err)
	}
	return req.URL, nil
}

// PresignedDownloadURL returns a presigned URL for downloading an object.
func (r *R2Provider) PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	presigner := s3.NewPresignClient(r.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("failed to presign download: %w", err)
	}
	return req.URL, nil
}

// HeadObject returns metadata for an object if it exists.
func (r *R2Provider) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	out, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
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

// DeleteObject removes an object from R2.
func (r *R2Provider) DeleteObject(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
