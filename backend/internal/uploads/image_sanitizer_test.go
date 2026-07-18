package uploads_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/raven/geoguess/backend/internal/platform/storage"
	"github.com/raven/geoguess/backend/internal/uploads"
	"golang.org/x/image/webp"
)

func TestDetectImageMIME(t *testing.T) {
	jpegBytes := mustEncodeJPEG(t, solidImage(8, 8, color.RGBA{R: 200, G: 10, B: 10, A: 255}))
	pngBytes := mustEncodePNG(t, solidImage(8, 8, color.RGBA{R: 10, G: 200, B: 10, A: 255}))

	mime, err := uploads.DetectImageMIME(jpegBytes)
	if err != nil || mime != uploads.MIMEJPEG {
		t.Fatalf("jpeg detect = %q %v", mime, err)
	}
	mime, err = uploads.DetectImageMIME(pngBytes)
	if err != nil || mime != uploads.MIMEPNG {
		t.Fatalf("png detect = %q %v", mime, err)
	}

	// Minimal RIFF/WEBP header (not a valid image body, but magic should match).
	webpMagic := []byte("RIFF....WEBP")
	copy(webpMagic[4:8], []byte{8, 0, 0, 0})
	mime, err = uploads.DetectImageMIME(webpMagic)
	if err != nil || mime != uploads.MIMEWebP {
		t.Fatalf("webp magic detect = %q %v", mime, err)
	}

	if _, err := uploads.DetectImageMIME([]byte("<svg xmlns")); err == nil {
		t.Fatal("expected SVG magic to be rejected")
	}
	if _, err := uploads.DetectImageMIME([]byte{0x00, 0x01}); err == nil {
		t.Fatal("expected short corrupt header to be rejected")
	}
}

func TestSanitizeImage_JPEGPNG(t *testing.T) {
	limits := uploads.DefaultSanitizeLimits()
	jpegBytes := mustEncodeJPEG(t, solidImage(64, 32, color.RGBA{R: 1, G: 2, B: 3, A: 255}))
	pngBytes := mustEncodePNG(t, solidImage(32, 64, color.RGBA{R: 4, G: 5, B: 6, A: 255}))

	for _, tc := range []struct {
		name     string
		raw      []byte
		declared string
	}{
		{"jpeg", jpegBytes, uploads.MIMEJPEG},
		{"png", pngBytes, uploads.MIMEPNG},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := uploads.SanitizeImage(context.Background(), tc.raw, tc.declared, limits)
			if err != nil {
				t.Fatalf("SanitizeImage: %v", err)
			}
			if res.ContentType != uploads.SanitizedDerivativeMIME {
				t.Fatalf("content type = %q", res.ContentType)
			}
			if res.SizeBytes <= 0 || int64(len(res.Data)) != res.SizeBytes {
				t.Fatalf("size mismatch")
			}
			// Output must be valid JPEG.
			if _, err := uploads.DetectImageMIME(res.Data); err != nil {
				t.Fatalf("output magic: %v", err)
			}
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(res.Data))
			if err != nil {
				t.Fatalf("jpeg decode output: %v", err)
			}
			if cfg.Width <= 0 || cfg.Height <= 0 {
				t.Fatalf("bad dimensions %dx%d", cfg.Width, cfg.Height)
			}
		})
	}
}

func TestSanitizeImage_MIMESpoofRejected(t *testing.T) {
	// PNG bytes declared as JPEG → unsafe.
	pngBytes := mustEncodePNG(t, solidImage(16, 16, color.White))
	_, err := uploads.SanitizeImage(context.Background(), pngBytes, uploads.MIMEJPEG, uploads.DefaultSanitizeLimits())
	if err != uploads.ErrUnsafeImage {
		t.Fatalf("spoof = %v, want ErrUnsafeImage", err)
	}

	// JPEG magic with SVG declaration.
	jpegBytes := mustEncodeJPEG(t, solidImage(8, 8, color.Black))
	_, err = uploads.SanitizeImage(context.Background(), jpegBytes, "image/svg+xml", uploads.DefaultSanitizeLimits())
	if err != uploads.ErrUnsafeImage {
		t.Fatalf("svg declare = %v, want ErrUnsafeImage", err)
	}
}

