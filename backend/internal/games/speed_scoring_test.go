package games

import (
	"testing"
	"time"
)

func TestSpeedBonusV1Table(t *testing.T) {
	t.Parallel()

	const duration60s = int64(60_000)

	cases := []struct {
		name          string
		accuracyScore int
		remainingMs   int64
		durationMs    int64
		want          int
	}{
		// Full remaining-time fraction: floor(4000 * 0.05 * 1) = 200.
		{name: "full remaining 4000 accuracy", accuracyScore: 4000, remainingMs: duration60s, durationMs: duration60s, want: 200},
		// Half remaining: floor(4000 * 0.05 * 0.5) = 100.
		{name: "half remaining fraction", accuracyScore: 4000, remainingMs: 30_000, durationMs: duration60s, want: 100},
		// Quarter remaining: floor(4000 * 0.05 * 0.25) = 50.
		{name: "quarter remaining fraction", accuracyScore: 4000, remainingMs: 15_000, durationMs: duration60s, want: 50},
		// Cap at 250: floor(5000 * 0.05 * 1) = 250.
		{name: "250 point cap at perfect accuracy full time", accuracyScore: 5000, remainingMs: duration60s, durationMs: duration60s, want: 250},
		// Cap would exceed without min: floor(5000 * 0.05 * 1) = 250 (exactly cap).
		{name: "cap exact not exceeded", accuracyScore: 5000, remainingMs: duration60s, durationMs: duration60s, want: 250},
		// Zero accuracy → zero bonus regardless of remaining.
		{name: "zero accuracy full remaining", accuracyScore: 0, remainingMs: duration60s, durationMs: duration60s, want: 0},
		{name: "negative accuracy treated as zero path", accuracyScore: -100, remainingMs: duration60s, durationMs: duration60s, want: 0},
		// Deadline clamp: remaining 0 → bonus 0.
		{name: "deadline clamp remaining zero", accuracyScore: 5000, remainingMs: 0, durationMs: duration60s, want: 0},
		// Deadline clamp: remaining negative is treated as 0 by helper.
		{name: "deadline clamp remaining negative", accuracyScore: 5000, remainingMs: -5_000, durationMs: duration60s, want: 0},
		// Deadline clamp: remaining > duration clamps to full duration.
		{name: "remaining above duration clamps to full", accuracyScore: 4000, remainingMs: 120_000, durationMs: duration60s, want: 200},
		// Non-positive duration → 0.
		{name: "zero duration", accuracyScore: 4000, remainingMs: 1_000, durationMs: 0, want: 0},
		{name: "negative duration", accuracyScore: 4000, remainingMs: 1_000, durationMs: -1, want: 0},
		// Rounding: floor toward zero (integer division).
		// floor(4020 * 0.05 * 25000/60000) = floor(201 * 25000/60000) wait:
		// int: 4020 * 25000 * 5 / (60000 * 100) = 502500000 / 6000000 = 83.
		{name: "floor rounding mid window", accuracyScore: 4020, remainingMs: 25_000, durationMs: duration60s, want: 83},
		// floor(1 * 0.05 * 1) = floor(0.05) = 0 via integer math.
		{name: "tiny accuracy floors to zero", accuracyScore: 1, remainingMs: duration60s, durationMs: duration60s, want: 0},
		// floor(19 * 0.05) = floor(0.95) = 0.
		{name: "accuracy 19 floors to zero", accuracyScore: 19, remainingMs: duration60s, durationMs: duration60s, want: 0},
		// floor(20 * 0.05) = 1.
		{name: "accuracy 20 yields one point at full time", accuracyScore: 20, remainingMs: duration60s, durationMs: duration60s, want: 1},
		// One millisecond remaining of 60s: floor(5000 * 0.05 * 1/60000) = floor(250/60000) = 0.
		{name: "one ms remaining floors to zero", accuracyScore: 5000, remainingMs: 1, durationMs: duration60s, want: 0},
		// Near-deadline with high accuracy that still yields a point:
		// floor(5000 * 0.05 * 240/60000) = floor(250 * 240/60000) = floor(1) = 1.
		{name: "near deadline still awards one", accuracyScore: 5000, remainingMs: 240, durationMs: duration60s, want: 1},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SpeedBonusV1(tc.accuracyScore, tc.remainingMs, tc.durationMs)
			if got != tc.want {
				t.Fatalf("SpeedBonusV1(%d, %d, %d) = %d, want %d",
					tc.accuracyScore, tc.remainingMs, tc.durationMs, got, tc.want)
			}
			if got < 0 || got > maxSpeedBonus {
				t.Fatalf("bonus %d out of bounds 0..%d", got, maxSpeedBonus)
			}
		})
	}
}

