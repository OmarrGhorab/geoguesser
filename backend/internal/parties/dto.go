package parties

import (
	"time"

	"github.com/google/uuid"
)

// CreatePartyRequest is the body for POST /parties.
type CreatePartyRequest struct {
	Format string `json:"format"`
}

// InviteRequest is the body for POST /parties/{partyId}/invites.
type InviteRequest struct {
	UserID string `json:"user_id"`
}

// ReadinessRequest is the body for PUT /parties/{partyId}/readiness/me.
type ReadinessRequest struct {
	Ready bool `json:"ready"`
}

// PartyMemberDTO is one roster entry in a party response.
type PartyMemberDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	Ready       bool      `json:"ready"`
	IsLeader    bool      `json:"is_leader"`
	JoinedAt    time.Time `json:"joined_at"`
}

// PartyDTO is the public party projection.
type PartyDTO struct {
	ID            uuid.UUID        `json:"id"`
	Format        string           `json:"format"`
	Capacity      int              `json:"capacity"`
	Status        string           `json:"status"`
	Version       int              `json:"version"`
	LeaderUserID  uuid.UUID        `json:"leader_user_id"`
	ActiveMatchID *uuid.UUID       `json:"active_match_id"`
	Members       []PartyMemberDTO `json:"members"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// PartyResponse wraps a party resource.
type PartyResponse struct {
	Party PartyDTO `json:"party"`
}

// CurrentPartyResponse returns the caller's party or null.
type CurrentPartyResponse struct {
	Party *PartyDTO `json:"party"`
}

// PartyInviteDTO is a pending invitation as seen by the invitee.
type PartyInviteDTO struct {
	ID        uuid.UUID `json:"id"`
	PartyID   uuid.UUID `json:"party_id"`
	Format    string    `json:"format"`
	Capacity  int       `json:"capacity"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Inviter   struct {
		UserID      uuid.UUID `json:"user_id"`
		DisplayName string    `json:"display_name"`
		AvatarURL   *string   `json:"avatar_url"`
	} `json:"inviter"`
}

// PartyInviteResponse wraps a created invite.
type PartyInviteResponse struct {
	Invite PartyInviteDTO `json:"invite"`
}

// PartyInviteListResponse lists pending incoming invites.
type PartyInviteListResponse struct {
	Invites []PartyInviteDTO `json:"invites"`
}

func toPartyDTO(snap PartySnapshot) PartyDTO {
	members := make([]PartyMemberDTO, 0, len(snap.Members))
	for _, m := range snap.Members {
		if m.Member.Status != MemberStatusActive {
			continue
		}
		members = append(members, PartyMemberDTO{
			UserID:      m.Member.UserID,
			DisplayName: m.Profile.DisplayName,
			AvatarURL:   m.Profile.AvatarURL,
			Ready:       m.Member.Ready,
			IsLeader:    m.Member.UserID == snap.Party.LeaderUserID,
			JoinedAt:    m.Member.JoinedAt.UTC(),
		})
	}
	return PartyDTO{
		ID:            snap.Party.ID,
		Format:        snap.Party.Format,
		Capacity:      int(snap.Party.Capacity),
		Status:        snap.Party.Status,
		Version:       snap.Party.Version,
		LeaderUserID:  snap.Party.LeaderUserID,
		ActiveMatchID: snap.Party.ActiveMatchID,
		Members:       members,
		CreatedAt:     snap.Party.CreatedAt.UTC(),
		UpdatedAt:     snap.Party.UpdatedAt.UTC(),
	}
}

func toInviteDTO(snap InviteSnapshot) PartyInviteDTO {
	dto := PartyInviteDTO{
		ID:        snap.Invite.ID,
		PartyID:   snap.Invite.PartyID,
		Format:    snap.Format,
		Capacity:  int(snap.Capacity),
		ExpiresAt: snap.Invite.ExpiresAt.UTC(),
		CreatedAt: snap.Invite.CreatedAt.UTC(),
	}
	dto.Inviter.UserID = snap.Inviter.UserID
	dto.Inviter.DisplayName = snap.Inviter.DisplayName
	dto.Inviter.AvatarURL = snap.Inviter.AvatarURL
	return dto
}
