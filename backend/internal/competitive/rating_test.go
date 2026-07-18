package competitive_test

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestExpectedProbability_EqualRatings(t *testing.T) {
	t.Parallel()
	p := competitive.ExpectedProbability(800, 800)
	if math.Abs(p-0.5) > 1e-9 {
		t.Fatalf("equal expected = %v, want 0.5", p)
	}
	if bps := competitive.ExpectedBPS(800, 800); bps != 5000 {
		t.Fatalf("equal bps = %d, want 5000", bps)
	}
}

func TestExpectedProbability_favoriteAndUnderdog(t *testing.T) {
	t.Parallel()
	// Higher rated player has higher expected score.
	pHigh := competitive.ExpectedProbability(1200, 800)
	pLow := competitive.ExpectedProbability(800, 1200)
	if pHigh <= 0.5 || pLow >= 0.5 {
		t.Fatalf("pHigh=%v pLow=%v", pHigh, pLow)
	}
	if math.Abs(pHigh+pLow-1.0) > 1e-9 {
		t.Fatalf("expected scores should sum to 1: %v + %v", pHigh, pLow)
	}
}

func TestBaseDelta_equalRatedWinLossDraw(t *testing.T) {
	t.Parallel()
	k := 32
	// Equal rated: expected=0.5 → win +16, loss -16, draw 0.
	win := competitive.BaseDelta(k, competitive.ActualWin, 0.5)
	loss := competitive.BaseDelta(k, competitive.ActualLoss, 0.5)
	draw := competitive.BaseDelta(k, competitive.ActualDraw, 0.5)
	if win != 16 {
		t.Fatalf("equal win delta = %d, want +16", win)
	}
	if loss != -16 {
		t.Fatalf("equal loss delta = %d, want -16", loss)
	}
	if draw != 0 {
		t.Fatalf("equal draw delta = %d, want 0", draw)
	}

	// Helpers match.
	if got := competitive.BaseDeltaForAverages(k, 1000, 1000, competitive.OutcomeWin); got != 16 {
		t.Fatalf("BaseDeltaForAverages win = %d, want 16", got)
	}
	if got := competitive.BaseDeltaForAverages(k, 1000, 1000, competitive.OutcomeLoss); got != -16 {
		t.Fatalf("BaseDeltaForAverages loss = %d, want -16", got)
	}
	if got := competitive.BaseDeltaForAverages(k, 1000, 1000, competitive.OutcomeDraw); got != 0 {
		t.Fatalf("BaseDeltaForAverages draw = %d, want 0", got)
	}
}

func TestBaseDelta_teamEqualBaseDelta(t *testing.T) {
	t.Parallel()
	// Team averages drive one shared base delta for all teammates.
	teamAvg, oppAvg := 900, 1100
	delta := competitive.BaseDeltaForAverages(32, teamAvg, oppAvg, competitive.OutcomeWin)
	// Every non-abandoning teammate must receive the same value.
	for i := 0; i < 4; i++ {
		if got := competitive.BaseDeltaForAverages(32, teamAvg, oppAvg, competitive.OutcomeWin); got != delta {
			t.Fatalf("teammate %d delta = %d, want shared %d", i, got, delta)
		}
	}
	// Opposite team loss is not necessarily the negation when averages differ,
	// but teammates on the same side share the loss delta.
	lossDelta := competitive.BaseDeltaForAverages(32, teamAvg, oppAvg, competitive.OutcomeLoss)
	for i := 0; i < 4; i++ {
		if got := competitive.BaseDeltaForAverages(32, teamAvg, oppAvg, competitive.OutcomeLoss); got != lossDelta {
			t.Fatalf("losing teammate %d delta = %d, want shared %d", i, got, lossDelta)
		}
	}
}

