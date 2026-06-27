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
