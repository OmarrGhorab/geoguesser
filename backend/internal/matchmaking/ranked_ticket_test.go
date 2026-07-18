package matchmaking_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/matchmaking"
)

func TestRatingWindowHalfWidth_InitialExpandCap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		waited time.Duration
		want   int
	}{
		{"initial", 0, 100},
		{"before first step", 29 * time.Second, 100},
		{"first expansion", 30 * time.Second, 150},
		{"second expansion", 60 * time.Second, 200},
		{"third expansion", 90 * time.Second, 250},
		{"at cap boundary", 180 * time.Second, 400}, // 100 + 50*6 = 400
		{"beyond cap", 10 * time.Minute, 400},
		{"negative wait treated as zero", -time.Second, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := matchmaking.RatingWindowHalfWidth(tc.waited)
			if got != tc.want {
				t.Fatalf("RatingWindowHalfWidth(%v) = %d, want %d", tc.waited, got, tc.want)
			}
		})
	}
}

func TestRatingWindowForAverage_FloorsAtZero(t *testing.T) {
	t.Parallel()
	win := matchmaking.RatingWindowForAverage(50, 0)
	if win.Minimum != 0 {
		t.Fatalf("minimum = %d, want 0", win.Minimum)
	}
	if win.Maximum != 150 {
		t.Fatalf("maximum = %d, want 150", win.Maximum)
	}

	win = matchmaking.RatingWindowForAverage(1000, 0)
	if win.Minimum != 900 || win.Maximum != 1100 {
		t.Fatalf("window = %+v, want 900–1100", win)
	}
}

func TestTeamAverageRating_UsesIntegerMean(t *testing.T) {
	t.Parallel()
	if got := matchmaking.TeamAverageRating(nil); got != 0 {
		t.Fatalf("empty = %d", got)
	}
	if got := matchmaking.TeamAverageRating([]int{800, 1000}); got != 900 {
		t.Fatalf("avg = %d, want 900", got)
	}
	// Placement hidden ratings participate in the mean.
	if got := matchmaking.TeamAverageRating([]int{800, 800, 1000, 1200}); got != 950 {
		t.Fatalf("placement-inclusive avg = %d, want 950", got)
	}
}

func TestTeamAveragesCompatible_ExpandingWindows(t *testing.T) {
	t.Parallel()

	// Fresh tickets: only ±100.
	if !matchmaking.TeamAveragesCompatible(1000, 1099, 0, 0) {
		t.Fatal("expected 99-delta compatible within ±100")
	}
	if matchmaking.TeamAveragesCompatible(1000, 1101, 0, 0) {
		t.Fatal("expected 101-delta incompatible at initial window")
	}

	// One ticket has expanded after 30s to ±150; max of windows is used.
	if !matchmaking.TeamAveragesCompatible(1000, 1140, 30*time.Second, 0) {
		t.Fatal("long-waiting ticket should accept 140-delta after first expansion")
	}

	// Cap at ±400.
	if !matchmaking.TeamAveragesCompatible(1000, 1400, 10*time.Minute, 10*time.Minute) {
		t.Fatal("400-delta should match at cap")
	}
	if matchmaking.TeamAveragesCompatible(1000, 1401, 10*time.Minute, 10*time.Minute) {
		t.Fatal("401-delta must not match even at cap")
	}
}

func TestNamedRankTier_BandsAndPlacementHidden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rating int
		want   int
	}{
		{0, 0},    // Scout
		{299, 0},  // Scout
		{300, 1},  // Pathfinder
		{800, 2},  // Trailblazer (default placement hidden rating)
		{900, 3},  // Navigator
		{1500, 5}, // Explorer
		{2000, 6}, // Geo Master
		{5000, 6}, // still Geo Master
	}
	for _, tc := range cases {
		if got := matchmaking.NamedRankTier(tc.rating); got != tc.want {
			t.Fatalf("NamedRankTier(%d) = %d, want %d", tc.rating, got, tc.want)
		}
	}

	// Placement: incomplete placements always derive from hidden rating.
	got := matchmaking.EffectiveNamedRankTier(800, 0, 2, "")
	if got != 2 {
		t.Fatalf("placement effective tier = %d, want 2 (Trailblazer from hidden 800)", got)
	}
}

