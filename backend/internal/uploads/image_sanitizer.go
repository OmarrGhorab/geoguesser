package uploads

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	"github.com/raven/geoguess/backend/internal/platform/storage"
)

// Supported team-chat image MIME types (declared and detected).
const (
	MIMEJPEG = "image/jpeg"
	MIMEPNG  = "image/png"
	MIMEWebP = "image/webp"
	// SanitizedDerivativeMIME is always produced for team-chat attachments.
	SanitizedDerivativeMIME = MIMEJPEG
)

// SanitizeLimits bounds technical image sanitization.
type SanitizeLimits struct {
	MaxInputBytes  int64
	MaxPixels      int
	MaxDimension   int
	MaxOutputBytes int64
	JPEGQuality    int
}

// DefaultSanitizeLimits returns production defaults (5 MB / 20 MP / 2048 px).
func DefaultSanitizeLimits() SanitizeLimits {
	return SanitizeLimits{
		MaxInputBytes:  5 * 1024 * 1024,
		MaxPixels:      20_000_000,
		MaxDimension:   2048,
		MaxOutputBytes: 5 * 1024 * 1024,
		JPEGQuality:    85,
	}
}

// SanitizeResult is a re-encoded private JPEG derivative with metadata stripped.
type SanitizeResult struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
	SizeBytes   int64
}

// DetectImageMIME verifies magic bytes and returns a canonical MIME type.
// Unsupported or corrupt headers return ErrInvalidImage.
func DetectImageMIME(data []byte) (string, error) {
	if len(data) < 12 {
		return "", ErrInvalidImage
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return MIMEJPEG, nil
	}
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return MIMEPNG, nil
	}
	// WebP: RIFF....WEBP
	if bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return MIMEWebP, nil
	}
	return "", ErrInvalidImage
}

// NormalizeDeclaredMIME lowercases and trims a Content-Type value (no params).
func NormalizeDeclaredMIME(declared string) string {
	declared = strings.TrimSpace(strings.ToLower(declared))
	if i := strings.IndexByte(declared, ';'); i >= 0 {
		declared = strings.TrimSpace(declared[:i])
	}
	return declared
}

// SanitizeImage verifies magic bytes, rejects MIME spoofing, decodes, enforces
// pixel/dimension limits, resizes the longest edge, re-encodes as JPEG (stripping
// metadata), and enforces the output size cap.
func SanitizeImage(ctx context.Context, raw []byte, declaredMIME string, limits SanitizeLimits) (*SanitizeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limits.MaxInputBytes <= 0 || limits.MaxPixels <= 0 || limits.MaxDimension <= 0 || limits.MaxOutputBytes <= 0 {
		return nil, fmt.Errorf("invalid sanitize limits")
	}
	if limits.JPEGQuality <= 0 || limits.JPEGQuality > 100 {
		limits.JPEGQuality = 85
	}
	if int64(len(raw)) == 0 {
		return nil, ErrInvalidImage
	}
	if int64(len(raw)) > limits.MaxInputBytes {
		return nil, ErrFileTooLarge
	}

	detected, err := DetectImageMIME(raw)
	if err != nil {
		return nil, err
	}
	declared := NormalizeDeclaredMIME(declaredMIME)
	if declared == "" {
		return nil, ErrContentTypeRequired
	}
	// Reject MIME spoofing: declared type must match magic-byte detection.
	// Allow image/jpg as alias for image/jpeg.
	if !mimeMatches(declared, detected) {
		return nil, ErrUnsafeImage
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrInvalidImage
	}
	_ = format // format is informational; magic bytes already verified

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidImage
	}
	pixels := width * height
	if pixels > limits.MaxPixels {
		return nil, ErrImageTooManyPixels
	}

	outImg := resizeLongestEdge(img, limits.MaxDimension)
	outBounds := outImg.Bounds()
	outW, outH := outBounds.Dx(), outBounds.Dy()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, outImg, &jpeg.Options{Quality: limits.JPEGQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode: %w", err)
	}
	if int64(buf.Len()) > limits.MaxOutputBytes {
		// Retry at lower quality once before rejecting.
		buf.Reset()
		quality := limits.JPEGQuality
		if quality > 60 {
			quality = 60
		} else {
			quality = 40
		}
		if err := jpeg.Encode(&buf, outImg, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("jpeg encode retry: %w", err)
		}
		if int64(buf.Len()) > limits.MaxOutputBytes {
			return nil, ErrSanitizedTooLarge
		}
	}

	data := buf.Bytes()
	return &SanitizeResult{
		Data:        data,
		ContentType: SanitizedDerivativeMIME,
		Width:       outW,
		Height:      outH,
		SizeBytes:   int64(len(data)),
	}, nil
}

func mimeMatches(declared, detected string) bool {
	if declared == "image/jpg" {
		declared = MIMEJPEG
	}
	return declared == detected
}

// resizeLongestEdge scales img so the longest edge is at most maxDim.
// Returns img unchanged when already within limits. Uses high-quality CatmullRom.
func resizeLongestEdge(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxDim && h <= maxDim {
		// Copy into a new NRGBA to drop any exotic color models / ancillary data.
		return copyNRGBA(img)
	}
	scale := float64(maxDim) / float64(w)
	if h > w {
		scale = float64(maxDim) / float64(h)
	}
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func copyNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(dst, dst.Bounds(), img, bounds.Min, draw.Src)
	return dst
}

