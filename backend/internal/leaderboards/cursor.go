package leaderboards

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const nullDurationSortValue = int64(1<<63 - 1)

type leaderboardCursor struct {
	Score                int       `json:"score"`
	CompletionDurationMS *int64    `json:"duration_ms,omitempty"`
	CompletedAt          time.Time `json:"completed_at"`
	StableID             uuid.UUID `json:"id"`
}

func encodeCursor(score int, completionDurationMS *int64, completedAt time.Time, stableID uuid.UUID) string {
	duration := ""
	if completionDurationMS != nil {
		duration = strconv.FormatInt(*completionDurationMS, 10)
	}
	raw := strings.Join([]string{
		strconv.Itoa(score),
		duration,
		completedAt.UTC().Format(time.RFC3339Nano),
		stableID.String(),
	}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (*leaderboardCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(raw), "|", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid cursor")
	}
	score, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}
	var duration *int64
	if parts[1] != "" {
		parsedDuration, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || parsedDuration < 0 {
			return nil, fmt.Errorf("invalid cursor")
		}
		duration = &parsedDuration
	}
	completedAt, err := time.Parse(time.RFC3339Nano, parts[2])
	if err != nil {
		return nil, err
	}
	stableID, err := uuid.Parse(parts[3])
	if err != nil {
		return nil, err
	}
	parsed := leaderboardCursor{
		Score:                score,
		CompletionDurationMS: duration,
		CompletedAt:          completedAt,
		StableID:             stableID,
	}
	if parsed.Score < 0 || parsed.CompletedAt.IsZero() || parsed.StableID == uuid.Nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	return &parsed, nil
}

func cursorSortValues(cursor leaderboardCursor) []any {
	durationIsNull := cursor.CompletionDurationMS == nil
	durationValue := nullDurationSortValue
	if cursor.CompletionDurationMS != nil {
		durationValue = *cursor.CompletionDurationMS
	}
	return []any{
		-cursor.Score,
		durationIsNull,
		durationValue,
		cursor.CompletedAt.UTC(),
		cursor.StableID,
	}
}
