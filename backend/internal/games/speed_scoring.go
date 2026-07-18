package games

import "time"

const (
	// RankedRoundTimerSeconds is the authoritative Ranked round duration.
	RankedRoundTimerSeconds = 60
	// maxSpeedBonus is the hard cap on accuracy-weighted Ranked speed bonus points.
	maxSpeedBonus = 250
	// MaxCombinedRoundScore is accuracy + max ranked speed bonus.
	MaxCombinedRoundScore = maxRoundScore + maxSpeedBonus
)

// MaxSpeedBonusPoints is the maximum ranked speed bonus for a single round.
func MaxSpeedBonusPoints() int { return maxSpeedBonus }

// SpeedBonusV1 computes the ranked speed bonus from accuracy and remaining time fraction.
//
// Spec formula (authoritative):
//
//	duration_ms  = max(1, ends_at - starts_at)
//	remaining_ms = clamp(ends_at - submitted_at, 0, duration_ms)
//	speed_bonus  = min(250, floor(accuracy_score * 0.05 * remaining_ms / duration_ms))
//
// Callers should pass remaining/duration already produced by RoundTimerWindowMs.
// Non-positive duration, non-positive remaining, or non-positive accuracy yields 0.
// remaining is clamped to [0, duration] defensively.
func SpeedBonusV1(accuracyScore int, remainingMs, durationMs int64) int {
	if accuracyScore <= 0 || durationMs <= 0 || remainingMs <= 0 {
		return 0
	}
	if remainingMs > durationMs {
		remainingMs = durationMs
	}
	// floor(accuracy * 0.05 * remaining / duration) via integer math:
	// accuracy * remaining * 5 / (duration * 100)
	bonus := int(int64(accuracyScore) * remainingMs * 5 / (durationMs * 100))
	if bonus < 0 {
		return 0
	}
	if bonus > maxSpeedBonus {
		return maxSpeedBonus
	}
	return bonus
}

// ComposeGuessScores sets accuracy, bonus, and total score for a guess.
// Casual and non-ranked modes always receive zero speed bonus.
// Invariant: total == accuracy + bonus.
func ComposeGuessScores(mode string, accuracyScore int, remainingMs, durationMs int64) (accuracy, bonus, total int) {
	accuracy = accuracyScore
	if accuracy < 0 {
		accuracy = 0
	}
	if accuracy > maxRoundScore {
		accuracy = maxRoundScore
	}
	bonus = 0
	if IsRankedMode(mode) {
		bonus = SpeedBonusV1(accuracy, remainingMs, durationMs)
	}
	total = accuracy + bonus
	return accuracy, bonus, total
}

// RoundTimerWindowMs returns remaining and duration milliseconds for Ranked speed bonus.
//
//	duration_ms  = max(1, ends_at - starts_at)
//	remaining_ms = clamp(ends_at - submitted_at, 0, duration_ms)
//
// When starts/ends are missing, both values are 0 (no timer window → no bonus).
func RoundTimerWindowMs(startsAt, endsAt *time.Time, submittedAt time.Time) (remainingMs, durationMs int64) {
	if startsAt == nil || endsAt == nil {
		return 0, 0
	}
	durationMs = endsAt.Sub(*startsAt).Milliseconds()
	if durationMs < 1 {
		durationMs = 1
	}
	remainingMs = endsAt.Sub(submittedAt).Milliseconds()
	if remainingMs < 0 {
		remainingMs = 0
	}
	if remainingMs > durationMs {
		remainingMs = durationMs
	}
	return remainingMs, durationMs
}

// ScoreWithSpeedBonus is a convenience that derives the timer window then composes scores.
// submittedAt is server time at acceptance; starts/ends come from the locked round row.
func ScoreWithSpeedBonus(mode string, accuracyScore int, startsAt, endsAt *time.Time, submittedAt time.Time) (accuracy, bonus, total int) {
	remainingMs, durationMs := RoundTimerWindowMs(startsAt, endsAt, submittedAt)
	return ComposeGuessScores(mode, accuracyScore, remainingMs, durationMs)
}

// DefaultRankedTimerSeconds returns a pointer to the 60-second Ranked timer.
func DefaultRankedTimerSeconds() *int {
	v := RankedRoundTimerSeconds
	return &v
}

// ApplyRankedTimerDefaults ensures Ranked games use the 60-second timer when unset.
// Casual always returns nil (no deadline). Non-ranked non-casual passes timer through.
func ApplyRankedTimerDefaults(mode string, timerSeconds *int) *int {
	if IsCasualMode(mode) {
		return nil
	}
	if IsRankedMode(mode) && (timerSeconds == nil || *timerSeconds <= 0) {
		return DefaultRankedTimerSeconds()
	}
	return timerSeconds
}
