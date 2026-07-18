package competitive

// World Legend presentation constants.
const (
	WorldLegendCode    = "world_legend"
	WorldLegendNameKey = "worldLegend"
	Top500Limit        = 500
	// GeoMaster1MinRating is the start of unbounded Geo Master I.
	GeoMaster1MinRating = 2000
	// DivisionWidth is the rating span of each standard division.
	DivisionWidth = 100
)

// Division describes one of the 21 standard 100-point rank divisions.
type Division struct {
	Code      string
	NameKey   string
	NamedTier int // Scout=1 … Geo Master=7
	Division  int // 3, 2, or 1 (III/II/I)
	MinRating int
	MaxRating int // inclusive; -1 means unbounded (Geo Master I)
}

// StandardDivisions is the ordered ladder from Scout III through Geo Master I.
var StandardDivisions = []Division{
	{Code: "scout_3", NameKey: "scout3", NamedTier: 1, Division: 3, MinRating: 0, MaxRating: 99},
	{Code: "scout_2", NameKey: "scout2", NamedTier: 1, Division: 2, MinRating: 100, MaxRating: 199},
	{Code: "scout_1", NameKey: "scout1", NamedTier: 1, Division: 1, MinRating: 200, MaxRating: 299},
	{Code: "pathfinder_3", NameKey: "pathfinder3", NamedTier: 2, Division: 3, MinRating: 300, MaxRating: 399},
	{Code: "pathfinder_2", NameKey: "pathfinder2", NamedTier: 2, Division: 2, MinRating: 400, MaxRating: 499},
	{Code: "pathfinder_1", NameKey: "pathfinder1", NamedTier: 2, Division: 1, MinRating: 500, MaxRating: 599},
	{Code: "trailblazer_3", NameKey: "trailblazer3", NamedTier: 3, Division: 3, MinRating: 600, MaxRating: 699},
	{Code: "trailblazer_2", NameKey: "trailblazer2", NamedTier: 3, Division: 2, MinRating: 700, MaxRating: 799},
	{Code: "trailblazer_1", NameKey: "trailblazer1", NamedTier: 3, Division: 1, MinRating: 800, MaxRating: 899},
	{Code: "navigator_3", NameKey: "navigator3", NamedTier: 4, Division: 3, MinRating: 900, MaxRating: 999},
	{Code: "navigator_2", NameKey: "navigator2", NamedTier: 4, Division: 2, MinRating: 1000, MaxRating: 1099},
	{Code: "navigator_1", NameKey: "navigator1", NamedTier: 4, Division: 1, MinRating: 1100, MaxRating: 1199},
	{Code: "cartographer_3", NameKey: "cartographer3", NamedTier: 5, Division: 3, MinRating: 1200, MaxRating: 1299},
	{Code: "cartographer_2", NameKey: "cartographer2", NamedTier: 5, Division: 2, MinRating: 1300, MaxRating: 1399},
	{Code: "cartographer_1", NameKey: "cartographer1", NamedTier: 5, Division: 1, MinRating: 1400, MaxRating: 1499},
	{Code: "explorer_3", NameKey: "explorer3", NamedTier: 6, Division: 3, MinRating: 1500, MaxRating: 1599},
	{Code: "explorer_2", NameKey: "explorer2", NamedTier: 6, Division: 2, MinRating: 1600, MaxRating: 1699},
	{Code: "explorer_1", NameKey: "explorer1", NamedTier: 6, Division: 1, MinRating: 1700, MaxRating: 1799},
	{Code: "geo_master_3", NameKey: "geoMaster3", NamedTier: 7, Division: 3, MinRating: 1800, MaxRating: 1899},
	{Code: "geo_master_2", NameKey: "geoMaster2", NamedTier: 7, Division: 2, MinRating: 1900, MaxRating: 1999},
	{Code: "geo_master_1", NameKey: "geoMaster1", NamedTier: 7, Division: 1, MinRating: 2000, MaxRating: -1},
}

