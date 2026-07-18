package matchplay

import (
	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

// AudienceKind mirrors realtime audience kinds for matchplay callers.
const (
	AudienceAllParticipants = realtime.AudienceAllParticipants
	AudienceTeam            = realtime.AudienceTeam
	AudienceUsers           = realtime.AudienceUsers
)

// ResolveMatchLifecycleAudience returns the audience for match lifecycle events
// (started, completed, round started/ended/revealed, forfeit). All participants.
func ResolveMatchLifecycleAudience() realtime.Audience {
	return realtime.Audience{Kind: AudienceAllParticipants}
}

// ResolveTeamAudience returns a same-team audience for markers and team chat.
// teamSlot must be 1 or 2; invalid slots fall back to empty user list (no delivery).
func ResolveTeamAudience(teamSlot int) realtime.Audience {
	if teamSlot != 1 && teamSlot != 2 {
		return realtime.Audience{Kind: AudienceUsers, UserIDs: nil}
	}
	slot := teamSlot
	return realtime.Audience{Kind: AudienceTeam, TeamSlot: &slot}
}

// ResolveTeamAudienceExcludingMutes returns team recipients minus muted authors.
// When mutedUserIDs is empty this equals ResolveTeamAudience. When non-empty,
// delivery is narrowed to explicit UserIDs of the remaining teammates so the
// hub can filter per-connection without embedding mute lists in payloads.
func ResolveTeamAudienceExcludingMutes(teamSlot int, teammateIDs, mutedAuthorIDs []uuid.UUID) realtime.Audience {
	if len(mutedAuthorIDs) == 0 {
		return ResolveTeamAudience(teamSlot)
	}
	muted := make(map[uuid.UUID]struct{}, len(mutedAuthorIDs))
	for _, id := range mutedAuthorIDs {
		if id != uuid.Nil {
			muted[id] = struct{}{}
		}
	}
	// Recipient-side mute filtering: exclude muted authors from the fanout when
	// the event itself is authored by a muted user. Callers pass the author as
	// mutedAuthorIDs when the recipient has muted them — for broadcast of a
	// single author message, pass that author and the full teammate list; the
	// publisher should invoke this per-recipient or use Users audience.
	out := make([]uuid.UUID, 0, len(teammateIDs))
	for _, id := range teammateIDs {
		if id == uuid.Nil {
			continue
		}
		if _, skip := muted[id]; skip {
			continue
		}
		out = append(out, id)
	}
	return realtime.Audience{Kind: AudienceUsers, UserIDs: out}
}

// ResolveMarkerAudience is team-only (markers never cross the opposing team).
func ResolveMarkerAudience(teamSlot int) realtime.Audience {
	return ResolveTeamAudience(teamSlot)
}

// ResolveChatAudience is team-only with optional mute exclusions for a message author.
// teammates should be active same-team user IDs; mutedByRecipient maps recipient -> authors they muted
// is complex — for a single message from author, pass recipients who have not muted the author.
func ResolveChatAudience(teamSlot int, eligibleRecipients []uuid.UUID) realtime.Audience {
	if len(eligibleRecipients) == 0 {
		return ResolveTeamAudience(teamSlot)
	}
	return realtime.Audience{Kind: AudienceUsers, UserIDs: append([]uuid.UUID(nil), eligibleRecipients...)}
}

// ResolveViewAudience delivers provider-safe view state only to authorized spectators.
// Callers supply the currently allowed spectator user IDs from snapshot policy.
func ResolveViewAudience(spectatorUserIDs []uuid.UUID) realtime.Audience {
	if len(spectatorUserIDs) == 0 {
		return realtime.Audience{Kind: AudienceUsers, UserIDs: nil}
	}
	return realtime.Audience{Kind: AudienceUsers, UserIDs: append([]uuid.UUID(nil), spectatorUserIDs...)}
}

// TeammateUserIDs returns user IDs on the given team slot from a participant roster.
func TeammateUserIDs(participants []MatchParticipant, teamSlot int) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		if p.TeamSlot == teamSlot && p.UserID != uuid.Nil {
			out = append(out, p.UserID)
		}
	}
	return out
}

// ParticipantUserIDs returns all participant user IDs.
func ParticipantUserIDs(participants []MatchParticipant) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		if p.UserID != uuid.Nil {
			out = append(out, p.UserID)
		}
	}
	return out
}

// TeamSlotOf returns the team slot for userID, or 0 when not found.
func TeamSlotOf(participants []MatchParticipant, userID uuid.UUID) int {
	for _, p := range participants {
		if p.UserID == userID {
			return p.TeamSlot
		}
	}
	return 0
}
