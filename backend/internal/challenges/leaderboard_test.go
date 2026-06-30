package challenges

import (
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLeaderboardOrderingTieBreakers(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	fast := int64(5000)
	slow := int64(7000)
	entries := []LeaderboardEntry{
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Score: 1000, CompletionDurationMS: &slow, CompletedAt: now},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Score: 1200, CompletionDurationMS: &slow, CompletedAt: now},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Score: 1200, CompletionDurationMS: &fast, CompletedAt: now.Add(time.Second)},
	}

	sort.Slice(entries, func(i, j int) bool {
		return leaderboardLess(entries[i], entries[j])
	})

	if got := entries[0].AttemptID.String(); got != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("first attempt = %s", got)
	}
	if got := entries[1].AttemptID.String(); got != "00000000-0000-0000-0000-000000000002" {
		t.Fatalf("second attempt = %s", got)
	}
}
