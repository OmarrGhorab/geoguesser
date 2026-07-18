package redis

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func testRedisV2(t *testing.T) *goredis.Client {
	t.Helper()
	return testRedis(t)
}

func cleanupV2Mode(ctx context.Context, client *goredis.Client, mode string, users ...uuid.UUID) {
	for _, u := range users {
		_ = client.Del(ctx, matchmakingV2UserKey(u)).Err()
	}
	_ = client.Del(ctx, matchmakingV2QueueKey(mode), matchmakingV2ClaimsIndexKey()).Err()
}

func TestMatchmakingV2KeyBuilders(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if got := matchmakingV2QueueKey("casual_solo"); got != "matchmaking:v2:queue:casual_solo" {
		t.Fatalf("queue key = %q", got)
	}
	if got := matchmakingV2TicketKey("t1"); got != "matchmaking:v2:ticket:t1" {
		t.Fatalf("ticket key = %q", got)
	}
	if got := matchmakingV2UserKey(userID); got != "matchmaking:v2:user:00000000-0000-0000-0000-000000000001" {
		t.Fatalf("user key = %q", got)
	}
	if got := matchmakingV2ClaimKey("c1"); got != "matchmaking:v2:claim:c1" {
		t.Fatalf("claim key = %q", got)
	}
	if got := matchmakingV2ClaimsIndexKey(); got != "matchmaking:v2:claims" {
		t.Fatalf("claims index key = %q", got)
	}
}

func TestMatchmakingV2JoinSoloDuoSquad(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()

	// Solo
	soloUser := uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, "casual_solo", soloUser) })
	solo, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode:     "casual_solo",
		UserIDs:  []uuid.UUID{soloUser},
		TeamSize: 1,
	}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("solo join: %v", err)
	}
	if solo.State != V2StateSearching || solo.TeamSize != 1 || len(solo.UserIDs) != 1 || solo.UserIDs[0] != soloUser {
		t.Fatalf("solo ticket unexpected: %+v", solo)
	}
	if solo.TicketID == "" {
		t.Fatal("solo ticket id empty")
	}

	// Duo
	duoA, duoB := uuid.New(), uuid.New()
	partyID := uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, "casual_duo", duoA, duoB) })
	duo, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode:         "casual_duo",
		UserIDs:      []uuid.UUID{duoA, duoB},
		PartyID:      partyID,
		PartyVersion: 3,
		TeamSize:     2,
	}, now.Add(time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("duo join: %v", err)
	}
	if duo.TeamSize != 2 || duo.PartyVersion != 3 || duo.PartyID != partyID {
		t.Fatalf("duo ticket unexpected: %+v", duo)
	}
	if len(duo.UserIDs) != 2 || duo.UserIDs[0] != duoA || duo.UserIDs[1] != duoB {
		t.Fatalf("duo roster order: %+v", duo.UserIDs)
	}
	// Both members resolve the same ticket.
	for _, u := range []uuid.UUID{duoA, duoB} {
		got, getErr := coord.GetTicketByUser(ctx, u)
		if getErr != nil || got == nil || got.TicketID != duo.TicketID {
			t.Fatalf("member %s ticket = %+v err=%v", u, got, getErr)
		}
	}

	// Squad
	squad := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	t.Cleanup(func() { cleanupV2Mode(ctx, client, "casual_squad", squad...) })
	sq, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode:         "casual_squad",
		UserIDs:      squad,
		PartyID:      uuid.New(),
		PartyVersion: 1,
		TeamSize:     4,
	}, now.Add(2*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("squad join: %v", err)
	}
	if sq.TeamSize != 4 || len(sq.UserIDs) != 4 {
		t.Fatalf("squad ticket unexpected: %+v", sq)
	}
}

