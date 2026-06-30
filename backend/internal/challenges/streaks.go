package challenges

import "time"

func EmptyStreakSummary(guestLimited bool) StreakSummary {
	return StreakSummary{
		CurrentCount:    0,
		BestCount:       0,
		Status:          StreakStatusInactive,
		ProtectionState: ProtectionNone,
		GuestLimited:    guestLimited,
	}
}

func ApplyDailyCompletion(current *Streak, challengeDate time.Time, now time.Time) Streak {
	if current == nil {
		return Streak{CurrentCount: 1, BestCount: 1, LastCompletedChallengeDate: &challengeDate, Status: StreakStatusActive, ProtectionState: ProtectionNone, UpdatedAt: now}
	}
	next := *current
	if next.LastCompletedChallengeDate != nil {
		last := next.LastCompletedChallengeDate.UTC()
		switch {
		case sameDate(last, challengeDate):
			next.UpdatedAt = now
			return next
		case sameDate(last.AddDate(0, 0, 1), challengeDate):
			next.CurrentCount++
		default:
			next.CurrentCount = 1
			next.Status = StreakStatusBroken
		}
	} else {
		next.CurrentCount = 1
	}
	if next.CurrentCount > next.BestCount {
		next.BestCount = next.CurrentCount
	}
	next.Status = StreakStatusActive
	next.LastCompletedChallengeDate = &challengeDate
	next.UpdatedAt = now
	return next
}

func sameDate(a, b time.Time) bool {
	au := a.UTC()
	bu := b.UTC()
	return au.Year() == bu.Year() && au.YearDay() == bu.YearDay()
}
