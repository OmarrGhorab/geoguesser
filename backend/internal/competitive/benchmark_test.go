package competitive_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/competitive"
)

// BenchmarkRankCodeFromRating measures pure rank mapping (p95 fixture for divisions).
func BenchmarkRankCodeFromRating(b *testing.B) {
	ratings := make([]int, 0, 21*2)
	for r := 0; r <= 2000; r += 50 {
		ratings = append(ratings, r)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = competitive.RankCodeFromRating(ratings[i%len(ratings)])
	}
}

// BenchmarkPresentationRank measures World Legend overlay path.
func BenchmarkPresentationRank(b *testing.B) {
	pos := 42
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = competitive.PresentationRank(2100+i%100, 5, &pos)
	}
}

// BenchmarkSoftResetRating measures season reset formula.
func BenchmarkSoftResetRating(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = competitive.SoftResetRating(800+i%2000, 800, 5000)
	}
}

// BenchmarkEligibleOrderingSim simulates the ≤501 ordered scan budget in-process.
func BenchmarkEligibleOrderingSim(b *testing.B) {
	type row struct {
		rating, wins int
		user         int
	}
	const n = 501
	rows := make([]row, n)
	for i := 0; i < n; i++ {
		rows[i] = row{rating: 3000 - i, wins: i % 40, user: i}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Linear scan of 501 already-ordered rows (matches query budget).
		count := 0
		for _, r := range rows {
			if r.rating >= competitive.GeoMaster1MinRating {
				count++
			}
		}
		if count != n {
			b.Fatalf("count=%d", count)
		}
	}
}