func TestMatchmakingV2JoinIdempotent(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	user := uuid.New()
	mode := "casual_solo"
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, user) })

	first, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{user}, TeamSize: 1,
	}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("first join: %v", err)
	}
	second, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{user}, TeamSize: 1,
	}, now.Add(5*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("second join: %v", err)
	}
	if second.TicketID != first.TicketID {
		t.Fatalf("ticket id changed on duplicate join: %s vs %s", first.TicketID, second.TicketID)
	}
	if second.EnqueuedAtMs != first.EnqueuedAtMs {
		t.Fatalf("priority score changed: %d vs %d", first.EnqueuedAtMs, second.EnqueuedAtMs)
	}
}

func TestMatchmakingV2PerUserExclusivity(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	userA, userB, userC := uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() {
		cleanupV2Mode(ctx, client, "casual_solo", userA)
		cleanupV2Mode(ctx, client, "casual_duo", userA, userB, userC)
	})

	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: "casual_solo", UserIDs: []uuid.UUID{userA}, TeamSize: 1,
	}, now, 30*time.Second); err != nil {
		t.Fatalf("solo join A: %v", err)
	}

	// A cannot join a duo while already searching solo.
	_, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: "casual_duo", UserIDs: []uuid.UUID{userA, userB}, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now.Add(time.Second), 30*time.Second)
	if !V2UserConflict(err) {
		t.Fatalf("expected user conflict, got %v", err)
	}

	// Partial occupancy: B free, A on another ticket — conflict when joining duo with C free? A still bound.
	_, err = coord.JoinTicket(ctx, JoinTicketInput{
		Mode: "casual_duo", UserIDs: []uuid.UUID{userB, userC}, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now.Add(2*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("unrelated duo should join: %v", err)
	}

	// Cross-ticket exclusivity: B already on duo cannot start solo.
	_, err = coord.JoinTicket(ctx, JoinTicketInput{
		Mode: "casual_solo", UserIDs: []uuid.UUID{userB}, TeamSize: 1,
	}, now.Add(3*time.Second), 30*time.Second)
	if !V2UserConflict(err) {
		t.Fatalf("expected exclusivity conflict for B, got %v", err)
	}
}

func TestMatchmakingV2PartyVersionValidation(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	a, b := uuid.New(), uuid.New()
	partyID := uuid.New()
	mode := "casual_duo"
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, a, b) })

	first, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{a, b}, PartyID: partyID, PartyVersion: 1, TeamSize: 2,
	}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join v1: %v", err)
	}

	// Same version is idempotent.
	same, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{a, b}, PartyID: partyID, PartyVersion: 1, TeamSize: 2,
	}, now.Add(time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("rejoin same version: %v", err)
	}
	if same.TicketID != first.TicketID {
		t.Fatalf("expected same ticket for same party version")
	}

	// Different party version must not silently replace; leader should leave first.
	_, err = coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{a, b}, PartyID: partyID, PartyVersion: 2, TeamSize: 2,
	}, now.Add(2*time.Second), 30*time.Second)
	if !V2PartyVersionMismatch(err) {
		t.Fatalf("expected party version mismatch, got %v", err)
	}

	// After leave, new version can join.
	if err := coord.LeaveTicket(ctx, a); err != nil {
		t.Fatalf("leave: %v", err)
	}
	next, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{a, b}, PartyID: partyID, PartyVersion: 2, TeamSize: 2,
	}, now.Add(3*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("join v2 after leave: %v", err)
	}
	if next.PartyVersion != 2 || next.TicketID == first.TicketID {
		t.Fatalf("expected new ticket with version 2, got %+v", next)
	}
}

