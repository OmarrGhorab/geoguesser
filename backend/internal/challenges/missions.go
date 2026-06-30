package challenges

import "time"

func DefaultMissionSummaries(now time.Time) []MissionSummary {
	end := now.UTC().Add(24 * time.Hour)
	return []MissionSummary{
		{Code: "daily_completion", TitleKey: "Challenges.missions.dailyCompletion.title", DescriptionKey: "Challenges.missions.dailyCompletion.description", MissionType: "daily_completion", CurrentValue: 0, TargetValue: 1, Status: "not_started", ActiveEndsAt: &end},
		{Code: "shared_participation", TitleKey: "Challenges.missions.sharedParticipation.title", DescriptionKey: "Challenges.missions.sharedParticipation.description", MissionType: "shared_participation", CurrentValue: 0, TargetValue: 1, Status: "not_started", ActiveEndsAt: &end},
		{Code: "score_threshold", TitleKey: "Challenges.missions.scoreThreshold.title", DescriptionKey: "Challenges.missions.scoreThreshold.description", MissionType: "score_threshold", CurrentValue: 0, TargetValue: 15000, Status: "not_started", ActiveEndsAt: &end},
		{Code: "leaderboard_milestone", TitleKey: "Challenges.missions.leaderboardMilestone.title", DescriptionKey: "Challenges.missions.leaderboardMilestone.description", MissionType: "leaderboard_milestone", CurrentValue: 0, TargetValue: 10, Status: "not_started", ActiveEndsAt: &end},
		{Code: "streak_milestone", TitleKey: "Challenges.missions.streakMilestone.title", DescriptionKey: "Challenges.missions.streakMilestone.description", MissionType: "streak_milestone", CurrentValue: 0, TargetValue: 3, Status: "not_started", ActiveEndsAt: &end},
		{Code: "round_accuracy", TitleKey: "Challenges.missions.roundAccuracy.title", DescriptionKey: "Challenges.missions.roundAccuracy.description", MissionType: "round_accuracy", CurrentValue: 0, TargetValue: 1, Status: "not_started", ActiveEndsAt: &end},
	}
}
