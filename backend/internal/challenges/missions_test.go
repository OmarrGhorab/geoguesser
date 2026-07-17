package challenges

import (
	"testing"
	"time"
)

func TestDefaultMissionsHaveDailyAndWeeklyLevelRewards(t *testing.T) {
	missions := DefaultMissions(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC), 0)
	seen := map[string]bool{}
	for _, mission := range missions {
		seen[mission.MissionType] = true
		if mission.TargetValue <= 0 || mission.RewardXP <= 0 || mission.IconKey == "" {
			t.Fatalf("mission %s target = %d", mission.Code, mission.TargetValue)
		}
	}

	for _, required := range []string{"daily_completion", "shared_participation", "score_threshold", "streak_milestone", "round_accuracy"} {
		if !seen[required] {
			t.Fatalf("missing mission type %s in %+v", required, missions)
		}
	}
}

func TestMissionPeriodUsesConfiguredResetAndISOWeek(t *testing.T) {
	key, starts, ends := missionPeriod(time.Date(2026, 6, 27, 1, 0, 0, 0, time.UTC), MissionCadenceDaily, 4)
	if key != "2026-06-26" || starts.Hour() != 4 || !ends.Equal(starts.AddDate(0, 0, 1)) {
		t.Fatalf("daily period = %q %s %s", key, starts, ends)
	}
	key, _, ends = missionPeriod(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC), MissionCadenceWeekly, 0)
	if key != "2026-W26" || ends.Weekday() != time.Monday {
		t.Fatalf("weekly period = %q ends %s", key, ends)
	}
}

func TestMissionPeriodUsesISOWeekYearAtCalendarBoundary(t *testing.T) {
	t.Parallel()

	key, starts, ends := missionPeriod(
		time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC),
		MissionCadenceWeekly,
		0,
	)
	if key != "2026-W53" {
		t.Fatalf("weekly period = %q, want 2026-W53", key)
	}
	if starts.Weekday() != time.Monday || !ends.Equal(starts.AddDate(0, 0, 7)) {
		t.Fatalf("weekly bounds = %s..%s", starts, ends)
	}
}