// DivisionByCode looks up a standard division definition.
func DivisionByCode(code string) (Division, bool) {
	for _, d := range StandardDivisions {
		if d.Code == code {
			return d, true
		}
	}
	return Division{}, false
}

// DivisionFromRating returns the standard division for a rating (floor 0).
func DivisionFromRating(rating int) Division {
	code := RankCodeFromRating(rating)
	d, ok := DivisionByCode(code)
	if !ok {
		// Defensive fallback — RankCodeFromRating always maps a known code.
		return StandardDivisions[0]
	}
	return d
}

// DivisionProgress returns rating progress within the current 100-point band.
// For unbounded Geo Master I this is rating - 2000 (non-negative).
// Returns 0 when placements hide the rank.
func DivisionProgress(rating, placementsCompleted int) int {
	if placementsCompleted < PlacementsRequired {
		return 0
	}
	if rating < 0 {
		rating = 0
	}
	d := DivisionFromRating(rating)
	progress := rating - d.MinRating
	if progress < 0 {
		return 0
	}
	return progress
}

// IsWorldLegendPosition reports whether a 1-based position is in the top 500.
func IsWorldLegendPosition(position int) bool {
	return position >= 1 && position <= Top500Limit
}

// EligibleForTop500 reports whether a standing meets top-500 qualification rules
// (placements done, min matches, rating ≥ 2000). User good-standing is checked separately.
func EligibleForTop500(rating, placementsCompleted, matchesPlayed, top500MinMatches int) bool {
	if placementsCompleted < PlacementsRequired {
		return false
	}
	if top500MinMatches < 1 {
		top500MinMatches = DefaultTop500Min
	}
	if matchesPlayed < top500MinMatches {
		return false
	}
	return rating >= GeoMaster1MinRating
}

// PresentationRank builds the display rank for a visible standing.
// When top500Position is non-nil and ≤500, World Legend overlays the standard code.
// While placements are incomplete, returns nil (rank hidden).
func PresentationRank(rating, placementsCompleted int, top500Position *int) *RankDetailDTO {
	if placementsCompleted < PlacementsRequired {
		return nil
	}
	standard := StandardRankDetail(rating)
	if top500Position != nil && IsWorldLegendPosition(*top500Position) {
		return &RankDetailDTO{
			Code:     WorldLegendCode,
			NameKey:  WorldLegendNameKey,
			Division: nil,
		}
	}
	return standard
}

// StandardRankDetail maps rating to a standard division DTO (ignores World Legend).
func StandardRankDetail(rating int) *RankDetailDTO {
	d := DivisionFromRating(rating)
	div := d.Division
	return &RankDetailDTO{
		Code:     d.Code,
		NameKey:  d.NameKey,
		Division: &div,
	}
}

// EndingRankCode freezes the season-end presentation code.
// World Legend only when finalPosition is set and ≤500.
func EndingRankCode(rating int, placementsCompleted int, finalPosition *int) *string {
	if placementsCompleted < PlacementsRequired {
		return nil
	}
	if finalPosition != nil && IsWorldLegendPosition(*finalPosition) {
		code := WorldLegendCode
		return &code
	}
	code := RankCodeFromRating(rating)
	return &code
}

// PeakRankCode freezes the season peak presentation code.
// World Legend only when finalPosition is set and ≤500 (exclusive top-500 membership).
func PeakRankCode(peakRating int, placementsCompleted int, finalPosition *int) *string {
	if placementsCompleted < PlacementsRequired {
		return nil
	}
	if finalPosition != nil && IsWorldLegendPosition(*finalPosition) {
		code := WorldLegendCode
		return &code
	}
	code := RankCodeFromRating(peakRating)
	return &code
}

// RankNameKey returns the stable localization key for a rank code.
func RankNameKey(code string) string {
	if code == WorldLegendCode {
		return WorldLegendNameKey
	}
	if d, ok := DivisionByCode(code); ok {
		return d.NameKey
	}
	return code
}