func TestMatchmakingV2LeaseRenewal(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	user := uuid.New()
	mode := "casual_solo"
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, user) })

	first, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{user}, TeamSize: 1,
	}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	renewed, err := coord.RenewTicketLease(ctx, user, 30*time.Second, now.Add(10*time.Second))
	if err != nil || renewed == nil {
		t.Fatalf("renew: %v %+v", err, renewed)
	}
	if renewed.TicketID != first.TicketID {
		t.Fatalf("ticket id changed on renew")
	}
	if renewed.EnqueuedAtMs != first.EnqueuedAtMs {
		t.Fatalf("renew changed priority")
	}
	if renewed.LeaseExpiresAtMs <= first.LeaseExpiresAtMs {
		t.Fatalf("lease not extended: %d vs %d", renewed.LeaseExpiresAtMs, first.LeaseExpiresAtMs)
	}

	// Expired searching lease is cleared on renew.
	expiredUser := uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, expiredUser) })
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{expiredUser}, TeamSize: 1,
	}, now, 2*time.Second); err != nil {
		t.Fatalf("join expired fixture: %v", err)
	}
	gone, err := coord.RenewTicketLease(ctx, expiredUser, 30*time.Second, now.Add(5*time.Second))
	if err != nil {
		t.Fatalf("renew expired: %v", err)
	}
	if gone != nil {
		t.Fatalf("expected nil after expired lease cleanup, got %+v", gone)
	}
}

func TestMatchmakingV2AtomicEqualRosterClaim(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	mode := "casual_duo"

	team1 := []uuid.UUID{uuid.New(), uuid.New()}
	team2 := []uuid.UUID{uuid.New(), uuid.New()}
	team3 := []uuid.UUID{uuid.New(), uuid.New()}
	all := append(append(append([]uuid.UUID{}, team1...), team2...), team3...)
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, all...) })

	// One ticket alone cannot form a claim.
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: team1, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now, 30*time.Second); err != nil {
		t.Fatalf("join team1: %v", err)
	}
	claim, err := coord.ClaimTickets(ctx, mode, now.Add(time.Second), 15*time.Second, 20, 0)
	if err != nil {
		t.Fatalf("claim alone: %v", err)
	}
	if claim != nil {
		t.Fatalf("expected nil claim with single candidate, got %+v", claim)
	}

	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: team2, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now.Add(2*time.Second), 30*time.Second); err != nil {
		t.Fatalf("join team2: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: team3, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now.Add(3*time.Second), 30*time.Second); err != nil {
		t.Fatalf("join team3: %v", err)
	}

	firstClaim, err := coord.ClaimTickets(ctx, mode, now.Add(4*time.Second), 15*time.Second, 20, 0)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if firstClaim == nil {
		t.Fatal("expected claim for two duo tickets")
	}
	if firstClaim.TeamSize != 2 {
		t.Fatalf("team size = %d", firstClaim.TeamSize)
	}
	if firstClaim.FormationKey == "" || firstClaim.ClaimID == "" {
		t.Fatalf("missing claim identity: %+v", firstClaim)
	}
	// Oldest two tickets (team1, team2).
	if !sameUUIDSet(firstClaim.UserIDsA, team1) && !sameUUIDSet(firstClaim.UserIDsB, team1) {
		t.Fatalf("team1 missing from claim: %+v", firstClaim)
	}
	if !sameUUIDSet(firstClaim.UserIDsA, team2) && !sameUUIDSet(firstClaim.UserIDsB, team2) {
		t.Fatalf("team2 missing from claim: %+v", firstClaim)
	}
	// Disjoint rosters.
	if !disjointUUIDSets(firstClaim.UserIDsA, firstClaim.UserIDsB) {
		t.Fatalf("overlapping rosters in claim: %+v", firstClaim)
	}

	// Tickets marked claimed; leave blocked.
	for _, u := range team1 {
		ticket, getErr := coord.GetTicketByUser(ctx, u)
		if getErr != nil || ticket == nil || ticket.State != V2StateClaimed || ticket.ClaimID != firstClaim.ClaimID {
			t.Fatalf("team1 member state: %+v err=%v", ticket, getErr)
		}
	}
	if err := coord.LeaveTicket(ctx, team1[0]); !V2ClaimInProgress(err) {
		t.Fatalf("leave claimed: %v", err)
	}

	// Second claim must not re-claim the same tickets; team3 remains alone.
	secondClaim, err := coord.ClaimTickets(ctx, mode, now.Add(5*time.Second), 15*time.Second, 20, 0)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if secondClaim != nil {
		t.Fatalf("expected no second claim with only one remaining ticket, got %+v", secondClaim)
	}

	// Finalize clears claimed tickets.
	if err := coord.FinalizeTeamClaim(ctx, firstClaim); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	for _, u := range append(team1, team2...) {
		ticket, getErr := coord.GetTicketByUser(ctx, u)
		if getErr != nil {
			t.Fatalf("get %s: %v", u, getErr)
		}
		if ticket != nil {
			t.Fatalf("expected cleared ticket after finalize, got %+v", ticket)
		}
	}

	// Remaining team3 still searching.
	ticket3, err := coord.GetTicketByUser(ctx, team3[0])
	if err != nil || ticket3 == nil || ticket3.State != V2StateSearching {
		t.Fatalf("team3 state = %+v err=%v", ticket3, err)
	}
}

