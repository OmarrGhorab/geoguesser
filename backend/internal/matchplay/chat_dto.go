package matchplay

import (
	"time"

	"github.com/google/uuid"
)

// SendMessageRequest is POST /matches/{matchId}/messages.
// Text is trimmed server-side; either non-empty text or one ready file is required.
type SendMessageRequest struct {
	ClientMessageID uuid.UUID  `json:"client_message_id"`
	Text            *string    `json:"text,omitempty"`
	FileID          *uuid.UUID `json:"file_id,omitempty"`
}

// TeamMessageResponse wraps a single team message for create/list payloads.
type TeamMessageResponse struct {
	ID         uuid.UUID             `json:"id"`
	Sequence   int64                 `json:"sequence"`
	Author     MessageAuthorDTO      `json:"author"`
	Text       string                `json:"text"`
	Attachment *MessageAttachmentDTO `json:"attachment"`
	Status     string                `json:"status"`
	CreatedAt  time.Time             `json:"created_at"`
}

// MessageAuthorDTO is the public author projection on a chat message.
type MessageAuthorDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
}

// MessageAttachmentDTO is attachment metadata without storage keys or raw URLs.
type MessageAttachmentDTO struct {
	FileID      uuid.UUID `json:"file_id"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
}

// MessageListResponse is GET /matches/{matchId}/messages.
type MessageListResponse struct {
	Data []TeamMessageResponse `json:"data"`
	Page MessagePageInfo       `json:"page"`
}

// MessagePageInfo carries sequence pagination metadata.
// NextCursor is the last sequence on the page as a decimal string when more may exist.
type MessagePageInfo struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor"`
}

// AttachmentURLResponse is GET /matches/{matchId}/messages/{messageId}/attachment.
type AttachmentURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ReportMessageRequest is POST /matches/{matchId}/messages/{messageId}/report.
type ReportMessageRequest struct {
	Reason string `json:"reason"`
}

// ReportMessageResponse is the 202 accepted report acknowledgement.
// Reporter identity is never broadcast; this body stays minimal.
type ReportMessageResponse struct {
	Status string `json:"status"`
}

// SendMessageResult is the service outcome for send (create vs idempotent replay).
type SendMessageResult struct {
	Message TeamMessageResponse
	Created bool
}

// ToTeamMessageResponse projects a durable message plus optional file and display name.
func ToTeamMessageResponse(msg TeamMessage, displayName string, file *ChatFile) TeamMessageResponse {
	out := TeamMessageResponse{
		ID:       msg.ID,
		Sequence: msg.Sequence,
		Author: MessageAuthorDTO{
			UserID:      msg.AuthorUserID,
			DisplayName: displayName,
		},
		Text:      msg.Text,
		Status:    msg.Status,
		CreatedAt: msg.CreatedAt.UTC(),
	}
	if file != nil && msg.FileID != nil {
		out.Attachment = &MessageAttachmentDTO{
			FileID:      file.ID,
			ContentType: file.ContentType,
			SizeBytes:   file.SizeBytes,
		}
	}
	return out
}