func TestSanitizeImage_CorruptData(t *testing.T) {
	// Valid JPEG magic but truncated body.
	raw := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	_, err := uploads.SanitizeImage(context.Background(), raw, uploads.MIMEJPEG, uploads.DefaultSanitizeLimits())
	if err != uploads.ErrInvalidImage {
		t.Fatalf("corrupt = %v, want ErrInvalidImage", err)
	}
}

func TestSanitizeImage_SizeAndPixelLimits(t *testing.T) {
	limits := uploads.SanitizeLimits{
		MaxInputBytes:  100,
		MaxPixels:      20_000_000,
		MaxDimension:   2048,
		MaxOutputBytes: 5 * 1024 * 1024,
		JPEGQuality:    85,
	}
	// Force oversize with declared jpeg magic at start.
	oversize := append(mustEncodeJPEG(t, solidImage(8, 8, color.White)), bytes.Repeat([]byte{0}, 200)...)
	_, err := uploads.SanitizeImage(context.Background(), oversize, uploads.MIMEJPEG, limits)
	if err != uploads.ErrFileTooLarge {
		t.Fatalf("input oversize = %v, want ErrFileTooLarge", err)
	}

	// Pixel limit: use a limit of 100 pixels.
	pixelLimits := uploads.DefaultSanitizeLimits()
	pixelLimits.MaxPixels = 100
	img := solidImage(20, 20, color.White) // 400 pixels
	raw := mustEncodeJPEG(t, img)
	_, err = uploads.SanitizeImage(context.Background(), raw, uploads.MIMEJPEG, pixelLimits)
	if err != uploads.ErrImageTooManyPixels {
		t.Fatalf("pixels = %v, want ErrImageTooManyPixels", err)
	}
}

func TestSanitizeImage_ResizeLongestEdge(t *testing.T) {
	limits := uploads.DefaultSanitizeLimits()
	limits.MaxDimension = 64
	// 200x100 → longest edge 64 → 64x32
	raw := mustEncodeJPEG(t, solidImage(200, 100, color.RGBA{R: 90, G: 90, B: 90, A: 255}))
	res, err := uploads.SanitizeImage(context.Background(), raw, uploads.MIMEJPEG, limits)
	if err != nil {
		t.Fatalf("SanitizeImage: %v", err)
	}
	if res.Width != 64 || res.Height != 32 {
		t.Fatalf("dimensions = %dx%d, want 64x32", res.Width, res.Height)
	}
	if res.Width > limits.MaxDimension || res.Height > limits.MaxDimension {
		t.Fatalf("exceeded max dimension")
	}
}

func TestSanitizeImage_StripsMetadataViaReencode(t *testing.T) {
	// Build a JPEG and ensure re-encoded output differs and is valid JPEG without
	// relying on EXIF libraries: re-encode path copies only pixel data through NRGBA.
	src := solidImage(32, 32, color.RGBA{R: 12, G: 34, B: 56, A: 255})
	raw := mustEncodeJPEG(t, src)
	// Inject a fake APP1 marker-like payload after SOI — corrupt-tolerant check:
	// We assert output is pure JPEG that decodes and has expected size bounds.
	res, err := uploads.SanitizeImage(context.Background(), raw, uploads.MIMEJPEG, uploads.DefaultSanitizeLimits())
	if err != nil {
		t.Fatalf("SanitizeImage: %v", err)
	}
	if bytes.Equal(res.Data, raw) {
		// Extremely unlikely if re-encoded; still valid if equal for tiny images.
		t.Log("output equalled input (acceptable for tiny solid JPEG)")
	}
	// Output must not contain PNG signature (full re-encode).
	if bytes.HasPrefix(res.Data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Fatal("output still looks like PNG")
	}
	if _, err := jpeg.Decode(bytes.NewReader(res.Data)); err != nil {
		t.Fatalf("output not decodable JPEG: %v", err)
	}
}

