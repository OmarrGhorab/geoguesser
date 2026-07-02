package leaderboards

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	duration := int64(1234)
	completedAt := time.Date(2026, 7, 2, 12, 34, 56, 789, time.UTC)

	cursor := encodeCursor(4200, &duration, completedAt, id)
	decoded, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decodeCursor failed: %v", err)
	}
	if decoded.Score != 4200 || decoded.StableID != id || decoded.CompletionDurationMS == nil || *decoded.CompletionDurationMS != duration || !decoded.CompletedAt.Equal(completedAt) {
		t.Fatalf("decoded cursor = %+v, want score=4200 duration=%d completed_at=%s id=%s", decoded, duration, completedAt, id)
	}
}

func TestCursorRoundTripWithNullDuration(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000043")
	completedAt := time.Date(2026, 7, 2, 12, 34, 56, 0, time.UTC)

	cursor := encodeCursor(4100, nil, completedAt, id)
	decoded, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decodeCursor failed: %v", err)
	}
	if decoded.Score != 4100 || decoded.StableID != id || decoded.CompletionDurationMS != nil || !decoded.CompletedAt.Equal(completedAt) {
		t.Fatalf("decoded cursor = %+v, want score=4100 null duration completed_at=%s id=%s", decoded, completedAt, id)
	}
}

func TestDecodeCursorRejectsMalformedInput(t *testing.T) {
	if _, err := decodeCursor("not-base64"); err == nil {
		t.Fatal("expected malformed cursor to fail")
	}
}
