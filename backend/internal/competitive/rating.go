package competitive

import (
	"math"
	"strings"
)

// Actual score values for Elo outcomes.
const (
	ActualWin  = 1.0
	ActualDraw = 0.5
	ActualLoss = 0.0
)

// ExpectedProbability returns the Elo expected score for ownAvg against opponentAvg.
// Formula: 1 / (1 + 10^((opponent_average - own_average) / 400)).
func ExpectedProbability(ownAvg, opponentAvg int) float64 {
	diff := float64(opponentAvg-ownAvg) / 400.0
	return 1.0 / (1.0 + math.Pow(10, diff))
}

// ExpectedBPS returns the expected result in basis points (0–10000).
func ExpectedBPS(ownAvg, opponentAvg int) int {
	p := ExpectedProbability(ownAvg, opponentAvg)
	return int(math.Round(p * 10000.0))
}

// ActualFromOutcome maps win/loss/draw to the Elo actual score.
// Unknown outcomes default to loss (0) for safety.
func ActualFromOutcome(outcome string) float64 {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case OutcomeWin:
		return ActualWin
	case OutcomeDraw:
		return ActualDraw
	case OutcomeLoss:
		return ActualLoss
	default:
		return ActualLoss
	}
}

// BaseDelta computes round(K × (actual − expected)).
func BaseDelta(k int, actual, expected float64) int {
	if k <= 0 {
		k = DefaultEloK
	}
	return int(math.Round(float64(k) * (actual - expected)))
}

// BaseDeltaForAverages is the team-average Elo base delta for an outcome.
func BaseDeltaForAverages(k, ownAvg, opponentAvg int, outcome string) int {
	expected := ExpectedProbability(ownAvg, opponentAvg)
	actual := ActualFromOutcome(outcome)
	return BaseDelta(k, actual, expected)
}

// ApplyRating applies base delta and abandon penalty with a floor of zero.
// totalDelta is the realized change after the floor (new − old).
func ApplyRating(oldRating, baseDelta, abandonPenalty int) (newRating, totalDelta int) {
	if oldRating < 0 {
		oldRating = 0
	}
	// abandonPenalty is non-positive (0 or -15).
	sum := oldRating + baseDelta + abandonPenalty
	if sum < 0 {
		sum = 0
	}
	return sum, sum - oldRating
}

// TeamAverage returns the integer average of ratings, rounded half away from zero.
// Empty input yields 0.
func TeamAverage(ratings []int) int {
	if len(ratings) == 0 {
		return 0
	}
	sum := 0
	for _, r := range ratings {
		sum += r
	}
	n := len(ratings)
	// Round half away from zero for positive competitive ratings.
	if sum >= 0 {
		return (sum + n/2) / n
	}
	return -((-sum + n/2) / n)
}

// AbandonPenaltyValue returns the stored abandon penalty (0 or -magnitude).
func AbandonPenaltyValue(abandoned bool, magnitude int) int {
	if !abandoned {
		return 0
	}
	if magnitude < 0 {
		magnitude = -magnitude
	}
	if magnitude == 0 {
		magnitude = DefaultAbandonPenalty
	}
	return -magnitude
}

// RankCodeFromRating maps a non-negative rating to a standard division code.
// World Legend is presentation-only and never returned here.
func RankCodeFromRating(rating int) string {
	if rating < 0 {
		rating = 0
	}
	switch {
	case rating < 100:
		return "scout_3"
	case rating < 200:
		return "scout_2"
	case rating < 300:
		return "scout_1"
	case rating < 400:
		return "pathfinder_3"
	case rating < 500:
		return "pathfinder_2"
	case rating < 600:
		return "pathfinder_1"
	case rating < 700:
		return "trailblazer_3"
	case rating < 800:
		return "trailblazer_2"
	case rating < 900:
		return "trailblazer_1"
	case rating < 1000:
		return "navigator_3"
	case rating < 1100:
		return "navigator_2"
	case rating < 1200:
		return "navigator_1"
	case rating < 1300:
		return "cartographer_3"
	case rating < 1400:
		return "cartographer_2"
	case rating < 1500:
		return "cartographer_1"
	case rating < 1600:
		return "explorer_3"
	case rating < 1700:
		return "explorer_2"
	case rating < 1800:
		return "explorer_1"
	case rating < 1900:
		return "geo_master_3"
	case rating < 2000:
		return "geo_master_2"
	default:
		return "geo_master_1"
	}
}

// NamedRankTier returns a coarse named-rank index for party spread checks.
// Scout=1 … Geo Master=7. Returns 0 when placements hide the rank.
func NamedRankTier(rating int, placementsCompleted int) int {
	if placementsCompleted < PlacementsRequired {
		return 0
	}
	return NamedRankTierFromRating(rating)
}

// NamedRankTierFromRating maps rating to named-rank index ignoring placements.
// Scout=1, Pathfinder=2, Trailblazer=3, Navigator=4, Cartographer=5, Explorer=6, Geo Master=7.
func NamedRankTierFromRating(rating int) int {
	if rating < 0 {
		rating = 0
	}
	// Each named rank spans 300 points; Geo Master starts at 1800.
	tier := rating/300 + 1
	if tier > 7 {
		tier = 7
	}
	if tier < 1 {
		tier = 1
	}
	return tier
}

// OutcomeForTeam maps match result + team slot to win/loss/draw.
// winnerTeamSlot is required for forfeit/abandoned/non-draw results when set.
func OutcomeForTeam(result string, teamSlot int, winnerTeamSlot *int) (string, bool) {
	switch result {
	case "team_one_win":
		if teamSlot == TeamSlotOne {
			return OutcomeWin, true
		}
		return OutcomeLoss, true
	case "team_two_win":
		if teamSlot == TeamSlotTwo {
			return OutcomeWin, true
		}
		return OutcomeLoss, true
	case "draw":
		return OutcomeDraw, true
	case "forfeit", "abandoned":
		if winnerTeamSlot == nil {
			return "", false
		}
		if teamSlot == *winnerTeamSlot {
			return OutcomeWin, true
		}
		return OutcomeLoss, true
	default:
		return "", false
	}
}

// NextPlacements returns the placements_completed value after one ranked match.
func NextPlacements(current int) int {
	if current < 0 {
		current = 0
	}
	if current >= PlacementsRequired {
		return PlacementsRequired
	}
	return current + 1
}

// OptionalRankCode returns a rank code pointer when placements make rank visible.
func OptionalRankCode(rating, placementsCompleted int) *string {
	if placementsCompleted < PlacementsRequired {
		return nil
	}
	code := RankCodeFromRating(rating)
	return &code
}

// SoftResetRating computes next-season start: round(800 + 0.5 × (old − 800)), floor 0.
// resetFactorBPS is typically 5000 (50%). initialRating is typically 800.
func SoftResetRating(previousRating, initialRating, resetFactorBPS int) int {
	if initialRating < 0 {
		initialRating = 0
	}
	if resetFactorBPS < 0 {
		resetFactorBPS = 0
	}
	if resetFactorBPS > 10000 {
		resetFactorBPS = 10000
	}
	// round(initial + (resetFactor/10000) * (previous - initial))
	delta := previousRating - initialRating
	scaled := int(math.Round(float64(delta) * float64(resetFactorBPS) / 10000.0))
	next := initialRating + scaled
	if next < 0 {
		return 0
	}
	return next
}
