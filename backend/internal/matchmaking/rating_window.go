package matchmaking

import "time"

// Ranked matchmaking window constants (research.md / FR matchmaking policy).
const (
	// RatingWindowInitial is the initial half-width (±) around team-average rating.
	RatingWindowInitial = 100
	// RatingWindowStep expands the half-width after each expansion interval.
	RatingWindowStep = 50
	// RatingWindowStepEvery is how long a ticket waits between expansions.
	RatingWindowStepEvery = 30 * time.Second
	// RatingWindowCap is the maximum half-width (±).
	RatingWindowCap = 400
	// MaxNamedRankSpread is the maximum difference between highest and lowest
	// named-rank tiers allowed in a premade ranked party (inclusive: at most two apart).
	MaxNamedRankSpread = 2
	// DefaultHiddenPlacementRating is the starting hidden rating for new profiles.
	DefaultHiddenPlacementRating = 800
	// NamedRankTierCount is the number of coarse named-rank bands (Scout … Geo Master).
	NamedRankTierCount = 7
	// NamedRankBandWidth is the rating width of one named rank (3 × 100-point divisions).
	NamedRankBandWidth = 300
)

// RatingWindowHalfWidth returns the current ranked search half-width for a ticket
// that has been waiting waited duration. Starts at ±100, expands +50 every 30s, caps at ±400.
func RatingWindowHalfWidth(waited time.Duration) int {
	if waited < 0 {
		waited = 0
	}
	steps := int(waited / RatingWindowStepEvery)
	half := RatingWindowInitial + RatingWindowStep*steps
	if half > RatingWindowCap {
		return RatingWindowCap
	}
	return half
}

// RatingWindowForAverage builds the public rating window around a team-average rating.
// The lower bound is floored at zero.
func RatingWindowForAverage(teamAvgRating int, waited time.Duration) RatingWindow {
	half := RatingWindowHalfWidth(waited)
	min := teamAvgRating - half
	if min < 0 {
		min = 0
	}
	return RatingWindow{
		Minimum: min,
		Maximum: teamAvgRating + half,
	}
}

// TeamAveragesCompatible reports whether two team averages may match given their wait times.
// Each ticket expands independently; a pair matches when the absolute difference is within
// the wider of the two half-widths (so long-waiting tickets can still find opponents).
func TeamAveragesCompatible(avgA, avgB int, waitedA, waitedB time.Duration) bool {
	delta := avgA - avgB
	if delta < 0 {
		delta = -delta
	}
	halfA := RatingWindowHalfWidth(waitedA)
	halfB := RatingWindowHalfWidth(waitedB)
	allowed := halfA
	if halfB > allowed {
		allowed = halfB
	}
	return delta <= allowed
}

// TeamAverageRating returns the integer mean of member ratings (placement uses hidden rating).
// Empty input returns 0.
func TeamAverageRating(ratings []int) int {
	if len(ratings) == 0 {
		return 0
	}
	sum := 0
	for _, r := range ratings {
		sum += r
	}
	return sum / len(ratings)
}

// NamedRankTier returns the coarse named-rank index for a rating.
// Scout=0, Pathfinder=1, Trailblazer=2, Navigator=3, Cartographer=4, Explorer=5, Geo Master=6.
// Placement players use this mapping on their hidden rating for party-spread checks.
func NamedRankTier(rating int) int {
	if rating < 0 {
		rating = 0
	}
	tier := rating / NamedRankBandWidth
	if tier >= NamedRankTierCount {
		return NamedRankTierCount - 1
	}
	return tier
}

// EffectiveNamedRankTier prefers an explicit standing tier when set, otherwise derives from rating.
// Zero NamedRankTier with incomplete placements is treated as derived-from-hidden-rating.
func EffectiveNamedRankTier(rating, namedRankTier, placementsCompleted int, rankCode string) int {
	if rankCode != "" || placementsCompleted >= 5 {
		if namedRankTier > 0 {
			return namedRankTier
		}
		return NamedRankTier(rating)
	}
	// Placement / hidden: always derive from hidden rating.
	return NamedRankTier(rating)
}

// PartyNamedRankSpreadOK reports whether all named-rank tiers are within MaxNamedRankSpread.
// Empty tiers are allowed (solo / no data).
func PartyNamedRankSpreadOK(tiers []int) bool {
	if len(tiers) <= 1 {
		return true
	}
	min, max := tiers[0], tiers[0]
	for _, t := range tiers[1:] {
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}
	return max-min <= MaxNamedRankSpread
}

// PartySpreadFromStandings extracts effective named-rank tiers and reports spread validity.
func PartySpreadFromStandings(standings []CompetitiveStandingSnapshot) (tiers []int, ok bool) {
	tiers = make([]int, 0, len(standings))
	for _, st := range standings {
		tiers = append(tiers, EffectiveNamedRankTier(st.Rating, st.NamedRankTier, st.PlacementsCompleted, st.RankCode))
	}
	return tiers, PartyNamedRankSpreadOK(tiers)
}
