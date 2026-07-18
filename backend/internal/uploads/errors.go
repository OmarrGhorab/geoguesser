package uploads

import "errors"

// Domain errors used by the uploads package.
var (
	ErrFileNameRequired       = errors.New("file_name is required")
	ErrContentTypeRequired    = errors.New("content_type is required")
	ErrInvalidSize            = errors.New("size_bytes must be positive")
	ErrFileTooLarge           = errors.New("file exceeds maximum size")
	ErrInvalidContentType     = errors.New("content_type is not allowed")
	ErrUploadNotFound         = errors.New("upload not found")
	ErrUploadExpired          = errors.New("upload expired")
	ErrUploadAlreadyComplete  = errors.New("upload already complete")
	ErrFileNotFound           = errors.New("file not found")
	ErrObjectNotFound         = errors.New("stored object not found")
	ErrObjectMetadataMismatch = errors.New("stored object metadata does not match upload")
	ErrForbidden              = errors.New("you do not have access to this file")

	// Team-chat / sanitization errors.
	ErrInvalidPurpose         = errors.New("invalid upload purpose")
	ErrContextIDRequired      = errors.New("context_id is required for team_chat uploads")
	ErrInvalidContextID       = errors.New("context_id is invalid")
	ErrImagesDisabled         = errors.New("team chat images are disabled")
	ErrMatchAccessDenied      = errors.New("not an active participant of this match")
	ErrInvalidImage           = errors.New("invalid image")
	ErrUnsafeImage            = errors.New("unsafe image")
	ErrImageTooManyPixels     = errors.New("image exceeds maximum pixel count")
	ErrSanitizedTooLarge      = errors.New("sanitized image exceeds maximum size")
	ErrAttachmentNotReady     = errors.New("attachment not ready")
	ErrStorageUnavailable     = errors.New("object storage unavailable")
	ErrPurposeContextMismatch = errors.New("file purpose or context does not match")
)

// errorsIs is a package-local alias to keep image_sanitizer free of import cycles noise.
func errorsIs(err, target error) bool { return errors.Is(err, target) }
