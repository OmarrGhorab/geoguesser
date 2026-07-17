package challenges

import (
	"fmt"
	"time"
)

const (
	MissionCadenceDaily  = "daily"
	MissionCadenceWeekly = "weekly"
)

type missionDefinition struct {
	Key, TitleKey, DescriptionKey, MissionType, Cadence, IconKey string
	TargetValue, RewardXP                                        int
}

var missionDefinitions = []missionDefinition{
	{"daily_completion", "Challenges.missions.dailyCompletion.title", "Challenges.missions.dailyCompletion.description", "daily_completion", MissionCadenceDaily, "daily-completion", 1, 75},
	{"score_threshold", "Challenges.missions.scoreThreshold.title", "Challenges.missions.scoreThreshold.description", "score_threshold", MissionCadenceDaily, "score-threshold", 15000, 100},
	{"round_accuracy", "Challenges.missions.roundAccuracy.title", "Challenges.missions.roundAccuracy.description", "round_accuracy", MissionCadenceDaily, "round-accuracy", 1, 125},
	{"shared_participation", "Challenges.missions.sharedParticipation.title", "Challenges.missions.sharedParticipation.description", "shared_participation", MissionCadenceWeekly, "shared-participation", 3, 250},
	{"streak_milestone", "Challenges.missions.streakMilestone.title", "Challenges.missions.streakMilestone.description", "streak_milestone", MissionCadenceWeekly, "streak-milestone", 3, 300},
}

func missionPeriod(now time.Time, cadence string, resetHourUTC int) (string, time.Time, time.Time) {
	now = now.UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), resetHourUTC, 0, 0, 0, time.UTC)
	if now.Before(dayStart) {
		dayStart = dayStart.AddDate(0, 0, -1)
	}
	if cadence == MissionCadenceDaily {
		return dayStart.Format("2006-01-02"), dayStart, dayStart.AddDate(0, 0, 1)
	}
	weekStart := dayStart.AddDate(0, 0, -((int(dayStart.Weekday()) + 6) % 7))
	isoYear, week := weekStart.ISOWeek()
	return fmt.Sprintf("%d-W%02d", isoYear, week), weekStart, weekStart.AddDate(0, 0, 7)
}

func DefaultMissions(now time.Time, resetHourUTC int) []Mission {
	missions := make([]Mission, 0, len(missionDefinitions))
	for _, definition := range missionDefinitions {
		periodKey, startsAt, endsAt := missionPeriod(now, definition.Cadence, resetHourUTC)
		missions = append(missions, Mission{
			Code: fmt.Sprintf("%s:%s:%s", definition.Cadence, periodKey, definition.Key), MissionKey: definition.Key,
			Cadence: definition.Cadence, PeriodKey: periodKey, IconKey: definition.IconKey,
			TitleKey: definition.TitleKey, DescriptionKey: definition.DescriptionKey, MissionType: definition.MissionType,
			TargetValue: definition.TargetValue, ActiveStartsAt: startsAt, ActiveEndsAt: &endsAt,
			RewardSnapshot: []byte(fmt.Sprintf(`{"xp":%d}`, definition.RewardXP)), RewardXP: definition.RewardXP, Status: "active",
		})
	}
	return missions
}

// DefaultMissionSummaries is used by challenge metadata before a mission read is requested.
func DefaultMissionSummaries(now time.Time) []MissionSummary {
	return DefaultMissionSummariesForReset(now, 0)
}

func DefaultMissionSummariesForReset(now time.Time, resetHourUTC int) []MissionSummary {
	missions := DefaultMissions(now, resetHourUTC)
	summaries := make([]MissionSummary, len(missions))
	for i, mission := range missions {
		summaries[i] = MissionSummary{Code: mission.Code, MissionKey: mission.MissionKey, TitleKey: mission.TitleKey, DescriptionKey: mission.DescriptionKey, MissionType: mission.MissionType, Cadence: mission.Cadence, PeriodKey: mission.PeriodKey, IconKey: mission.IconKey, RewardXP: mission.RewardXP, TargetValue: mission.TargetValue, Status: "not_started", ActiveEndsAt: mission.ActiveEndsAt}
	}
	return summaries
}