func TestApplyRating_floorZero(t *testing.T) {
	t.Parallel()
	newR, total := competitive.ApplyRating(5, -16, 0)
	if newR != 0 {
		t.Fatalf("floor new = %d, want 0", newR)
	}
	if total != -5 {
		t.Fatalf("floor total_delta = %d, want -5", total)
	}

	newR, total = competitive.ApplyRating(10, -16, -15)
	if newR != 0 {
		t.Fatalf("floor with abandon new = %d, want 0", newR)
	}
	if total != -10 {
		t.Fatalf("floor with abandon total = %d, want -10", total)
	}

	newR, total = competitive.ApplyRating(800, 16, 0)
	if newR != 816 || total != 16 {
		t.Fatalf("normal apply = %d/%d, want 816/16", newR, total)
	}
}

func TestAbandonPenalty_minusFifteen(t *testing.T) {
	t.Parallel()
	if got := competitive.AbandonPenaltyValue(false, 15); got != 0 {
		t.Fatalf("non-abandoner penalty = %d, want 0", got)
	}
	if got := competitive.AbandonPenaltyValue(true, 15); got != -15 {
		t.Fatalf("abandoner penalty = %d, want -15", got)
	}
	// Config stores positive magnitude.
	if got := competitive.AbandonPenaltyValue(true, competitive.DefaultAbandonPenalty); got != -15 {
		t.Fatalf("default abandon = %d, want -15", got)
	}

	// Quitter gets loss base + abandon penalty.
	base := competitive.BaseDeltaForAverages(32, 800, 800, competitive.OutcomeLoss)
	if base != -16 {
		t.Fatalf("loss base = %d, want -16", base)
	}
	pen := competitive.AbandonPenaltyValue(true, 15)
	newR, total := competitive.ApplyRating(800, base, pen)
	if newR != 800-16-15 {
		t.Fatalf("quitter new = %d, want 769", newR)
	}
	if total != -31 {
		t.Fatalf("quitter total = %d, want -31", total)
	}
}

func TestPlacements_hiddenUntilFive(t *testing.T) {
	t.Parallel()
	if competitive.PlacementsRequired != 5 {
		t.Fatalf("placements required = %d, want 5", competitive.PlacementsRequired)
	}
	for i := 0; i < 5; i++ {
		code := competitive.OptionalRankCode(800, i)
		if code != nil {
			t.Fatalf("placements=%d rank should be hidden, got %v", i, *code)
		}
		if tier := competitive.NamedRankTier(800, i); tier != 0 {
			t.Fatalf("placements=%d named tier = %d, want 0", i, tier)
		}
	}
	code := competitive.OptionalRankCode(800, 5)
	if code == nil || *code != "trailblazer_1" {
		t.Fatalf("after 5 placements rank = %v, want trailblazer_1", code)
	}
	if tier := competitive.NamedRankTier(800, 5); tier != 3 {
		t.Fatalf("trailblazer named tier = %d, want 3", tier)
	}

	// Placement counter clamps at 5.
	if got := competitive.NextPlacements(4); got != 5 {
		t.Fatalf("next from 4 = %d, want 5", got)
	}
	if got := competitive.NextPlacements(5); got != 5 {
		t.Fatalf("next from 5 = %d, want 5", got)
	}
	if got := competitive.NextPlacements(0); got != 1 {
		t.Fatalf("next from 0 = %d, want 1", got)
	}
}

func TestTeamAverage(t *testing.T) {
	t.Parallel()
	if got := competitive.TeamAverage(nil); got != 0 {
		t.Fatalf("empty avg = %d", got)
	}
	if got := competitive.TeamAverage([]int{800, 800}); got != 800 {
		t.Fatalf("equal avg = %d", got)
	}
	if got := competitive.TeamAverage([]int{800, 900}); got != 850 {
		t.Fatalf("avg 800/900 = %d, want 850", got)
	}
	// Half rounds away from zero: (800+901)/2 = 850.5 → 851
	if got := competitive.TeamAverage([]int{800, 901}); got != 851 {
		t.Fatalf("avg 800/901 = %d, want 851", got)
	}
}

