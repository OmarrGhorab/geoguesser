package challenges

import (
	"testing"
	"time"
)

func TestApplyDailyCompletionStartsAndIncrements(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)

	started := ApplyDailyCompletion(nil, day, now)
	if started.CurrentCount != 1 || started.BestCount != 1 || started.Status != StreakStatusActive {
		t.Fatalf("started streak = %+v", started)
	}

	nextDay := day.AddDate(0, 0, 1)
	incremented := ApplyDailyCompletion(&started, nextDay, now.Add(24*time.Hour))
	if incremented.CurrentCount != 2 || incremented.BestCount != 2 {
		t.Fatalf("incremented streak = %+v", incremented)
	}
}

func TestApplyDailyCompletionBreaksAfterMissedDay(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	last := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	current := &Streak{CurrentCount: 3, BestCount: 3, LastCompletedChallengeDate: &last, Status: StreakStatusActive, ProtectionState: ProtectionNone}

	got := ApplyDailyCompletion(current, last.AddDate(0, 0, 2), now)
	if got.CurrentCount != 1 || got.BestCount != 3 || got.Status != StreakStatusActive {
		t.Fatalf("missed-day streak = %+v", got)
	}
}

func TestApplyDailyCompletionIsIdempotentForSameDate(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	current := &Streak{CurrentCount: 4, BestCount: 5, LastCompletedChallengeDate: &day, Status: StreakStatusActive, ProtectionState: ProtectionAvailable}

	got := ApplyDailyCompletion(current, day, now)
	if got.CurrentCount != 4 || got.BestCount != 5 || got.ProtectionState != ProtectionAvailable {
		t.Fatalf("same-day streak = %+v", got)
	}
}
