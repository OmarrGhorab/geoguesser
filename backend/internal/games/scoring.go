package games

import "math"

const (
	ScoringVersionV1         = 1
	maxRoundScore            = 5000
	decayFactorKilometers    = 1492.0
	fullScoreThresholdMeters = 25
	earthRadiusMeters        = 6371000.0
)

// DistanceMeters returns the haversine distance between two lat/lng points.
func DistanceMeters(lat1, lng1, lat2, lng2 float64) int {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return int(math.Round(earthRadiusMeters * c))
}

// ScoreV1 returns the version 1 score for a distance in meters.
func ScoreV1(distanceMeters int) int {
	if distanceMeters <= fullScoreThresholdMeters {
		return maxRoundScore
	}
	distanceKm := float64(distanceMeters) / 1000
	score := int(math.Round(maxRoundScore * math.Exp(-distanceKm/decayFactorKilometers)))
	if score < 0 {
		return 0
	}
	if score > maxRoundScore {
		return maxRoundScore
	}
	return score
}
