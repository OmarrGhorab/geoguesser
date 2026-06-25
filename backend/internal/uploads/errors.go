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
)