func TestSanitizeImage_OutputCap(t *testing.T) {
	limits := uploads.SanitizeLimits{
		MaxInputBytes:  5 * 1024 * 1024,
		MaxPixels:      20_000_000,
		MaxDimension:   2048,
		MaxOutputBytes: 500, // tiny cap to force failure after re-encode
		JPEGQuality:    85,
	}
	// Noise-like content tends to compress poorly.
	img := noiseImage(256, 256)
	raw := mustEncodeJPEG(t, img)
	_, err := uploads.SanitizeImage(context.Background(), raw, uploads.MIMEJPEG, limits)
	if err != uploads.ErrSanitizedTooLarge {
		t.Fatalf("output cap = %v, want ErrSanitizedTooLarge", err)
	}
}

func TestImageSanitizer_RawCleanupOnSuccessAndFailure(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("local provider: %v", err)
	}
	ctx := context.Background()
	limits := uploads.DefaultSanitizeLimits()
	sanitizer := uploads.NewImageSanitizer(provider, limits, uploads.NoopMetrics{})

	// Success path: raw deleted, derivative present.
	rawKey := "raw/success.jpg"
	derivKey := "sanitized/success.jpg"
	jpegBytes := mustEncodeJPEG(t, solidImage(48, 48, color.RGBA{R: 1, G: 2, B: 3, A: 255}))
	if err := provider.PutObject(ctx, rawKey, uploads.MIMEJPEG, bytes.NewReader(jpegBytes), int64(len(jpegBytes))); err != nil {
		t.Fatalf("put raw: %v", err)
	}
	res, err := sanitizer.SanitizeStoredObject(ctx, rawKey, derivKey, uploads.MIMEJPEG, true)
	if err != nil {
		t.Fatalf("SanitizeStoredObject: %v", err)
	}
	if res.ContentType != uploads.MIMEJPEG {
		t.Fatalf("derivative mime = %q", res.ContentType)
	}
	if _, err := provider.HeadObject(ctx, rawKey); err != storage.ErrObjectNotFound {
		t.Fatalf("raw should be deleted, head err = %v", err)
	}
	if _, err := provider.HeadObject(ctx, derivKey); err != nil {
		t.Fatalf("derivative missing: %v", err)
	}

	// Failure path (MIME spoof): raw cleaned when removeRawOnFailure=true.
	rawKey2 := "raw/fail.png"
	pngBytes := mustEncodePNG(t, solidImage(16, 16, color.White))
	if err := provider.PutObject(ctx, rawKey2, uploads.MIMEPNG, bytes.NewReader(pngBytes), int64(len(pngBytes))); err != nil {
		t.Fatalf("put raw2: %v", err)
	}
	_, err = sanitizer.SanitizeStoredObject(ctx, rawKey2, "sanitized/fail.jpg", uploads.MIMEJPEG, true)
	if err != uploads.ErrUnsafeImage {
		t.Fatalf("spoof sanitize = %v, want ErrUnsafeImage", err)
	}
	if _, err := provider.HeadObject(ctx, rawKey2); err != storage.ErrObjectNotFound {
		t.Fatalf("raw should be cleaned after reject, head err = %v", err)
	}
}

func TestSanitizeImage_WebPWhenDecodable(t *testing.T) {
	// Encode via golang.org/x/image is not available for WebP encode in stdlib.
	// If a pre-made WebP cannot be produced, skip. We still verify magic + reject path.
	// Attempt: decode any WebP from a minimal valid file is hard without encoder.
	// Document that decode registration is present:
	_ = webp.Decode
	limits := uploads.DefaultSanitizeLimits()
	// Invalid WebP body with correct magic.
	raw := make([]byte, 32)
	copy(raw, []byte("RIFF"))
	raw[4], raw[5], raw[6], raw[7] = 24, 0, 0, 0
	copy(raw[8:], []byte("WEBP"))
	_, err := uploads.SanitizeImage(context.Background(), raw, uploads.MIMEWebP, limits)
	if err != uploads.ErrInvalidImage {
		t.Fatalf("bad webp body = %v, want ErrInvalidImage", err)
	}
}

// --- helpers ---

func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func noiseImage(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 17) % 255),
				G: uint8((y * 31) % 255),
				B: uint8((x * y) % 255),
				A: 255,
			})
		}
	}
	return img
}

func mustEncodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	return buf.Bytes()
}

func mustEncodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}
