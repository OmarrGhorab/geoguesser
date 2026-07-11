package friends

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	maxCursorLen = 512
)

// listCursor is an opaque pagination cursor for relationship lists ordered by
// created_at DESC, id DESC.
type listCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > maxLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

func validateCursor(cursor string) error {
	if len(cursor) > maxCursorLen {
		return ErrInvalidCursor
	}
	if cursor == "" {
		return nil
	}
	if _, err := decodeListCursor(cursor); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

func encodeListCursor(createdAt time.Time, id uuid.UUID) string {
	raw := strings.Join([]string{
		createdAt.UTC().Format(time.RFC3339Nano),
		id.String(),
	}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeListCursor(cursor string) (*listCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	if createdAt.IsZero() || id == uuid.Nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &listCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
}