func TestRoundTimerWindowMsTable(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	ends := starts.Add(60 * time.Second)

	cases := []struct {
		name          string
		startsAt      *time.Time
		endsAt        *time.Time
		submittedAt   time.Time
		wantRemaining int64
		wantDuration  int64
	}{
		{
			name:     "full window at start",
			startsAt: &starts, endsAt: &ends,
			submittedAt:   starts,
			wantRemaining: 60_000, wantDuration: 60_000,
		},
		{
			name:     "half remaining",
			startsAt: &starts, endsAt: &ends,
			submittedAt:   starts.Add(30 * time.Second),
			wantRemaining: 30_000, wantDuration: 60_000,
		},
		{
			name:     "at deadline remaining zero",
			startsAt: &starts, endsAt: &ends,
			submittedAt:   ends,
			wantRemaining: 0, wantDuration: 60_000,
		},
		{
			name:     "after deadline clamps remaining to zero",
			startsAt: &starts, endsAt: &ends,
			submittedAt:   ends.Add(5 * time.Second),
			wantRemaining: 0, wantDuration: 60_000,
		},
		{
			name:     "before start clamps remaining to duration",
			startsAt: &starts, endsAt: &ends,
			submittedAt:   starts.Add(-10 * time.Second),
			wantRemaining: 60_000, wantDuration: 60_000,
		},
		{
			name:     "nil starts yields zero window",
			startsAt: nil, endsAt: &ends,
			submittedAt:   starts,
			wantRemaining: 0, wantDuration: 0,
		},
		{
			name:     "nil ends yields zero window",
			startsAt: &starts, endsAt: nil,
			submittedAt:   starts,
			wantRemaining: 0, wantDuration: 0,
		},
		{
			name:     "sub-millisecond duration floors to one",
			startsAt: ptrTime(starts), endsAt: ptrTime(starts.Add(time.Nanosecond)),
			submittedAt: starts,
			// Sub() of 1ns is 0ms → max(1, 0) = 1; remaining ends-submitted = 0ms.
			wantRemaining: 0, wantDuration: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rem, dur := RoundTimerWindowMs(tc.startsAt, tc.endsAt, tc.submittedAt)
			if rem != tc.wantRemaining || dur != tc.wantDuration {
				t.Fatalf("RoundTimerWindowMs = remaining %d duration %d, want %d / %d",
					rem, dur, tc.wantRemaining, tc.wantDuration)
			}
			if rem < 0 || (dur > 0 && rem > dur) {
				t.Fatalf("remaining %d not clamped to [0, %d]", rem, dur)
			}
		})
	}
}

