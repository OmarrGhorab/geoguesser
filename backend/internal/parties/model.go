package parties

import (
	"time"

	"github.com/google/uuid"
)

// Party format constants (durable parties are Duo/Squad only; Solo has no party row).
const (
	FormatDuo   = "duo"
	FormatSquad = "squad"
)

// Party status lifecycle.
const (
	StatusForming = "forming"
	StatusQueued  = "queued"
	StatusInMatch = "in_match"
	StatusClosed  = "closed"
)

// Member status values.
const (
	MemberStatusActive = "active"
	MemberStatusLeft   = "left"
	MemberStatusKicked = "kicked"
)

// Invite status values.
const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusDeclined = "declined"
	InviteStatusExpired  = "expired"
	InviteStatusRevoked  = "revoked"
)

// CapacityForFormat returns the roster capacity for a party format.
func CapacityForFormat(format string) (int, bool) {
	switch format {
	case FormatDuo:
		return 2, true
	case FormatSquad:
		return 4, true
	default:
		return 0, false
	}
}

// Party is a durable Duo/Squad premade group.
type Party struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Format        string     `gorm:"column:format;not null"`
	Capacity      int16      `gorm:"column:capacity;not null"`
	LeaderUserID  uuid.UUID  `gorm:"column:leader_user_id;type:uuid;not null"`
	Status        string     `gorm:"column:status;not null"`
	Version       int        `gorm:"column:version;not null"`
	ActiveMatchID *uuid.UUID `gorm:"column:active_match_id;type:uuid"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;not null"`
	ClosedAt      *time.Time `gorm:"column:closed_at"`
}

// TableName returns the parties table name.
func (Party) TableName() string { return "parties" }

// PartyMember is a membership row on a party.
type PartyMember struct {
	PartyID  uuid.UUID  `gorm:"column:party_id;type:uuid;primaryKey"`
	UserID   uuid.UUID  `gorm:"column:user_id;type:uuid;primaryKey"`
	Status   string     `gorm:"column:status;not null"`
	Ready    bool       `gorm:"column:ready;not null"`
	JoinedAt time.Time  `gorm:"column:joined_at;not null"`
	LeftAt   *time.Time `gorm:"column:left_at"`
}

// TableName returns the party_members table name.
func (PartyMember) TableName() string { return "party_members" }

// PartyInvite is a time-limited invitation to join a forming party.
type PartyInvite struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	PartyID       uuid.UUID  `gorm:"column:party_id;type:uuid;not null"`
	InviterUserID uuid.UUID  `gorm:"column:inviter_user_id;type:uuid;not null"`
	InviteeUserID uuid.UUID  `gorm:"column:invitee_user_id;type:uuid;not null"`
	Status        string     `gorm:"column:status;not null"`
	ExpiresAt     time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
	RespondedAt   *time.Time `gorm:"column:responded_at"`
}

// TableName returns the party_invites table name.
func (PartyInvite) TableName() string { return "party_invites" }

// PublicProfile is a privacy-safe profile projection for party surfaces.
type PublicProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CountryCode *string
}

// MemberSnapshot is an active (or recently loaded) member with profile fields.
type MemberSnapshot struct {
	Member  PartyMember
	Profile PublicProfile
}

// PartySnapshot is a versioned party view with active members and profiles.
type PartySnapshot struct {
	Party   Party
	Members []MemberSnapshot
}

// InviteSnapshot is a pending invite with inviter profile and party format facts.
type InviteSnapshot struct {
	Invite   PartyInvite
	Inviter  PublicProfile
	Format   string
	Capacity int16
}

// ActiveMemberCount returns the number of active members in the snapshot.
func (s PartySnapshot) ActiveMemberCount() int {
	n := 0
	for _, m := range s.Members {
		if m.Member.Status == MemberStatusActive {
			n++
		}
	}
	return n
}

// AllReady reports whether every active member is ready.
func (s PartySnapshot) AllReady() bool {
	if len(s.Members) == 0 {
		return false
	}
	for _, m := range s.Members {
		if m.Member.Status != MemberStatusActive {
			continue
		}
		if !m.Member.Ready {
			return false
		}
	}
	return s.ActiveMemberCount() > 0
}

// ActiveUserIDs returns active member user IDs in joined order.
func (s PartySnapshot) ActiveUserIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(s.Members))
	for _, m := range s.Members {
		if m.Member.Status == MemberStatusActive {
			ids = append(ids, m.Member.UserID)
		}
	}
	return ids
}

// IsComplete reports whether the party has a full roster.
func (s PartySnapshot) IsComplete() bool {
	return s.ActiveMemberCount() == int(s.Party.Capacity)
}