func TestMatchmakingV2ClaimSoloAndSquad(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()

	// Solo 1v1
	modeSolo := "casual_solo"
	u1, u2 := uuid.New(), uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, modeSolo, u1, u2) })
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: modeSolo, UserIDs: []uuid.UUID{u1}, TeamSize: 1}, now, 30*time.Second); err != nil {
		t.Fatalf("solo1: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: modeSolo, UserIDs: []uuid.UUID{u2}, TeamSize: 1}, now.Add(time.Millisecond), 30*time.Second); err != nil {
		t.Fatalf("solo2: %v", err)
	}
	soloClaim, err := coord.ClaimTickets(ctx, modeSolo, now.Add(time.Second), 15*time.Second, 20, 0)
	if err != nil || soloClaim == nil || soloClaim.TeamSize != 1 {
		t.Fatalf("solo claim: %+v err=%v", soloClaim, err)
	}
	_ = coord.FinalizeTeamClaim(ctx, soloClaim)

	// Squad 4v4
	modeSquad := "casual_squad"
	s1 := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	s2 := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	t.Cleanup(func() { cleanupV2Mode(ctx, client, modeSquad, append(s1, s2...)...) })
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: modeSquad, UserIDs: s1, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 4,
	}, now, 30*time.Second); err != nil {
		t.Fatalf("squad1: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: modeSquad, UserIDs: s2, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 4,
	}, now.Add(time.Millisecond), 30*time.Second); err != nil {
		t.Fatalf("squad2: %v", err)
	}
	squadClaim, err := coord.ClaimTickets(ctx, modeSquad, now.Add(time.Second), 15*time.Second, 20, 0)
	if err != nil || squadClaim == nil || squadClaim.TeamSize != 4 {
		t.Fatalf("squad claim: %+v err=%v", squadClaim, err)
	}
	if len(squadClaim.UserIDsA) != 4 || len(squadClaim.UserIDsB) != 4 {
		t.Fatalf("squad roster sizes: %d / %d", len(squadClaim.UserIDsA), len(squadClaim.UserIDsB))
	}
}