func TestRankCodeFromRating_boundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rating int
		code   string
	}{
		{0, "scout_3"},
		{99, "scout_3"},
		{100, "scout_2"},
		{800, "trailblazer_1"},
		{899, "trailblazer_1"},
		{900, "navigator_3"},
		{1999, "geo_master_2"},
		{2000, "geo_master_1"},
		{5000, "geo_master_1"},
	}
	for _, tc := range cases {
		if got := competitive.RankCodeFromRating(tc.rating); got != tc.code {
			t.Fatalf("rating %d code = %s, want %s", tc.rating, got, tc.code)
		}
	}
}

func TestOutcomeForTeam(t *testing.T) {
	t.Parallel()
	win1 := 1
	out, ok := competitive.OutcomeForTeam("team_one_win", 1, nil)
	if !ok || out != competitive.OutcomeWin {
		t.Fatalf("team_one_win slot1 = %s ok=%v", out, ok)
	}
	out, ok = competitive.OutcomeForTeam("draw", 2, nil)
	if !ok || out != competitive.OutcomeDraw {
		t.Fatalf("draw = %s ok=%v", out, ok)
	}
	out, ok = competitive.OutcomeForTeam("forfeit", 1, &win1)
	if !ok || out != competitive.OutcomeWin {
		t.Fatalf("forfeit winner = %s ok=%v", out, ok)
	}
	out, ok = competitive.OutcomeForTeam("forfeit", 2, &win1)
	if !ok || out != competitive.OutcomeLoss {
		t.Fatalf("forfeit loser = %s ok=%v", out, ok)
	}
	_, ok = competitive.OutcomeForTeam("forfeit", 1, nil)
	if ok {
		t.Fatal("forfeit without winner must fail")
	}
}

func TestSoftResetRating(t *testing.T) {
	t.Parallel()
	// round(800 + 0.5*(1200-800)) = 1000
	if got := competitive.SoftResetRating(1200, 800, 5000); got != 1000 {
		t.Fatalf("soft reset 1200 = %d, want 1000", got)
	}
	if got := competitive.SoftResetRating(0, 800, 5000); got != 400 {
		t.Fatalf("soft reset 0 = %d, want 400", got)
	}
	// Floor at zero.
	if got := competitive.SoftResetRating(0, 0, 5000); got != 0 {
		t.Fatalf("soft reset floor = %d", got)
	}
}

func TestProjectPlayerProgression_hidesDuringPlacements(t *testing.T) {
	t.Parallel()
	change := competitive.RatingChange{
		UserID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Outcome:        competitive.OutcomeWin,
		BaseDelta:      16,
		AbandonPenalty: 0,
		OldRating:      800,
		NewRating:      816,
		TotalDelta:     16,
		ExpectedBPS:    5000,
	}
	dto := competitive.ProjectPlayerProgression(change, 3)
	if !dto.Applied {
		t.Fatal("expected applied")
	}
	if dto.OldRating != nil || dto.NewRating != nil || dto.OldRank != nil || dto.NewRank != nil {
		t.Fatalf("placements must hide rating/rank: %+v", dto)
	}
	if dto.Placement == nil || dto.Placement.Completed != 3 || dto.Placement.Required != 5 {
		t.Fatalf("placement progress = %+v", dto.Placement)
	}

	code := "trailblazer_1"
	change.OldRankCode = &code
	change.NewRankCode = &code
	dto = competitive.ProjectPlayerProgression(change, 5)
	if dto.OldRating == nil || *dto.OldRating != 800 {
		t.Fatalf("visible old rating = %v", dto.OldRating)
	}
	if dto.NewRating == nil || *dto.NewRating != 816 {
		t.Fatalf("visible new rating = %v", dto.NewRating)
	}
	if dto.Placement != nil {
		t.Fatal("placement progress should be nil after completions")
	}
	if dto.OldRank == nil || dto.OldRank.Code != code {
		t.Fatalf("old rank = %+v", dto.OldRank)
	}
}
