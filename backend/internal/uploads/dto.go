package uploads

// CreateUploadRequest is the payload to create a presigned upload URL.
type CreateUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

// CreateUploadResponse returns the upload details and presigned URL.
type CreateUploadResponse struct {
	UploadID    string `json:"upload_id"`
	UploadURL   string `json:"upload_url"`
	ExpiresAt   string `json:"expires_at"`
	StorageKey  string `json:"storage_key"`
}

// CompleteUploadRequest is the payload to complete an upload.
type CompleteUploadRequest struct {
	UploadID string `json:"upload_id"`
}

// FileResponse is the public file metadata response.
type FileResponse struct {
	File FileDTO `json:"file"`
}

// FileDTO is the public file metadata.
type FileDTO struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	CreatedAt   string `json:"created_at"`
}

// SignedURLResponse returns a signed download URL.
type SignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}