func TestMatchmakingV2ReleaseAndOriginalPriority(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	mode := "casual_solo"
	uA, uB := uuid.New(), uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, uA, uB) })

	ticketA, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{uA}, TeamSize: 1}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{uB}, TeamSize: 1}, now.Add(time.Second), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}
	claim, err := coord.ClaimTickets(ctx, mode, now.Add(2*time.Second), 15*time.Second, 20, 0)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}

	// Selective requeue: only A.
	if err := coord.ReleaseTeamClaim(ctx, claim, true, false, 30*time.Second); err != nil {
		t.Fatalf("release selective: %v", err)
	}
	restoredA, err := coord.GetTicketByUser(ctx, uA)
	if err != nil || restoredA == nil || restoredA.State != V2StateSearching {
		t.Fatalf("A should be searching: %+v err=%v", restoredA, err)
	}
	if restoredA.EnqueuedAtMs != ticketA.EnqueuedAtMs {
		t.Fatalf("priority A lost: got %d want %d", restoredA.EnqueuedAtMs, ticketA.EnqueuedAtMs)
	}
	restoredB, err := coord.GetTicketByUser(ctx, uB)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	if restoredB != nil {
		t.Fatalf("B should be discarded, got %+v", restoredB)
	}

	// Re-join B and claim again; full requeue preserves both priorities.
	ticketB2, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{uB}, TeamSize: 1}, now.Add(3*time.Second), 30*time.Second)
	if err != nil {
		t.Fatalf("rejoin B: %v", err)
	}
	// A still has original priority earlier than B2.
	_ = ticketB2
	claim2, err := coord.ClaimTickets(ctx, mode, now.Add(4*time.Second), 15*time.Second, 20, 0)
	if err != nil || claim2 == nil {
		t.Fatalf("claim2: %v %+v", err, claim2)
	}
	if err := coord.ReleaseTeamClaim(ctx, claim2, true, true, 30*time.Second); err != nil {
		t.Fatalf("release both: %v", err)
	}
	a2, _ := coord.GetTicketByUser(ctx, uA)
	b2, _ := coord.GetTicketByUser(ctx, uB)
	if a2 == nil || b2 == nil {
		t.Fatalf("both should be restored: A=%+v B=%+v", a2, b2)
	}
	// A still holds original enqueue from first join (priority preserved across claim/release).
	if a2.EnqueuedAtMs != ticketA.EnqueuedAtMs {
		t.Fatalf("A priority after second release: got %d want %d", a2.EnqueuedAtMs, ticketA.EnqueuedAtMs)
	}
}

func TestMatchmakingV2DurableFirstRecoveryPrimitives(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	mode := "casual_solo"
	uA, uB := uuid.New(), uuid.New()
	t.Cleanup(func() {
		cleanupV2Mode(ctx, client, mode, uA, uB)
		_ = client.Del(ctx, matchmakingV2ClaimsIndexKey()).Err()
	})

	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{uA}, TeamSize: 1}, now, 30*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{uB}, TeamSize: 1}, now.Add(time.Millisecond), 30*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}

	// Short claim TTL so recover_after is soon in the past relative to later now.
	claim, err := coord.ClaimTickets(ctx, mode, now.Add(time.Second), time.Millisecond, 20, 0)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}

	// Claimed tickets survive searching-lease style renew past original search lease.
	for _, u := range []uuid.UUID{uA, uB} {
		ticket, renewErr := coord.RenewTicketLease(ctx, u, 30*time.Second, now.Add(time.Minute))
		if renewErr != nil || ticket == nil || ticket.State != V2StateClaimed || ticket.ClaimID != claim.ClaimID {
			t.Fatalf("claimed player %s lost: %+v err=%v", u, ticket, renewErr)
		}
	}

	// Claim recovery record has no Redis TTL (durable until finalize/release).
	ttl, err := client.PTTL(ctx, matchmakingV2ClaimKey(claim.ClaimID)).Result()
	if err != nil {
		t.Fatalf("claim TTL: %v", err)
	}
	if ttl != -1 {
		t.Fatalf("claim recovery record TTL = %v, want persistent until finalize/release", ttl)
	}

	// List expired claims — service layer checks PostgreSQL first, then either finalize or release.
	ids, err := coord.ListExpiredTeamClaims(ctx, now.Add(5*time.Second), 20)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(ids) < 1 {
		t.Fatalf("expired claims = %d, want >=1", len(ids))
	}
	found := false
	for _, id := range ids {
		if id == claim.ClaimID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("claim %s not in expired list %v", claim.ClaimID, ids)
	}

	// Durable-first path when no formation exists: requeue both.
	loaded, err := coord.GetTeamClaim(ctx, claim.ClaimID)
	if err != nil || loaded == nil {
		t.Fatalf("get claim: %v %+v", err, loaded)
	}
	if err := coord.ReleaseTeamClaim(ctx, loaded, true, true, 30*time.Second); err != nil {
		t.Fatalf("release after durable miss: %v", err)
	}
	for _, u := range []uuid.UUID{uA, uB} {
		ticket, getErr := coord.GetTicketByUser(ctx, u)
		if getErr != nil || ticket == nil || ticket.State != V2StateSearching {
			t.Fatalf("after recovery %s: %+v err=%v", u, ticket, getErr)
		}
	}

	// DropClaimIndex is safe for missing claim hashes.
	if err := coord.DropTeamClaimIndex(ctx, claim.ClaimID); err != nil {
		t.Fatalf("drop index: %v", err)
	}
}

