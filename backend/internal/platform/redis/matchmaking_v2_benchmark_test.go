package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// BenchmarkMatchmakingV2ClaimCasualWithQueueLoad measures claim latency against a
// large Casual queue with the production scan bound of 20 candidates.
//
// Default fixture size is 10,000 searcher tickets. Override with:
//
//	MATCHMAKING_V2_BENCH_TICKETS=1000 go test ./internal/platform/redis/ -bench MatchmakingV2 -count=1
//
// Modes exercised (sub-benchmarks):
//   - roster=1 casual_solo
//   - roster=2 casual_duo
//   - roster=4 casual_squad
//
// Each iteration claims at most two tickets (scan ≤20), then requeues them so the
// queue depth stays stable across b.N iterations.
func BenchmarkMatchmakingV2ClaimCasualWithQueueLoad(b *testing.B) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/12"
	}
	opt, err := goredis.ParseURL(url)
	if err != nil {
		b.Skipf("invalid REDIS_URL: %v", err)
	}
	client := goredis.NewClient(opt)
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("redis unavailable: %v", err)
	}
	defer func() { _ = client.Close() }()

	members := 10000
	if raw := os.Getenv("MATCHMAKING_V2_BENCH_TICKETS"); raw != "" {
		n, parseErr := strconv.Atoi(raw)
		if parseErr != nil || n < 4 {
			b.Fatalf("MATCHMAKING_V2_BENCH_TICKETS must be an integer >= 4, got %q", raw)
		}
		members = n
	}

	for _, tc := range []struct {
		name     string
		mode     string
		teamSize int
	}{
		{name: "roster1_casual_solo", mode: "casual_solo", teamSize: 1},
		{name: "roster2_casual_duo", mode: "casual_duo", teamSize: 2},
		{name: "roster4_casual_squad", mode: "casual_squad", teamSize: 4},
	} {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			runV2ClaimBenchmark(b, client, tc.mode, tc.teamSize, members)
		})
	}
}

func runV2ClaimBenchmark(b *testing.B, client *goredis.Client, mode string, teamSize, members int) {
	b.Helper()
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()

	// Seed members tickets (each ticket is one team roster).
	allUsers := make([]uuid.UUID, 0, members*teamSize)
	ticketIDs := make([]string, 0, members)
	for i := 0; i < members; i++ {
		roster := make([]uuid.UUID, teamSize)
		for j := 0; j < teamSize; j++ {
			roster[j] = uuid.New()
			allUsers = append(allUsers, roster[j])
		}
		partyID := uuid.Nil
		partyVersion := int64(0)
		if teamSize > 1 {
			partyID = uuid.New()
			partyVersion = 1
		}
		ticket, err := coord.JoinTicket(ctx, JoinTicketInput{
			Mode:         mode,
			UserIDs:      roster,
			PartyID:      partyID,
			PartyVersion: partyVersion,
			TeamSize:     teamSize,
		}, now.Add(time.Duration(i)*time.Millisecond), 10*time.Minute)
		if err != nil {
			b.Fatalf("seed join %d: %v", i, err)
		}
		ticketIDs = append(ticketIDs, ticket.TicketID)
	}

	b.Cleanup(func() {
		for _, u := range allUsers {
			_ = client.Del(ctx, matchmakingV2UserKey(u)).Err()
		}
		for _, tid := range ticketIDs {
			_ = client.Del(ctx, matchmakingV2TicketKey(tid)).Err()
		}
		_ = client.Del(ctx, matchmakingV2QueueKey(mode), matchmakingV2ClaimsIndexKey()).Err()
	})

	// Confirm queue depth before timing.
	card, err := client.ZCard(ctx, matchmakingV2QueueKey(mode)).Result()
	if err != nil {
		b.Fatalf("zcard: %v", err)
	}
	if card != int64(members) {
		b.Fatalf("queue depth = %d, want %d", card, members)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Production scan bound is 20 candidate tickets.
		claim, claimErr := coord.ClaimTickets(ctx, mode, now.Add(time.Minute), 15*time.Second, 20, 0)
		if claimErr != nil {
			b.Fatalf("claim: %v", claimErr)
		}
		if claim == nil {
			b.Fatalf("expected claim at iteration %d (queue should stay deep)", i)
		}
		// Requeue both tickets so subsequent iterations keep a full queue.
		if relErr := coord.ReleaseTeamClaim(ctx, claim, true, true, 10*time.Minute); relErr != nil {
			b.Fatalf("release: %v", relErr)
		}
	}
	b.StopTimer()

	// Report fixture parameters for CI documentation.
	b.ReportMetric(float64(members), "tickets")
	b.ReportMetric(20, "scan_limit")
	b.ReportMetric(float64(teamSize), "roster_size")
	_ = fmt.Sprintf("%s roster=%d tickets=%d scan=20", mode, teamSize, members)
}
