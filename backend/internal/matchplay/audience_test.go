package matchplay_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/realtime"
)

func TestResolveMarkerAudienceIsTeamOnly(t *testing.T) {
	t.Parallel()
	aud := matchplay.ResolveMarkerAudience(1)
	if aud.Kind != realtime.AudienceTeam || aud.TeamSlot == nil || *aud.TeamSlot != 1 {
		t.Fatalf("audience = %+v", aud)
	}
	aud2 := matchplay.ResolveMarkerAudience(2)
	if aud2.TeamSlot == nil || *aud2.TeamSlot != 2 {
		t.Fatalf("audience2 = %+v", aud2)
	}
	// Invalid slot must not broadcast to all participants.
	bad := matchplay.ResolveMarkerAudience(0)
	if bad.Kind == realtime.AudienceAllParticipants {
		t.Fatal("invalid team must not use all_participants")
	}
}

func TestResolveTeamAudienceExcludingMutes(t *testing.T) {
	t.Parallel()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	aud := matchplay.ResolveTeamAudienceExcludingMutes(1, []uuid.UUID{a, b, c}, []uuid.UUID{b})
	if aud.Kind != realtime.AudienceUsers {
		t.Fatalf("kind = %s", aud.Kind)
	}
	if len(aud.UserIDs) != 2 {
		t.Fatalf("users = %v", aud.UserIDs)
	}
	for _, id := range aud.UserIDs {
		if id == b {
			t.Fatal("muted user should be excluded")
		}
	}
}

func TestResolveViewAudienceNeverBroadcasts(t *testing.T) {
	t.Parallel()
	empty := matchplay.ResolveViewAudience(nil)
	if empty.Kind != realtime.AudienceUsers || len(empty.UserIDs) != 0 {
		t.Fatalf("empty view audience = %+v", empty)
	}
	u := uuid.New()
	aud := matchplay.ResolveViewAudience([]uuid.UUID{u})
	if aud.Kind != realtime.AudienceUsers || len(aud.UserIDs) != 1 || aud.UserIDs[0] != u {
		t.Fatalf("view audience = %+v", aud)
	}
}

func TestTeammateUserIDs(t *testing.T) {
	t.Parallel()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	parts := []matchplay.MatchParticipant{
		{UserID: u1, TeamSlot: 1},
		{UserID: u2, TeamSlot: 1},
		{UserID: u3, TeamSlot: 2},
	}
	got := matchplay.TeammateUserIDs(parts, 1)
	if len(got) != 2 {
		t.Fatalf("teammates = %v", got)
	}
	if matchplay.TeamSlotOf(parts, u3) != 2 {
		t.Fatal("slot of u3")
	}
}