func TestMatchmakingV2ClaimPrunesStale(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	mode := "casual_solo"
	stale, freshA, freshB := uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, stale, freshA, freshB) })

	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{stale}, TeamSize: 1}, now, 5*time.Second); err != nil {
		t.Fatalf("join stale: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{freshA}, TeamSize: 1}, now.Add(time.Second), 60*time.Second); err != nil {
		t.Fatalf("join A: %v", err)
	}
	if _, err := coord.JoinTicket(ctx, JoinTicketInput{Mode: mode, UserIDs: []uuid.UUID{freshB}, TeamSize: 1}, now.Add(2*time.Second), 60*time.Second); err != nil {
		t.Fatalf("join B: %v", err)
	}

	claim, err := coord.ClaimTickets(ctx, mode, now.Add(10*time.Second), 15*time.Second, 20, 0)
	if err != nil || claim == nil {
		t.Fatalf("claim: %v %+v", err, claim)
	}
	if containsUUID(claim.UserIDsA, stale) || containsUUID(claim.UserIDsB, stale) {
		t.Fatalf("stale user claimed: %+v", claim)
	}
}

func TestMatchmakingV2LeaveSearching(t *testing.T) {
	client := testRedisV2(t)
	ctx := context.Background()
	coord := NewMatchmakingV2Coordinator(client)
	now := time.Now().UTC()
	a, b := uuid.New(), uuid.New()
	mode := "casual_duo"
	t.Cleanup(func() { cleanupV2Mode(ctx, client, mode, a, b) })

	ticket, err := coord.JoinTicket(ctx, JoinTicketInput{
		Mode: mode, UserIDs: []uuid.UUID{a, b}, PartyID: uuid.New(), PartyVersion: 1, TeamSize: 2,
	}, now, 30*time.Second)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if err := coord.LeaveTicket(ctx, a); err != nil {
		t.Fatalf("leave: %v", err)
	}
	// Whole roster cleared.
	for _, u := range []uuid.UUID{a, b} {
		got, getErr := coord.GetTicketByUser(ctx, u)
		if getErr != nil {
			t.Fatalf("get %s: %v", u, getErr)
		}
		if got != nil {
			t.Fatalf("expected nil after leave, got %+v", got)
		}
	}
	// Ticket hash gone.
	gone, err := coord.GetTicket(ctx, ticket.TicketID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if gone != nil {
		t.Fatalf("ticket should be deleted: %+v", gone)
	}
	// Idempotent leave.
	if err := coord.LeaveTicket(ctx, a); err != nil {
		t.Fatalf("second leave: %v", err)
	}
}

func sameUUIDSet(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[uuid.UUID]int, len(a))
	for _, id := range a {
		counts[id]++
	}
	for _, id := range b {
		counts[id]--
		if counts[id] < 0 {
			return false
		}
	}
	return true
}

func disjointUUIDSets(a, b []uuid.UUID) bool {
	set := make(map[uuid.UUID]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; ok {
			return false
		}
	}
	return true
}

func containsUUID(list []uuid.UUID, target uuid.UUID) bool {
	for _, id := range list {
		if id == target {
			return true
		}
	}
	return false
}
