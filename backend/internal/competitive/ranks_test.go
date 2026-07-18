package competitive_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/competitive"
)

func TestStandardDivisions_all21HundredPointBands(t *testing.T) {
	t.Parallel()
	if len(competitive.StandardDivisions) != 21 {
		t.Fatalf("divisions = %d, want 21", len(competitive.StandardDivisions))
	}

	// Sample every 100-point boundary from 0 through 2000 inclusive.
	want := map[int]string{
		0: "scout_3", 99: "scout_3",
		100: "scout_2", 199: "scout_2",
		200: "scout_1", 299: "scout_1",
		300: "pathfinder_3", 399: "pathfinder_3",
		400: "pathfinder_2", 499: "pathfinder_2",
		500: "pathfinder_1", 599: "pathfinder_1",
		600: "trailblazer_3", 699: "trailblazer_3",
		700: "trailblazer_2", 799: "trailblazer_2",
		800: "trailblazer_1", 899: "trailblazer_1",
		900: "navigator_3", 999: "navigator_3",
		1000: "navigator_2", 1099: "navigator_2",
		1100: "navigator_1", 1199: "navigator_1",
		1200: "cartographer_3", 1299: "cartographer_3",
		1300: "cartographer_2", 1399: "cartographer_2",
		1400: "cartographer_1", 1499: "cartographer_1",
		1500: "explorer_3", 1599: "explorer_3",
		1600: "explorer_2", 1699: "explorer_2",
		1700: "explorer_1", 1799: "explorer_1",
		1800: "geo_master_3", 1899: "geo_master_3",
		1900: "geo_master_2", 1999: "geo_master_2",
		2000: "geo_master_1", 2500: "geo_master_1", 9999: "geo_master_1",
	}
	for rating, code := range want {
		if got := competitive.RankCodeFromRating(rating); got != code {
			t.Fatalf("rating %d code = %s, want %s", rating, got, code)
		}
		d := competitive.DivisionFromRating(rating)
		if d.Code != code {
			t.Fatalf("rating %d division code = %s, want %s", rating, d.Code, code)
		}
	}

	// Floor at zero for negative ratings.
	if got := competitive.RankCodeFromRating(-50); got != "scout_3" {
		t.Fatalf("negative rating code = %s, want scout_3", got)
	}

	// Geo Master I is unbounded.
	d := competitive.DivisionFromRating(5000)
	if d.MaxRating != -1 || d.Code != "geo_master_1" {
		t.Fatalf("geo master I max = %d code = %s", d.MaxRating, d.Code)
	}
}

func TestPlacementHiding(t *testing.T) {
	t.Parallel()
	for placements := 0; placements < competitive.PlacementsRequired; placements++ {
		if rank := competitive.PresentationRank(2100, placements, nil); rank != nil {
			t.Fatalf("placements %d must hide rank, got %+v", placements, rank)
		}
		if progress := competitive.DivisionProgress(850, placements); progress != 0 {
			t.Fatalf("placements %d progress = %d, want 0", placements, progress)
		}
		if competitive.EligibleForTop500(2500, placements, 100, 25) {
			t.Fatalf("placements %d must not be top-500 eligible", placements)
		}
	}
	if rank := competitive.PresentationRank(850, 5, nil); rank == nil || rank.Code != "trailblazer_1" {
		t.Fatalf("after placements rank = %+v", rank)
	}
}

func TestWorldLegendOverlay(t *testing.T) {
	t.Parallel()
	pos1 := 1
	pos500 := 500
	pos501 := 501

	rank := competitive.PresentationRank(2500, 5, &pos1)
	if rank == nil || rank.Code != competitive.WorldLegendCode || rank.Division != nil {
		t.Fatalf("pos 1 rank = %+v", rank)
	}
	rank = competitive.PresentationRank(2500, 5, &pos500)
	if rank == nil || rank.Code != competitive.WorldLegendCode {
		t.Fatalf("pos 500 rank = %+v", rank)
	}
	rank = competitive.PresentationRank(2500, 5, &pos501)
	if rank == nil || rank.Code != "geo_master_1" {
		t.Fatalf("pos 501 must fall back to standard: %+v", rank)
	}
	// Standard rank always from rating, never world_legend.
	std := competitive.StandardRankDetail(2500)
	if std == nil || std.Code != "geo_master_1" {
		t.Fatalf("standard rank = %+v", std)
	}
	if !competitive.IsWorldLegendPosition(500) || competitive.IsWorldLegendPosition(501) {
		t.Fatal("world legend position bounds wrong")
	}
}

func TestDivisionProgress(t *testing.T) {
	t.Parallel()
	if got := competitive.DivisionProgress(834, 5); got != 34 {
		t.Fatalf("trailblazer_1 progress = %d, want 34", got)
	}
	if got := competitive.DivisionProgress(2000, 5); got != 0 {
		t.Fatalf("geo_master_1 at floor progress = %d, want 0", got)
	}
	if got := competitive.DivisionProgress(2034, 5); got != 34 {
		t.Fatalf("geo_master_1 progress = %d, want 34", got)
	}
	if got := competitive.DivisionProgress(99, 5); got != 99 {
		t.Fatalf("scout_3 progress = %d, want 99", got)
	}
}

func TestEligibleForTop500(t *testing.T) {
	t.Parallel()
	if !competitive.EligibleForTop500(2000, 5, 25, 25) {
		t.Fatal("boundary 2000/25 should be eligible")
	}
	if competitive.EligibleForTop500(1999, 5, 100, 25) {
		t.Fatal("rating 1999 not eligible")
	}
	if competitive.EligibleForTop500(2500, 5, 24, 25) {
		t.Fatal("24 matches not eligible")
	}
}

func TestFrozenEndingAndPeakRanks(t *testing.T) {
	t.Parallel()
	pos := 12
	ending := competitive.EndingRankCode(2400, 5, &pos)
	if ending == nil || *ending != competitive.WorldLegendCode {
		t.Fatalf("ending top500 = %v", ending)
	}
	peak := competitive.PeakRankCode(2500, 5, &pos)
	if peak == nil || *peak != competitive.WorldLegendCode {
		t.Fatalf("peak top500 = %v", peak)
	}
	pos501 := 501
	ending = competitive.EndingRankCode(2400, 5, &pos501)
	if ending == nil || *ending != "geo_master_1" {
		t.Fatalf("ending pos501 = %v", ending)
	}
	// Incomplete placements freeze nothing.
	if competitive.EndingRankCode(2400, 3, &pos) != nil {
		t.Fatal("incomplete placements must not freeze ending")
	}
}
