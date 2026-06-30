package challenges

import (
	"testing"
	"time"
)

func TestDailyWindowUsesCanonicalResetBoundary(t *testing.T) {
	now := time.Date(2026, 6, 27, 3, 30, 0, 0, time.UTC)
	date, start, end := DailyWindow(now, 4)

	if got, want := date.Format("2006-01-02"), "2026-06-26"; got != want {
		t.Fatalf("date = %s, want %s", got, want)
	}
	if got, want := start.Format(time.RFC3339), "2026-06-26T04:00:00Z"; got != want {
		t.Fatalf("start = %s, want %s", got, want)
	}
	if got, want := end.Format(time.RFC3339), "2026-06-27T04:00:00Z"; got != want {
		t.Fatalf("end = %s, want %s", got, want)
	}
}

func TestDailySeedIsStableAndDateBound(t *testing.T) {
	day := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	if DailySeed(day) != DailySeed(day.Add(12*time.Hour)) {
		t.Fatal("same challenge date should produce the same seed")
	}
	if DailySeed(day) == DailySeed(day.AddDate(0, 0, 1)) {
		t.Fatal("different challenge dates should produce different seeds")
	}
}

func TestSharedCodeShape(t *testing.T) {
	code, err := SharedCode()
	if err != nil {
		t.Fatalf("SharedCode() error = %v", err)
	}
	if len(code) != 10 {
		t.Fatalf("code length = %d, want 10", len(code))
	}
}
