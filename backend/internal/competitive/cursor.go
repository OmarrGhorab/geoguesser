package competitive

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLeaderboardLimit = 50
	maxLeaderboardLimit     = 100
	defaultHistoryLimit     = 20
	maxHistoryLimit         = 100
	maxCursorLen            = 512
)

// leaderboardCursor is opaque pagination for eligible standings ordered by
// rating DESC, wins DESC, rating_reached_at ASC, user_id ASC.
type leaderboardCursor struct {
	Rating          int
	Wins            int
	RatingReachedAt time.Time
	UserID          uuid.UUID
}

// historyCursor is opaque pagination for rating changes newest first
// (created_at DESC, id DESC).
type historyCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func normalizeLeaderboardLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLeaderboardLimit, nil
	}
	if limit < 1 || limit > maxLeaderboardLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

func normalizeHistoryLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultHistoryLimit, nil
	}
	if limit < 1 || limit > maxHistoryLimit {
		return 0, ErrInvalidLimit
	}
	return limit, nil
}

func validateCursorString(cursor string) error {
	if len(cursor) > maxCursorLen {
		return ErrInvalidCursor
	}
	return nil
}

func encodeLeaderboardCursor(rating, wins int, reachedAt time.Time, userID uuid.UUID) string {
	raw := strings.Join([]string{
		strconv.Itoa(rating),
		strconv.Itoa(wins),
		reachedAt.UTC().Format(time.RFC3339Nano),
		userID.String(),
	}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeLeaderboardCursor(cursor string) (*leaderboardCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(raw), "|", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid cursor")
	}
	rating, err := strconv.Atoi(parts[0])
	if err != nil || rating < 0 {
		return nil, fmt.Errorf("invalid cursor")
	}
	wins, err := strconv.Atoi(parts[1])
	if err != nil || wins < 0 {
		return nil, fmt.Errorf("invalid cursor")
	}
	reachedAt, err := time.Parse(time.RFC3339Nano, parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	userID, err := uuid.Parse(parts[3])
	if err != nil || userID == uuid.Nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &leaderboardCursor{
		Rating:          rating,
		Wins:            wins,
		RatingReachedAt: reachedAt.UTC(),
		UserID:          userID,
	}, nil
}

func encodeHistoryCursor(createdAt time.Time, id uuid.UUID) string {
	raw := strings.Join([]string{
		createdAt.UTC().Format(time.RFC3339Nano),
		id.String(),
	}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeHistoryCursor(cursor string) (*historyCursor, error) {
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
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &historyCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
}
