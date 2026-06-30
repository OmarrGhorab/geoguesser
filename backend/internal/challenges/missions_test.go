package challenges

import (
	"testing"
	"time"
)

func TestDefaultMissionSummariesCoverRequiredChallengeTypes(t *testing.T) {
	missions := DefaultMissionSummaries(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC))
	seen := map[string]bool{}
	for _, mission := range missions {
		seen[mission.MissionType] = true
		if mission.TargetValue <= 0 {
			t.Fatalf("mission %s target = %d", mission.Code, mission.TargetValue)
		}
	}

	for _, required := range []string{"daily_completion", "shared_participation", "score_threshold", "leaderboard_milestone", "streak_milestone", "round_accuracy"} {
		if !seen[required] {
			t.Fatalf("missing mission type %s in %+v", required, missions)
		}
	}
}