func TestComposeGuessScoresRankedAndCasual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     string
		accuracy int
		rem, dur int64
		wantAcc  int
		wantBon  int
		wantTot  int
	}{
		{
			name: "ranked full bonus",
			mode: GameModeRankedDuo, accuracy: 4000, rem: 60_000, dur: 60_000,
			wantAcc: 4000, wantBon: 200, wantTot: 4200,
		},
		{
			name: "ranked solo alias",
			mode: GameModeRankedSolo, accuracy: 5000, rem: 60_000, dur: 60_000,
			wantAcc: 5000, wantBon: 250, wantTot: 5250,
		},
		{
			name: "ranked squad half remaining",
			mode: GameModeRankedSquad, accuracy: 4000, rem: 30_000, dur: 60_000,
			wantAcc: 4000, wantBon: 100, wantTot: 4100,
		},
		{
			name: "legacy ranked mode",
			mode: GameModeRanked, accuracy: 4000, rem: 60_000, dur: 60_000,
			wantAcc: 4000, wantBon: 200, wantTot: 4200,
		},
		{
			name: "casual duo zero bonus",
			mode: GameModeCasualDuo, accuracy: 4000, rem: 60_000, dur: 60_000,
			wantAcc: 4000, wantBon: 0, wantTot: 4000,
		},
		{
			name: "casual squad zero bonus",
			mode: GameModeCasualSquad, accuracy: 4020, rem: 45_000, dur: 60_000,
			wantAcc: 4020, wantBon: 0, wantTot: 4020,
		},
		{
			name: "private room zero bonus",
			mode: GameModePrivateRoom, accuracy: 4000, rem: 60_000, dur: 60_000,
			wantAcc: 4000, wantBon: 0, wantTot: 4000,
		},
		{
			name: "solo zero bonus",
			mode: GameModeSolo, accuracy: 4000, rem: 60_000, dur: 60_000,
			wantAcc: 4000, wantBon: 0, wantTot: 4000,
		},
		{
			name: "ranked zero accuracy zero bonus",
			mode: GameModeRankedDuo, accuracy: 0, rem: 60_000, dur: 60_000,
			wantAcc: 0, wantBon: 0, wantTot: 0,
		},
		{
			name: "accuracy clamped to max",
			mode: GameModeRankedDuo, accuracy: 9000, rem: 60_000, dur: 60_000,
			wantAcc: 5000, wantBon: 250, wantTot: 5250,
		},
		{
			name: "negative accuracy clamped",
			mode: GameModeRankedDuo, accuracy: -50, rem: 60_000, dur: 60_000,
			wantAcc: 0, wantBon: 0, wantTot: 0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			acc, bon, tot := ComposeGuessScores(tc.mode, tc.accuracy, tc.rem, tc.dur)
			if acc != tc.wantAcc || bon != tc.wantBon || tot != tc.wantTot {
				t.Fatalf("ComposeGuessScores = %d/%d/%d, want %d/%d/%d",
					acc, bon, tot, tc.wantAcc, tc.wantBon, tc.wantTot)
			}
			if tot != acc+bon {
				t.Fatalf("invariant total != accuracy+bonus: %d != %d+%d", tot, acc, bon)
			}
		})
	}
}

func TestScoreWithSpeedBonusUsesServerWindow(t *testing.T) {
	t.Parallel()

	starts := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	ends := starts.Add(60 * time.Second)
	// Submit at t+15s → remaining 45s → floor(4000 * 0.05 * 0.75) = 150.
	submitted := starts.Add(15 * time.Second)
	acc, bon, tot := ScoreWithSpeedBonus(GameModeRankedSolo, 4000, &starts, &ends, submitted)
	if acc != 4000 || bon != 150 || tot != 4150 {
		t.Fatalf("ScoreWithSpeedBonus = %d/%d/%d, want 4000/150/4150", acc, bon, tot)
	}
	// Casual ignores the timer window.
	acc, bon, tot = ScoreWithSpeedBonus(GameModeCasualSolo, 4000, &starts, &ends, submitted)
	if acc != 4000 || bon != 0 || tot != 4000 {
		t.Fatalf("casual ScoreWithSpeedBonus = %d/%d/%d", acc, bon, tot)
	}
}

func TestApplyRankedTimerDefaults(t *testing.T) {
	t.Parallel()

	if got := ApplyRankedTimerDefaults(GameModeCasualDuo, IntPtr(60)); got != nil {
		t.Fatalf("casual must force nil timer, got %v", *got)
	}
	if got := ApplyRankedTimerDefaults(GameModeRankedDuo, nil); got == nil || *got != RankedRoundTimerSeconds {
		t.Fatalf("ranked nil timer default = %v, want %d", got, RankedRoundTimerSeconds)
	}
	if got := ApplyRankedTimerDefaults(GameModeRankedSquad, IntPtr(0)); got == nil || *got != RankedRoundTimerSeconds {
		t.Fatalf("ranked zero timer default = %v, want %d", got, RankedRoundTimerSeconds)
	}
	if got := ApplyRankedTimerDefaults(GameModeRankedSolo, IntPtr(45)); got == nil || *got != 45 {
		t.Fatalf("explicit ranked timer should pass through, got %v", got)
	}
	if got := ApplyRankedTimerDefaults(GameModePrivateRoom, IntPtr(90)); got == nil || *got != 90 {
		t.Fatalf("private room timer pass-through = %v", got)
	}
}

func TestRankedRoundTimerSecondsConstant(t *testing.T) {
	t.Parallel()
	if RankedRoundTimerSeconds != 60 {
		t.Fatalf("RankedRoundTimerSeconds = %d, want 60", RankedRoundTimerSeconds)
	}
	if MaxCombinedRoundScore != 5250 {
		t.Fatalf("MaxCombinedRoundScore = %d, want 5250", MaxCombinedRoundScore)
	}
	if MaxSpeedBonusPoints() != 250 {
		t.Fatalf("MaxSpeedBonusPoints = %d, want 250", MaxSpeedBonusPoints())
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