func TestPartyNamedRankSpread_TwoNamedRanksMax(t *testing.T) {
	t.Parallel()

	if !matchmaking.PartyNamedRankSpreadOK([]int{2, 2}) {
		t.Fatal("same tier must be ok")
	}
	if !matchmaking.PartyNamedRankSpreadOK([]int{2, 3, 4}) {
		t.Fatal("spread of 2 (Trailblazer–Cartographer) must be ok")
	}
	if matchmaking.PartyNamedRankSpreadOK([]int{2, 5}) {
		t.Fatal("spread of 3 must be rejected")
	}
	if matchmaking.PartyNamedRankSpreadOK([]int{0, 1, 2, 3}) {
		t.Fatal("Scout–Navigator spread of 3 must be rejected")
	}
}

func TestPartySpreadFromStandings_PlacementUsesHiddenRating(t *testing.T) {
	t.Parallel()

	// Duo: one placed Navigator (tier 3), one placement at hidden 800 (Trailblazer tier 2).
	// Spread = 1 → ok.
	season := uuid.New()
	standings := []matchmaking.CompetitiveStandingSnapshot{
		{UserID: uuid.New(), SeasonID: season, Rating: 950, PlacementsCompleted: 5, RankCode: "navigator_3", NamedRankTier: 3},
		{UserID: uuid.New(), SeasonID: season, Rating: 800, PlacementsCompleted: 1, RankCode: "", NamedRankTier: 0},
	}
	tiers, ok := matchmaking.PartySpreadFromStandings(standings)
	if !ok {
		t.Fatalf("expected ok spread, tiers=%v", tiers)
	}
	if len(tiers) != 2 || tiers[0] != 3 || tiers[1] != 2 {
		t.Fatalf("tiers = %v, want [3 2]", tiers)
	}

	// Scout (0) with Explorer-level placement hidden 1600 (tier 5) → spread 5 → reject.
	standings = []matchmaking.CompetitiveStandingSnapshot{
		{UserID: uuid.New(), SeasonID: season, Rating: 50, PlacementsCompleted: 5, RankCode: "scout_3", NamedRankTier: 0},
		{UserID: uuid.New(), SeasonID: season, Rating: 1600, PlacementsCompleted: 0, RankCode: "", NamedRankTier: 0},
	}
	_, ok = matchmaking.PartySpreadFromStandings(standings)
	if ok {
		t.Fatal("expected party spread rejection for Scout vs Explorer-hidden")
	}
}

func TestTeamAverageComparison_UsesPlacementHiddenRatings(t *testing.T) {
	t.Parallel()
	// Ticket A: two placement players at 800 → avg 800.
	// Ticket B: 900 and 700 → avg 800. Compatible at initial window.
	avgA := matchmaking.TeamAverageRating([]int{800, 800})
	avgB := matchmaking.TeamAverageRating([]int{900, 700})
	if avgA != 800 || avgB != 800 {
		t.Fatalf("avgs = %d/%d, want 800/800", avgA, avgB)
	}
	if !matchmaking.TeamAveragesCompatible(avgA, avgB, 0, 0) {
		t.Fatal("equal averages must match")
	}

	// Ticket C: avg 1000 vs A 800 → delta 200 requires expansion.
	avgC := matchmaking.TeamAverageRating([]int{1000, 1000})
	if matchmaking.TeamAveragesCompatible(avgA, avgC, 0, 0) {
		t.Fatal("200-delta must fail at ±100")
	}
	if !matchmaking.TeamAveragesCompatible(avgA, avgC, 60*time.Second, 0) {
		t.Fatal("200-delta should pass once half-width reaches 200")
	}
}