// ImageSanitizer reads a raw object, writes a sanitized derivative, and deletes the raw key.
type ImageSanitizer struct {
	provider storage.Provider
	limits   SanitizeLimits
	metrics  MetricsRecorder
}

// NewImageSanitizer returns a sanitizer bound to the given storage provider.
func NewImageSanitizer(provider storage.Provider, limits SanitizeLimits, metrics MetricsRecorder) *ImageSanitizer {
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &ImageSanitizer{provider: provider, limits: limits, metrics: metrics}
}

// SanitizeStoredObject downloads rawKey (bounded), sanitizes, puts derivativeKey
// as a private JPEG, then best-effort deletes rawKey. On sanitization failure the
// raw object is still deleted when removeRawOnFailure is true.
func (s *ImageSanitizer) SanitizeStoredObject(
	ctx context.Context,
	rawKey, derivativeKey, declaredMIME string,
	removeRawOnFailure bool,
) (*SanitizeResult, error) {
	return s.sanitizeStoredObject(ctx, rawKey, derivativeKey, declaredMIME, removeRawOnFailure, true)
}

// SanitizeStoredObjectDeferredCleanup stores the sanitized derivative but leaves
// the raw object until the caller commits its database finalization.
func (s *ImageSanitizer) SanitizeStoredObjectDeferredCleanup(
	ctx context.Context,
	rawKey, derivativeKey, declaredMIME string,
	removeRawOnFailure bool,
) (*SanitizeResult, error) {
	return s.sanitizeStoredObject(ctx, rawKey, derivativeKey, declaredMIME, removeRawOnFailure, false)
}

func (s *ImageSanitizer) sanitizeStoredObject(
	ctx context.Context,
	rawKey, derivativeKey, declaredMIME string,
	removeRawOnFailure, removeRawOnSuccess bool,
) (*SanitizeResult, error) {
	start := nowUTC()
	outcome := "error"
	defer func() {
		s.metrics.ObserveSanitize(outcome, since(start))
	}()

	if s.provider == nil {
		outcome = "storage_unavailable"
		return nil, ErrStorageUnavailable
	}

	rc, info, err := s.provider.GetObject(ctx, rawKey, s.limits.MaxInputBytes)
	if err != nil {
		if errorsIs(err, storage.ErrObjectTooLarge) {
			outcome = "too_large"
			_ = s.provider.DeleteObject(ctx, rawKey)
			return nil, ErrFileTooLarge
		}
		if errorsIs(err, storage.ErrObjectNotFound) {
			outcome = "not_found"
			return nil, ErrObjectNotFound
		}
		if errorsIs(err, storage.ErrCanceled) {
			outcome = "canceled"
			return nil, err
		}
		outcome = "storage_unavailable"
		return nil, ErrStorageUnavailable
	}
	raw, readErr := storage.ReadAllBounded(rc, s.limits.MaxInputBytes)
	_ = rc.Close()
	if readErr != nil {
		if errorsIs(readErr, storage.ErrObjectTooLarge) {
			outcome = "too_large"
			_ = s.provider.DeleteObject(ctx, rawKey)
			return nil, ErrFileTooLarge
		}
		outcome = "storage_unavailable"
		return nil, ErrStorageUnavailable
	}
	if info != nil && info.Size > 0 && int64(len(raw)) != info.Size && int64(len(raw)) < info.Size {
		// truncated read
		outcome = "invalid"
		if removeRawOnFailure {
			_ = s.provider.DeleteObject(ctx, rawKey)
		}
		return nil, ErrInvalidImage
	}

	result, err := SanitizeImage(ctx, raw, declaredMIME, s.limits)
	if err != nil {
		outcome = sanitizeOutcome(err)
		if removeRawOnFailure {
			_ = s.deleteRaw(ctx, rawKey)
		}
		return nil, err
	}

	if err := s.provider.PutObject(ctx, derivativeKey, result.ContentType, bytes.NewReader(result.Data), result.SizeBytes); err != nil {
		outcome = "storage_unavailable"
		// Leave raw in place for retry/cleanup worker when derivative write fails.
		return nil, ErrStorageUnavailable
	}

	// Callers that still have a DB transition to commit defer raw deletion.
	if removeRawOnSuccess {
		if err := s.deleteRaw(ctx, rawKey); err != nil {
			// Derivative is stored; log via metric label but still return success so
			// attachment readiness is not blocked. Cleanup worker sweeps leftovers.
			s.metrics.ObserveRawCleanup("retry")
		} else {
			s.metrics.ObserveRawCleanup("deleted")
		}
	}

	outcome = "ready"
	return result, nil
}

func (s *ImageSanitizer) deleteRaw(ctx context.Context, rawKey string) error {
	if rawKey == "" {
		return nil
	}
	err := s.provider.DeleteObject(ctx, rawKey)
	if err == nil || errorsIs(err, storage.ErrObjectNotFound) {
		return nil
	}
	return err
}

func sanitizeOutcome(err error) string {
	switch {
	case errorsIs(err, ErrUnsafeImage):
		return "unsafe"
	case errorsIs(err, ErrInvalidImage), errorsIs(err, ErrImageTooManyPixels):
		return "invalid"
	case errorsIs(err, ErrFileTooLarge), errorsIs(err, ErrSanitizedTooLarge):
		return "too_large"
	default:
		return "error"
	}
}
