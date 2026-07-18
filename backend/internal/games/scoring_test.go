package games

import "testing"

func TestDistanceMetersKnownDistances(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		lat1 float64
		lng1 float64
		lat2 float64
		lng2 float64
		want int
	}{
		{name: "same point", lat1: 48.8584, lng1: 2.2945, lat2: 48.8584, lng2: 2.2945, want: 0},
		{name: "one degree longitude at equator", lat1: 0, lng1: 0, lat2: 0, lng2: 1, want: 111195},
		{name: "paris to cairo", lat1: 48.8584, lng1: 2.2945, lat2: 30.0444, lng2: 31.2357, want: 3214000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DistanceMeters(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
			tolerance := 5000
			if tc.want == 0 {
				tolerance = 0
			}
			if got < tc.want-tolerance || got > tc.want+tolerance {
				t.Fatalf("DistanceMeters() = %d, want around %d", got, tc.want)
			}
		})
	}
}

func TestScoreV1(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		distanceMeters int
		want           int
	}{
		{name: "zero distance", distanceMeters: 0, want: 5000},
		{name: "threshold distance", distanceMeters: 25, want: 5000},
		{name: "one thousand km", distanceMeters: 1_000_000, want: 2558},
		{name: "very far", distanceMeters: 20_000_000, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScoreV1(tc.distanceMeters); got != tc.want {
				t.Fatalf("ScoreV1(%d) = %d, want %d", tc.distanceMeters, got, tc.want)
			}
		})
	}
}

func TestScoreV1Bounds(t *testing.T) {
	t.Parallel()

	for _, distance := range []int{-1, 0, 1, 25, 26, 1000, 1_000_000, 20_000_000, 100_000_000} {
		score := ScoreV1(distance)
		if score < 0 || score > 5000 {
			t.Fatalf("ScoreV1(%d) = %d, want 0..5000", distance, score)
		}
	}
}

func TestScoringVersionConstant(t *testing.T) {
	t.Parallel()

	if ScoringVersionV1 != 1 {
		t.Fatalf("ScoringVersionV1 = %d, want 1", ScoringVersionV1)
	}
}

func TestSpeedBonusV1(t *testing.T) {
	t.Parallel()

	// Full remaining time: floor(4000 * 0.05 * 1) = 200.
	if got := SpeedBonusV1(4000, 60_000, 60_000); got != 200 {
		t.Fatalf("full remaining SpeedBonusV1 = %d, want 200", got)
	}
	// Half remaining: floor(4000 * 0.05 * 0.5) = 100.
	if got := SpeedBonusV1(4000, 30_000, 60_000); got != 100 {
		t.Fatalf("half remaining SpeedBonusV1 = %d, want 100", got)
	}
	// Cap at 250: floor(5000 * 0.05) = 250.
	if got := SpeedBonusV1(5000, 60_000, 60_000); got != 250 {
		t.Fatalf("cap SpeedBonusV1 = %d, want 250", got)
	}
	if got := SpeedBonusV1(5000, 0, 60_000); got != 0 {
		t.Fatalf("zero remaining = %d", got)
	}
	if got := SpeedBonusV1(0, 60_000, 60_000); got != 0 {
		t.Fatalf("zero accuracy = %d", got)
	}
}

func TestComposeGuessScoresCasualZeroBonus(t *testing.T) {
	t.Parallel()

	acc, bonus, total := ComposeGuessScores(GameModeCasualDuo, 4000, 60_000, 60_000)
	if acc != 4000 || bonus != 0 || total != 4000 {
		t.Fatalf("casual compose = %d/%d/%d", acc, bonus, total)
	}
	acc, bonus, total = ComposeGuessScores(GameModeRankedSolo, 4000, 60_000, 60_000)
	if acc != 4000 || bonus != 200 || total != 4200 {
		t.Fatalf("ranked compose = %d/%d/%d", acc, bonus, total)
	}
	acc, bonus, total = ComposeGuessScores(GameModePrivateRoom, 4000, 60_000, 60_000)
	if bonus != 0 || total != acc {
		t.Fatalf("private room must not grant speed bonus: %d/%d/%d", acc, bonus, total)
	}
}
